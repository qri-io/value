package value

import (
	"context"
)

// Resolver returns a value for a link
type Resolver interface {
	Get(ctx context.Context, path string) (res interface{}, err error)
}

// Link is an interface for fetching the value of a link
type Link interface {
	Path() string
}

// Map is an associative array complex type
type Map interface {
	ValueForKey(key interface{}) (val interface{}, err error)
	Iterate() Iterator
}

// Array is an ordered value sequence complex type
type Array interface {
	Iterate() Iterator
}

// Iterator provides a sequence of values to the caller
// Use Next to advance the sequence cursor. Callers must call close when
// finished.
// The caller must call Close when the iterator is no longer needed
// Operations that modify a sequence will fail if it has active iterators.
type Iterator interface {
	Next() bool
	Scan(dest interface{}) error
	Key() interface{}
	Close() error

	// IsOrdered returns true if the iterator returns advances deterministically
	IsOrdered() bool
}
