import { describe, expect, it } from "vitest";
import { resolveProcessPathApiBase } from "./config";

describe("resolveProcessPathApiBase", () => {
  it("builds the production API base from the runtime API origin", () => {
    expect(resolveProcessPathApiBase({ apiOrigin: "http://localhost:8000" }, true)).toBe(
      "http://localhost:8000/api/process-path-management",
    );
  });

  it("normalizes a trailing slash on the runtime API origin", () => {
    expect(resolveProcessPathApiBase({ apiOrigin: "https://warehouse.example/" }, true)).toBe(
      "https://warehouse.example/api/process-path-management",
    );
  });

  it("fails loudly when production runtime configuration has no API origin", () => {
    expect(() => resolveProcessPathApiBase({}, true)).toThrow(
      "window.__WAREHOUSE_CONFIG__.apiOrigin is required in production",
    );
  });

  it("retains the existing standalone API origin in Vite development", () => {
    expect(resolveProcessPathApiBase({}, false)).toBe("http://localhost:8087");
  });
});
