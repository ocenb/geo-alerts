package errs

import "errors"

var (
	ErrIncidentExists   = errors.New("incident already exists")
	ErrIncidentNotFound = errors.New("incident not found")
)
