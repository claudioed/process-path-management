// Package main_test contains the BDD / acceptance test suite: godog
// (Cucumber for Go) drives the Gherkin scenarios under features/ against
// the real chi router over HTTP, wired to the same in-memory adapters the
// service's own httptest suite uses (see
// internal/adapters/inbound/http/server_test.go's newTestServer helper).
// It is a black-box test — it only ever touches the REST API.
package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"

	inboundhttp "github.com/claudioed/process-path-management/internal/adapters/inbound/http"
	"github.com/claudioed/process-path-management/internal/adapters/outbound/memory"
	"github.com/claudioed/process-path-management/internal/application/usecases"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// bddClock is a fixed clock, so response bodies are deterministic across
// scenario runs.
type bddClock struct{ t time.Time }

func (c bddClock) Now() time.Time { return c.t }

// discardPublisher swallows every event -- these scenarios assert on HTTP
// responses, not on what got published.
type discardPublisher struct{}

func (discardPublisher) Publish(context.Context, shared.DomainEvent) error { return nil }

// newServer builds the production router over fresh in-memory adapters and
// serves it from an httptest server, mirroring the wiring in
// internal/adapters/inbound/http/server_test.go's newTestServer.
func newServer() *httptest.Server {
	repo := memory.NewProcessPathRepo()
	clock := bddClock{time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)}
	pub := &discardPublisher{}

	s := &inboundhttp.Server{
		DefinePath:     &usecases.DefinePath{Repo: repo, Publisher: pub, Clock: clock},
		RevisePath:     &usecases.RevisePath{Repo: repo, Publisher: pub, Clock: clock},
		DeactivatePath: &usecases.DeactivatePath{Repo: repo, Publisher: pub, Clock: clock},
		GetPath:        &usecases.GetPath{Repo: repo},
		ListPaths:      &usecases.ListPaths{Repo: repo},
	}

	return httptest.NewServer(inboundhttp.NewRouter(s, nil, ""))
}

// world is the per-scenario state: one server with its own in-memory
// adapters, plus the last HTTP response the steps made.
type world struct {
	server *httptest.Server

	lastStatus int
	lastBody   []byte
}

func (w *world) reset() {
	if w.server != nil {
		w.server.Close()
	}
	w.server = newServer()
	w.lastStatus = 0
	w.lastBody = nil
}

func (w *world) close() {
	if w.server != nil {
		w.server.Close()
		w.server = nil
	}
}

// do performs a real net/http call against the httptest server and records
// the response as the "last" one for the assertion steps.
func (w *world) do(method, path string, body any) error {
	var reader io.Reader = http.NoBody
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, w.server.URL+path, reader)
	if err != nil {
		return fmt.Errorf("build %s %s: %w", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.server.Client().Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read %s %s response: %w", method, path, err)
	}

	w.lastStatus = resp.StatusCode
	w.lastBody = raw
	return nil
}

func (w *world) decodeLastObject() (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal(w.lastBody, &out); err != nil {
		return nil, fmt.Errorf("decode response %q: %w", string(w.lastBody), err)
	}
	return out, nil
}

func (w *world) decodeLastArray() ([]any, error) {
	var out []any
	if err := json.Unmarshal(w.lastBody, &out); err != nil {
		return nil, fmt.Errorf("decode response %q: %w", string(w.lastBody), err)
	}
	return out, nil
}

func splitCapabilities(csv string) []string {
	parts := strings.Split(csv, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// --- Given/When steps ----------------------------------------------------

func (w *world) serviceIsRunning() error {
	if err := w.do(http.MethodGet, "/healthz", nil); err != nil {
		return err
	}
	if w.lastStatus != http.StatusOK {
		return fmt.Errorf("healthz returned %d, want 200", w.lastStatus)
	}
	return nil
}

func (w *world) definePath(pathId, matchPrefix, capabilitiesCSV string) error {
	return w.do(http.MethodPost, "/process-paths", map[string]any{
		"pathId":               pathId,
		"matchPrefix":          matchPrefix,
		"direct":               true,
		"requiredCapabilities": splitCapabilities(capabilitiesCSV),
	})
}

func (w *world) pathAlreadyDefined(pathId, matchPrefix, capabilitiesCSV string) error {
	if err := w.definePath(pathId, matchPrefix, capabilitiesCSV); err != nil {
		return err
	}
	if w.lastStatus != http.StatusCreated {
		return fmt.Errorf("setup define %s: want 201, got %d: %s", pathId, w.lastStatus, string(w.lastBody))
	}
	return nil
}

func (w *world) getPath(pathId string) error {
	return w.do(http.MethodGet, "/process-paths/"+pathId, nil)
}

func (w *world) listPaths() error {
	return w.do(http.MethodGet, "/process-paths", nil)
}

func (w *world) listAllPaths() error {
	return w.do(http.MethodGet, "/process-paths?all=true", nil)
}

func (w *world) revisePath(pathId, matchPrefix, capabilitiesCSV string) error {
	return w.do(http.MethodPut, "/process-paths/"+pathId, map[string]any{
		"matchPrefix":          matchPrefix,
		"requiredCapabilities": splitCapabilities(capabilitiesCSV),
	})
}

func (w *world) deactivatePath(pathId string) error {
	return w.do(http.MethodDelete, "/process-paths/"+pathId, nil)
}

func (w *world) pathIsDeactivated(pathId string) error {
	if err := w.deactivatePath(pathId); err != nil {
		return err
	}
	if w.lastStatus != http.StatusNoContent {
		return fmt.Errorf("setup deactivate %s: want 204, got %d: %s", pathId, w.lastStatus, string(w.lastBody))
	}
	return nil
}

// --- Then steps ------------------------------------------------------------

func (w *world) requestAccepted(status int) error {
	if w.lastStatus != status {
		return fmt.Errorf("got status %d, want %d: %s", w.lastStatus, status, string(w.lastBody))
	}
	return nil
}

func (w *world) processPathResponseReports(pathId, matchPrefix, status string) error {
	obj, err := w.decodeLastObject()
	if err != nil {
		return err
	}
	if got, _ := obj["pathId"].(string); got != pathId {
		return fmt.Errorf("got pathId %q, want %q", got, pathId)
	}
	if got, _ := obj["matchPrefix"].(string); got != matchPrefix {
		return fmt.Errorf("got matchPrefix %q, want %q", got, matchPrefix)
	}
	if got, _ := obj["status"].(string); got != status {
		return fmt.Errorf("got status %q, want %q", got, status)
	}
	return nil
}

func (w *world) processPathListReports(count int) error {
	arr, err := w.decodeLastArray()
	if err != nil {
		return err
	}
	if len(arr) != count {
		return fmt.Errorf("got %d paths, want %d: %+v", len(arr), count, arr)
	}
	return nil
}

func (w *world) pathNowHasStatus(pathId, status string) error {
	if err := w.getPath(pathId); err != nil {
		return err
	}
	if w.lastStatus != http.StatusOK {
		return fmt.Errorf("verify get %s: want 200, got %d: %s", pathId, w.lastStatus, string(w.lastBody))
	}
	obj, err := w.decodeLastObject()
	if err != nil {
		return err
	}
	if got, _ := obj["status"].(string); got != status {
		return fmt.Errorf("got status %q, want %q", got, status)
	}
	return nil
}

// InitializeScenario registers every step definition and gives each
// scenario a fresh server over fresh in-memory adapters, so scenarios are
// independent.
func InitializeScenario(sc *godog.ScenarioContext) {
	w := &world{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		w.reset()
		return ctx, nil
	})
	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		w.close()
		return ctx, nil
	})

	sc.Step(`^the Process Path Management service is running$`, w.serviceIsRunning)

	sc.Step(`^a process path "([^"]*)" is already defined with matchPrefix "([^"]*)" and capabilities "([^"]*)"$`, w.pathAlreadyDefined)
	sc.Step(`^a process path "([^"]*)" is defined with matchPrefix "([^"]*)" and capabilities "([^"]*)"$`, w.definePath)
	sc.Step(`^the process path "([^"]*)" is requested$`, w.getPath)
	sc.Step(`^the process paths are listed$`, w.listPaths)
	sc.Step(`^all process paths are listed including deactivated ones$`, w.listAllPaths)
	sc.Step(`^the process path "([^"]*)" is revised with matchPrefix "([^"]*)" and capabilities "([^"]*)"$`, w.revisePath)
	sc.Step(`^the process path "([^"]*)" is already deactivated$`, w.pathIsDeactivated)
	sc.Step(`^the process path "([^"]*)" is deactivated$`, w.deactivatePath)

	sc.Step(`^the request is accepted with status (\d+)$`, w.requestAccepted)
	sc.Step(`^the request is rejected with status (\d+)$`, w.requestAccepted)
	sc.Step(`^the deactivation is accepted with status (\d+)$`, w.requestAccepted)
	sc.Step(`^the deactivation is rejected with status (\d+)$`, w.requestAccepted)
	sc.Step(`^the process path response reports pathId "([^"]*)", matchPrefix "([^"]*)", and status "([^"]*)"$`, w.processPathResponseReports)
	sc.Step(`^the process path list reports (\d+) path$`, w.processPathListReports)
	sc.Step(`^the process path list reports (\d+) paths$`, w.processPathListReports)
	sc.Step(`^the process path "([^"]*)" now has status "([^"]*)"$`, w.pathNowHasStatus)
}

// TestFeatures runs the Gherkin acceptance suite under features/.
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format: "pretty",
			Paths:  []string{"features"},
			// Strict makes an undefined or pending step fail the suite
			// instead of silently skipping it.
			Strict:   true,
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
