package shared_test

import (
	"errors"
	"testing"

	"github.com/claudioed/process-path-management/internal/domain/shared"
)

func TestParseDestinationLocationRole_Empty_ReturnsUnset(t *testing.T) {
	role, err := shared.ParseDestinationLocationRole("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != shared.DestinationLocationRoleUnset {
		t.Fatalf("want DestinationLocationRoleUnset, got %q", role)
	}
}

func TestParseDestinationLocationRole_RecognizedValues(t *testing.T) {
	for _, want := range []shared.DestinationLocationRole{
		shared.DestinationLocationRoleDrop,
		shared.DestinationLocationRoleWorkCenter,
		shared.DestinationLocationRoleShipping,
	} {
		got, err := shared.ParseDestinationLocationRole(string(want))
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", want, err)
		}
		if got != want {
			t.Fatalf("want %q, got %q", want, got)
		}
	}
}

func TestParseDestinationLocationRole_RejectsUnrecognizedValue(t *testing.T) {
	// Storage is a real facility-layout LocationRole, but not one this
	// service recognizes as a valid DESTINATION for a process path's
	// completed work -- it must still be rejected, not silently accepted.
	for _, bad := range []string{"Storage", "storage", "drop", "WORKCENTER", "not-a-role"} {
		if _, err := shared.ParseDestinationLocationRole(bad); !errors.Is(err, shared.ErrInvalidDestinationLocationRole) {
			t.Fatalf("input %q: want ErrInvalidDestinationLocationRole, got %v", bad, err)
		}
	}
}
