package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDefinePath_WithDestinationLocationRole_IsEchoedInResponse proves
// the field round-trips through the REST surface: request -> use case ->
// aggregate -> response, matching this fleet's convention of never
// silently dropping an accepted request field.
func TestDefinePath_WithDestinationLocationRole_IsEchoedInResponse(t *testing.T) {
	router := newTestServer(t)
	body := `{"pathId":"PACK","matchPrefix":"pack","direct":true,"requiredCapabilities":["pack"],"destinationLocationRole":"Drop"}`
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
	if resp["destinationLocationRole"] != "Drop" {
		t.Fatalf("want destinationLocationRole Drop, got %+v", resp)
	}
}

// TestDefinePath_NoDestinationLocationRole_OmitsFieldFromResponse proves
// the field is omitted entirely (not present with an empty-string value)
// when the caller never declared one -- most paths, and the default
// established before this feature existed.
func TestDefinePath_NoDestinationLocationRole_OmitsFieldFromResponse(t *testing.T) {
	router := newTestServer(t)
	body := `{"pathId":"PICK","matchPrefix":"pick","direct":true,"requiredCapabilities":["pick"]}`
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
	if _, present := resp["destinationLocationRole"]; present {
		t.Fatalf("expected destinationLocationRole to be omitted, got %+v", resp)
	}
}

// TestDefinePath_UnrecognizedDestinationLocationRole_Returns422 proves an
// unrecognized value (including a REAL facility-layout LocationRole this
// service does not treat as a valid destination, e.g. Storage) is
// rejected with the fleet-standard RFC 7807 shape, not silently accepted
// or defaulted.
func TestDefinePath_UnrecognizedDestinationLocationRole_Returns422(t *testing.T) {
	router := newTestServer(t)
	body := `{"pathId":"PICK","matchPrefix":"pick","direct":true,"requiredCapabilities":["pick"],"destinationLocationRole":"Storage"}`
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

// TestRevisePath_DoesNotAcceptOrAlterDestinationLocationRole proves the
// revise request body has no way to change the destination role even if
// a caller sends one -- reviseProcessPathRequest deliberately has no such
// field, mirroring the same immutability the aggregate itself enforces
// for Direct.
func TestRevisePath_DoesNotAcceptOrAlterDestinationLocationRole(t *testing.T) {
	router := newTestServer(t)
	defineBody := `{"pathId":"PACK","matchPrefix":"pack","direct":true,"requiredCapabilities":["pack"],"destinationLocationRole":"WorkCenter"}`
	defineReq := httptest.NewRequest(http.MethodPost, "/process-paths", bytes.NewBufferString(defineBody))
	defineRR := httptest.NewRecorder()
	router.ServeHTTP(defineRR, defineReq)
	if defineRR.Code != http.StatusCreated {
		t.Fatalf("setup: want 201, got %d", defineRR.Code)
	}

	// The revise DTO has no destinationLocationRole field at all, so an
	// attempt to sneak one in via extra JSON is simply ignored by the
	// decoder -- the field remains WorkCenter regardless.
	reviseBody := `{"matchPrefix":"pack-v2","requiredCapabilities":["pack"],"destinationLocationRole":"Shipping"}`
	reviseReq := httptest.NewRequest(http.MethodPut, "/process-paths/PACK", bytes.NewBufferString(reviseBody))
	reviseRR := httptest.NewRecorder()
	router.ServeHTTP(reviseRR, reviseReq)
	if reviseRR.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", reviseRR.Code, reviseRR.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(reviseRR.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["destinationLocationRole"] != "WorkCenter" {
		t.Fatalf("expected destinationLocationRole to remain WorkCenter (revise cannot change it), got %+v", resp)
	}
}
