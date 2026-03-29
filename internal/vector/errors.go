package vector

import "errors"

// ErrClosed is returned when the index was closed or is not usable (e.g. after Stack.Close).
var ErrClosed = errors.New("vector index closed")
