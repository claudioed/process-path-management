package http

import (
	"errors"
	"net/http"

	"github.com/claudioed/process-path-management/internal/application/usecases"
	"github.com/claudioed/process-path-management/internal/domain/processpath"
)

// statusFor maps a typed domain/application error to an HTTP status code.
func statusFor(err error) int {
	switch {
	case errors.Is(err, usecases.ErrPathNotFound):
		return http.StatusNotFound

	case errors.Is(err, usecases.ErrPathAlreadyExists):
		return http.StatusConflict

	case errors.Is(err, processpath.ErrPathDeactivated),
		errors.Is(err, processpath.ErrEmptyMatchPrefix),
		errors.Is(err, processpath.ErrMatchPrefixNotLowercase),
		errors.Is(err, processpath.ErrNoRequiredCapabilities):
		return http.StatusUnprocessableEntity

	default:
		return http.StatusInternalServerError
	}
}

// problemBaseURI is the namespace for this service's RFC 7807 "type" URIs.
// It does not need to resolve to a real page — it's an identifier, unique
// per distinct error category in this service.
const problemBaseURI = "https://errors.process-path-management.warehouse-systems.dev/"

// problemInfo is the fixed, category-level (type, title) pair for an RFC
// 7807 problem response. slug becomes the last path segment of "type";
// title is a fixed human string for the category (the dynamic detail
// comes from err.Error() at write time, not from this table).
type problemInfo struct {
	slug  string
	title string
}

// problemFor maps a typed domain/application error to its RFC 7807
// (type, title) pair. Mirrors statusFor's error groupings one-for-one.
func problemFor(err error) problemInfo {
	switch {
	case errors.Is(err, usecases.ErrPathNotFound):
		return problemInfo{"path-not-found", "No process path exists with this id"}
	case errors.Is(err, usecases.ErrPathAlreadyExists):
		return problemInfo{"path-already-exists", "A process path with this id already exists (active or deactivated)"}
	case errors.Is(err, processpath.ErrPathDeactivated):
		return problemInfo{"path-deactivated", "This process path has been deactivated and can no longer be revised"}
	case errors.Is(err, processpath.ErrEmptyMatchPrefix):
		return problemInfo{"empty-match-prefix", "matchPrefix must not be empty"}
	case errors.Is(err, processpath.ErrMatchPrefixNotLowercase):
		return problemInfo{"match-prefix-not-lowercase", "matchPrefix must be lower-case"}
	case errors.Is(err, processpath.ErrNoRequiredCapabilities):
		return problemInfo{"no-required-capabilities", "requiredCapabilities must be non-empty"}
	default:
		return problemInfo{"internal-error", "An unexpected internal error occurred"}
	}
}
