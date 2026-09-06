package usecases

import "errors"

// ErrPathAlreadyExists is returned by DefinePath when the given PathId is
// already defined (active or deactivated) — a path's identity is
// permanent once created; a caller wanting to re-use an id after
// deactivation must be told explicitly, never silently overwritten.
var ErrPathAlreadyExists = errors.New("usecases: process path already exists")

// ErrPathNotFound is returned by any use case operating on a PathId that
// has never been defined.
var ErrPathNotFound = errors.New("usecases: process path not found")
