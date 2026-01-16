package errs

import "errors"

var (
	ErrCacheMiss = errors.New("key does not exist")
)
