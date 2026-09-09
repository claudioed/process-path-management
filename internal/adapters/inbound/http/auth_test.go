package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/claudioed/process-path-management/internal/adapters/inbound/auth"
	inboundhttp "github.com/claudioed/process-path-management/internal/adapters/inbound/http"
	"github.com/claudioed/process-path-management/internal/adapters/outbound/memory"
	"github.com/claudioed/process-path-management/internal/application/usecases"
)

// newAuthedTestServer wires the same real router newTestServer does, but
// with the fleet REST auth middleware mounted in the given mode and two
// static keys: "r-key" (read) and "rw-key" (read-write).
func newAuthedTestServer(t *testing.T, mode auth.Mode) http.Handler {
	t.Helper()
	repo := memory.NewProcessPathRepo()
	clock := fixedClock{time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)}
	pub := &logOnlyPublisher{}

	server := &inboundhttp.Server{
		DefinePath:     &usecases.DefinePath{Repo: repo, Publisher: pub, Clock: clock},
		RevisePath:     &usecases.RevisePath{Repo: repo, Publisher: pub, Clock: clock},
		DeactivatePath: &usecases.DeactivatePath{Repo: repo, Publisher: pub, Clock: clock},
		GetPath:        &usecases.GetPath{Repo: repo},
		ListPaths:      &usecases.ListPaths{Repo: repo},
		Auth: &auth.Middleware{
			Authn: auth.NewStaticKeyAuth(map[string]auth.Scope{"r-key": auth.ScopeRead, "rw-key": auth.ScopeReadWrite}),
			Mode:  mode,
		},
	}
	return inboundhttp.NewRouter(server, nil, "")
}

func doAuthed(router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	var rdr *bytes.Buffer
	if body != "" {
		rdr = bytes.NewBufferString(body)
	} else {
		rdr = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

const defineBody = `{"pathId":"PICK","matchPrefix":"pick","direct":true,"requiredCapabilities":["pick"]}`

// TestAuth_EnforceTable is the fleet-mandated router table (rest-auth
// rollout brief step 7): one GET and one mutating route, each scope
// combination, and /healthz staying open.
func TestAuth_EnforceTable(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		token  string
		body   string
		want   int
	}{
		{"no token GET -> 401", http.MethodGet, "/process-paths", "", "", http.StatusUnauthorized},
		{"bad token GET -> 401", http.MethodGet, "/process-paths", "nope", "", http.StatusUnauthorized},
		{"read key GET -> 200", http.MethodGet, "/process-paths", "r-key", "", http.StatusOK},
		{"read key POST -> 403", http.MethodPost, "/process-paths", "r-key", defineBody, http.StatusForbidden},
		{"rw key POST -> 201", http.MethodPost, "/process-paths", "rw-key", defineBody, http.StatusCreated},
		{"rw key GET -> 200", http.MethodGet, "/process-paths", "rw-key", "", http.StatusOK},
		{"no token DELETE -> 401", http.MethodDelete, "/process-paths/PICK", "", "", http.StatusUnauthorized},
		{"healthz no token -> 200", http.MethodGet, "/healthz", "", "", http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			router := newAuthedTestServer(t, auth.ModeEnforce)
			rr := doAuthed(router, c.method, c.path, c.token, c.body)
			if rr.Code != c.want {
				t.Fatalf("want %d, got %d: %s", c.want, rr.Code, rr.Body.String())
			}
			if rr.Code == http.StatusUnauthorized && rr.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("401 must carry WWW-Authenticate")
			}
			if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
				if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
					t.Fatalf("want application/problem+json, got %q", ct)
				}
				var problem map[string]any
				if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
					t.Fatalf("problem body is not JSON: %v", err)
				}
				typ, _ := problem["type"].(string)
				if !bytes.HasPrefix([]byte(typ), []byte("https://errors.process-path-management.warehouse-systems.dev/")) {
					t.Fatalf("problem type must use this service's base, got %q", typ)
				}
				if problem["status"] != float64(c.want) || problem["instance"] != c.path {
					t.Fatalf("problem body mismatch: %v", problem)
				}
			}
		})
	}
}

// TestAuth_LogMode_LetsUnauthenticatedThrough covers the rollout gate: in
// "log" mode every request reaches the handler regardless of credential.
func TestAuth_LogMode_LetsUnauthenticatedThrough(t *testing.T) {
	router := newAuthedTestServer(t, auth.ModeLog)
	if rr := doAuthed(router, http.MethodPost, "/process-paths", "", defineBody); rr.Code != http.StatusCreated {
		t.Fatalf("log mode must pass an unauthenticated POST, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr := doAuthed(router, http.MethodGet, "/process-paths", "", ""); rr.Code != http.StatusOK {
		t.Fatalf("log mode must pass an unauthenticated GET, got %d", rr.Code)
	}
}

// TestAuth_OffMode_IsNoop pins the local-dev posture the composition root
// falls back to when no key is configured.
func TestAuth_OffMode_IsNoop(t *testing.T) {
	router := newAuthedTestServer(t, auth.ModeOff)
	if rr := doAuthed(router, http.MethodPost, "/process-paths", "", defineBody); rr.Code != http.StatusCreated {
		t.Fatalf("off mode must be a no-op, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAuth_NilMiddleware_KeepsRoutesOpen guards the contract the existing
// handler tests rely on: a Server with no Auth mounts nothing.
func TestAuth_NilMiddleware_KeepsRoutesOpen(t *testing.T) {
	router := newTestServer(t)
	if rr := doAuthed(router, http.MethodGet, "/process-paths", "", ""); rr.Code != http.StatusOK {
		t.Fatalf("nil Auth must leave routes open, got %d", rr.Code)
	}
}
