import { useState, type FormEvent } from "react";
import { Card, DataTable, StatusPill, useFetch } from "@warehouse/ui-kit";
import { apiPost, apiPut, apiDelete, ApiError } from "../api";
import { PROCESS_PATH_API_BASE } from "../config";
import type { ProcessPath } from "../types";
import {
  CheckboxField,
  FormRow,
  InlineError,
  InlineSuccess,
  SubmitButton,
  TextField,
} from "../components/formkit";

/** Parses a comma-separated capability list into a trimmed, non-empty
 *  string array -- the form's one text input for what the DTO models as
 *  requiredCapabilities: []string. */
function parseCapabilities(raw: string): string[] {
  return raw
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

/**
 * The single screen for this remote: process-path-management is a flat
 * ProcessPath resource (no sub-hierarchy the way facility-layout's
 * Site->Zone->Aisle chain is), so one Card+DataTable pair covers the
 * whole use-case surface -- DefinePath (POST), RevisePath (PUT, inline
 * edit per row), DeactivatePath (DELETE, per-row action). Deactivation is
 * a soft delete: the row stays visible with status DEACTIVATED rather
 * than disappearing, matching the aggregate's own append-only lifecycle
 * (see this repo's CLAUDE.md).
 */
export function ProcessPathsScreen() {
  const [refreshKey, setRefreshKey] = useState(0);
  const {
    data: paths,
    loading,
    error: listError,
  } = useFetch<ProcessPath[]>(`${PROCESS_PATH_API_BASE}/process-paths?_r=${refreshKey}`);

  // Define form state
  const [pathId, setPathId] = useState("");
  const [matchPrefix, setMatchPrefix] = useState("");
  const [direct, setDirect] = useState(true);
  const [capabilities, setCapabilities] = useState("");
  const [defineError, setDefineError] = useState<string | null>(null);
  const [defineSuccess, setDefineSuccess] = useState<string | null>(null);
  const [defining, setDefining] = useState(false);

  // Inline revise state: at most one row editable at a time, keyed by pathId.
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editMatchPrefix, setEditMatchPrefix] = useState("");
  const [editCapabilities, setEditCapabilities] = useState("");
  const [reviseError, setReviseError] = useState<string | null>(null);
  const [revising, setRevising] = useState(false);

  const [deactivatingId, setDeactivatingId] = useState<string | null>(null);
  const [deactivateError, setDeactivateError] = useState<string | null>(null);

  async function onDefine(e: FormEvent) {
    e.preventDefault();
    setDefineError(null);
    setDefineSuccess(null);
    setDefining(true);
    try {
      await apiPost<ProcessPath>("/process-paths", {
        pathId: pathId.trim(),
        matchPrefix: matchPrefix.trim(),
        direct,
        requiredCapabilities: parseCapabilities(capabilities),
      });
      setDefineSuccess(`Process path ${pathId.trim()} defined.`);
      setPathId("");
      setMatchPrefix("");
      setDirect(true);
      setCapabilities("");
      setRefreshKey((k) => k + 1);
    } catch (err) {
      setDefineError(err instanceof ApiError ? err.message : "Failed to define process path.");
    } finally {
      setDefining(false);
    }
  }

  function startEdit(p: ProcessPath) {
    setEditingId(p.pathId);
    setEditMatchPrefix(p.matchPrefix);
    setEditCapabilities(p.requiredCapabilities.join(", "));
    setReviseError(null);
  }

  function cancelEdit() {
    setEditingId(null);
    setReviseError(null);
  }

  async function onRevise(p: ProcessPath) {
    setReviseError(null);
    setRevising(true);
    try {
      await apiPut<ProcessPath>(`/process-paths/${encodeURIComponent(p.pathId)}`, {
        matchPrefix: editMatchPrefix.trim(),
        requiredCapabilities: parseCapabilities(editCapabilities),
      });
      setEditingId(null);
      setRefreshKey((k) => k + 1);
    } catch (err) {
      setReviseError(err instanceof ApiError ? err.message : "Failed to revise process path.");
    } finally {
      setRevising(false);
    }
  }

  async function onDeactivate(p: ProcessPath) {
    setDeactivateError(null);
    setDeactivatingId(p.pathId);
    try {
      await apiDelete(`/process-paths/${encodeURIComponent(p.pathId)}`);
      setRefreshKey((k) => k + 1);
    } catch (err) {
      setDeactivateError(
        err instanceof ApiError ? err.message : "Failed to deactivate process path.",
      );
    } finally {
      setDeactivatingId(null);
    }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "var(--wh-space-5)" }}>
      <div>
        <h1 style={{ fontSize: "var(--wh-font-size-2xl)", margin: 0 }}>Process paths</h1>
        <p style={{ color: "var(--wh-color-text-muted)", marginTop: 4 }}>
          process-path-management · the fleet&apos;s declared process-path catalogue
        </p>
      </div>

      <Card title="Define a process path">
        <form onSubmit={onDefine} style={{ display: "flex", flexDirection: "column", gap: "var(--wh-space-3)" }}>
          <FormRow>
            <TextField
              label="Path ID"
              value={pathId}
              onChange={setPathId}
              placeholder="PICK"
              required
            />
            <TextField
              label="Match prefix"
              value={matchPrefix}
              onChange={setMatchPrefix}
              placeholder="pick"
              required
            />
            <TextField
              label="Required capabilities (comma-separated)"
              value={capabilities}
              onChange={setCapabilities}
              placeholder="pick, pick-heavy"
              required
            />
            <CheckboxField label="Direct" checked={direct} onChange={setDirect} />
            <SubmitButton
              disabled={defining || !pathId.trim() || !matchPrefix.trim() || !capabilities.trim()}
            >
              {defining ? "Defining…" : "Define path"}
            </SubmitButton>
          </FormRow>
          <InlineError message={defineError} />
          <InlineSuccess message={defineSuccess} />
        </form>
      </Card>

      <Card title="Declared paths">
        {listError && <InlineError message={listError.message} />}
        {deactivateError && <InlineError message={deactivateError} />}
        <DataTable
          rowKey={(p) => p.pathId}
          rows={paths ?? []}
          loading={loading}
          emptyState={<span>No process paths defined yet.</span>}
          columns={[
            { key: "pathId", header: "Path ID", render: (p) => p.pathId },
            {
              key: "matchPrefix",
              header: "Match prefix",
              render: (p) =>
                editingId === p.pathId ? (
                  <TextField
                    label=""
                    value={editMatchPrefix}
                    onChange={setEditMatchPrefix}
                    disabled={p.status === "DEACTIVATED"}
                  />
                ) : (
                  p.matchPrefix
                ),
            },
            { key: "direct", header: "Direct", render: (p) => (p.direct ? "Yes" : "No") },
            {
              key: "requiredCapabilities",
              header: "Required capabilities",
              render: (p) =>
                editingId === p.pathId ? (
                  <TextField
                    label=""
                    value={editCapabilities}
                    onChange={setEditCapabilities}
                    disabled={p.status === "DEACTIVATED"}
                  />
                ) : (
                  p.requiredCapabilities.join(", ")
                ),
            },
            { key: "status", header: "Status", render: (p) => <StatusPill status={p.status} size="sm" /> },
            {
              key: "actions",
              header: "Actions",
              render: (p) => {
                if (p.status === "DEACTIVATED") return null;
                if (editingId === p.pathId) {
                  return (
                    <div style={{ display: "flex", gap: "var(--wh-space-2)" }}>
                      <SubmitButton
                        type="button"
                        onClick={() => void onRevise(p)}
                        disabled={revising || !editMatchPrefix.trim() || !editCapabilities.trim()}
                      >
                        {revising ? "Saving…" : "Save"}
                      </SubmitButton>
                      <SubmitButton type="button" onClick={cancelEdit} disabled={revising}>
                        Cancel
                      </SubmitButton>
                    </div>
                  );
                }
                return (
                  <div style={{ display: "flex", gap: "var(--wh-space-2)" }}>
                    <SubmitButton type="button" onClick={() => startEdit(p)}>
                      Revise
                    </SubmitButton>
                    <SubmitButton
                      type="button"
                      tone="danger"
                      onClick={() => void onDeactivate(p)}
                      disabled={deactivatingId === p.pathId}
                    >
                      {deactivatingId === p.pathId ? "Deactivating…" : "Deactivate"}
                    </SubmitButton>
                  </div>
                );
              },
            },
          ]}
        />
        <InlineError message={reviseError} />
      </Card>
    </div>
  );
}
