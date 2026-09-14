package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDefinePath_WithCycleTimeP95AndEligibility_IsEchoedInResponse proves
// the fulfillment capability contract (ADR 0010) fields round-trip
// through the REST surface.
func TestDefinePath_WithCycleTimeP95AndEligibility_IsEchoedInResponse(t *testing.T) {
	router := newTestServer(t)
	body := `{"pathId":"SINGLES","matchPrefix":"singles","direct":true,"requiredCapabilities":["pick"],"cycleTimeP95":"90m","eligibility":{"maxUnitsPerLine":1,"nonSortable":true,"requiredProductAttributes":["giftWrap"],"excludedProductAttributes":["hazmat"]}}`
	req := httptest.NewRequest(http.MethodPost, "/process-paths", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["cycleTimeP95"] != "1h30m0s" {
		t.Fatalf("want cycleTimeP95 1h30m0s, got %+v", resp)
	}
	elig, ok := resp["eligibility"].(map[string]any)
	if !ok {
		t.Fatalf("want eligibility object, got %+v", resp)
	}
	if elig["maxUnitsPerLine"] != float64(1) || elig["nonSortable"] != true {
		t.Fatalf("unexpected eligibility: %+v", elig)
	}
}

// TestDefinePath_NoEligibility_DefaultsToPermissiveEmptyObject proves an
// absent eligibility object round-trips as the permissive zero value
// (ADR 0010: "a permissive eligibility ({}) is valid").
func TestDefinePath_NoEligibility_DefaultsToPermissiveEmptyObject(t *testing.T) {
	router := newTestServer(t)
	body := `{"pathId":"PICK","matchPrefix":"pick","direct":true,"requiredCapabilities":["pick"],"cycleTimeP95":"2h"}`
	req := httptest.NewRequest(http.MethodPost, "/process-paths", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	elig, ok := resp["eligibility"].(map[string]any)
	if !ok {
		t.Fatalf("want eligibility object present (even if empty), got %+v", resp)
	}
	if len(elig) != 0 {
		t.Fatalf("want an empty eligibility object, got %+v", elig)
	}
}

// TestDefinePath_MissingCycleTimeP95_Returns422 proves cycleTimeP95 is
// required (ADR 0010: "required, positive").
func TestDefinePath_MissingCycleTimeP95_Returns422(t *testing.T) {
	router := newTestServer(t)
	body := `{"pathId":"PICK","matchPrefix":"pick","direct":true,"requiredCapabilities":["pick"]}`
	req := httptest.NewRequest(http.MethodPost, "/process-paths", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("want application/problem+json, got %s", ct)
	}
}

// TestDefinePath_ZeroCycleTimeP95_Returns422 proves a syntactically valid
// but zero/negative duration is still rejected as an invariant violation.
func TestDefinePath_ZeroCycleTimeP95_Returns422(t *testing.T) {
	router := newTestServer(t)
	body := `{"pathId":"PICK","matchPrefix":"pick","direct":true,"requiredCapabilities":["pick"],"cycleTimeP95":"0s"}`
	req := httptest.NewRequest(http.MethodPost, "/process-paths", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestRevisePath_UpdatesCycleTimeP95AndEligibility proves both fields
// are revisable (ADR 0010).
func TestRevisePath_UpdatesCycleTimeP95AndEligibility(t *testing.T) {
	router := newTestServer(t)
	defineBody := `{"pathId":"PICK","matchPrefix":"pick","direct":true,"requiredCapabilities":["pick"],"cycleTimeP95":"2h"}`
	defineReq := httptest.NewRequest(http.MethodPost, "/process-paths", bytes.NewBufferString(defineBody))
	defineRR := httptest.NewRecorder()
	router.ServeHTTP(defineRR, defineReq)
	if defineRR.Code != http.StatusCreated {
		t.Fatalf("setup: want 201, got %d", defineRR.Code)
	}

	reviseBody := `{"matchPrefix":"pick","requiredCapabilities":["pick"],"cycleTimeP95":"4h","eligibility":{"nonSortable":true}}`
	reviseReq := httptest.NewRequest(http.MethodPut, "/process-paths/PICK", bytes.NewBufferString(reviseBody))
	reviseRR := httptest.NewRecorder()
	router.ServeHTTP(reviseRR, reviseReq)
	if reviseRR.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", reviseRR.Code, reviseRR.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(reviseRR.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["cycleTimeP95"] != "4h0m0s" {
		t.Fatalf("want cycleTimeP95 4h0m0s, got %+v", resp)
	}
	elig, ok := resp["eligibility"].(map[string]any)
	if !ok || elig["nonSortable"] != true {
		t.Fatalf("want eligibility.nonSortable=true, got %+v", resp["eligibility"])
	}
}

// --- CPT schedule (ADR 0010) -----------------------------------------------

func definePathForSchedule(t *testing.T, router http.Handler, pathId string) {
	t.Helper()
	body := `{"pathId":"` + pathId + `","matchPrefix":"` + strings.ToLower(pathId) + `","direct":true,"requiredCapabilities":["pick"],"cycleTimeP95":"2h"}`
	req := httptest.NewRequest(http.MethodPost, "/process-paths", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("setup define %s: want 201, got %d: %s", pathId, rr.Code, rr.Body.String())
	}
}

func TestDefineCPTSchedule_ValidRequest_Returns200(t *testing.T) {
	router := newTestServer(t)
	definePathForSchedule(t, router, "PICK")

	body := `{"timezone":"America/Sao_Paulo","cutoffs":[{"cptId":"sp1-1500","localTime":"15:00","daysOfWeek":["Mon","Tue"],"shipMethod":"ground","eligiblePathIds":["PICK"]}]}`
	req := httptest.NewRequest(http.MethodPut, "/sites/sp1/cpt-schedule", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["siteId"] != "sp1" || resp["timezone"] != "America/Sao_Paulo" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestDefineCPTSchedule_IneligiblePathId_Returns422(t *testing.T) {
	router := newTestServer(t)
	// PICK is never defined.
	body := `{"timezone":"America/Sao_Paulo","cutoffs":[{"cptId":"sp1-1500","localTime":"15:00","daysOfWeek":["Mon"],"shipMethod":"ground","eligiblePathIds":["PICK"]}]}`
	req := httptest.NewRequest(http.MethodPut, "/sites/sp1/cpt-schedule", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("want application/problem+json, got %s", ct)
	}
}

func TestDefineCPTSchedule_InvalidTimezone_Returns422(t *testing.T) {
	router := newTestServer(t)
	definePathForSchedule(t, router, "PICK")

	body := `{"timezone":"Not/AZone","cutoffs":[{"cptId":"sp1-1500","localTime":"15:00","daysOfWeek":["Mon"],"shipMethod":"ground","eligiblePathIds":["PICK"]}]}`
	req := httptest.NewRequest(http.MethodPut, "/sites/sp1/cpt-schedule", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDefineCPTSchedule_NoCutoffs_Returns422(t *testing.T) {
	router := newTestServer(t)
	body := `{"timezone":"America/Sao_Paulo","cutoffs":[]}`
	req := httptest.NewRequest(http.MethodPut, "/sites/sp1/cpt-schedule", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetCPTSchedule_Missing_Returns404(t *testing.T) {
	router := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/sites/sp1/cpt-schedule", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
}

func TestGetCPTSchedule_Found_Returns200(t *testing.T) {
	router := newTestServer(t)
	definePathForSchedule(t, router, "PICK")

	defineBody := `{"timezone":"America/Sao_Paulo","cutoffs":[{"cptId":"sp1-1500","localTime":"15:00","daysOfWeek":["Mon"],"shipMethod":"ground","eligiblePathIds":["PICK"]}]}`
	defineReq := httptest.NewRequest(http.MethodPut, "/sites/sp1/cpt-schedule", bytes.NewBufferString(defineBody))
	defineRR := httptest.NewRecorder()
	router.ServeHTTP(defineRR, defineReq)
	if defineRR.Code != http.StatusOK {
		t.Fatalf("setup: want 200, got %d: %s", defineRR.Code, defineRR.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/sites/sp1/cpt-schedule", nil)
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", getRR.Code, getRR.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(getRR.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cutoffs, ok := resp["cutoffs"].([]any)
	if !ok || len(cutoffs) != 1 {
		t.Fatalf("want 1 cutoff, got %+v", resp)
	}
}
