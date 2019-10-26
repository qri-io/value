package value

import (
	"context"
	"fmt"
	"io"
	"reflect"
)

// Value ("qri value") is the set of all types in the qri runtime
// Anything within the qri runtime that works with a qri value must handle
// any values,
// Values fall into one of three categories:
// Scalar: irreducible values
//   * nil
//   * uint8 (aka "byte")
//   * int
//   * float
//   * bool
//   * []byte
//   * string
// Compound: compositions of other values
//   * array
//   * map
// Complex: values defined by behaviour
//   * Link
//   * Array
//   * Map
//   * Iterator
//   * ByteReader
//
// Complex values are created through interface satisfaction. All complex values
// are also compound values
//
// The Value type provides no compiler or runtime guarantees, and only serves as
// documentation. This is not idiomatic go, and our choice to do this is
// intentional.
// The go language as presently designed doesn't encourage this kind of thinking
// Which is fine, but our code should note where we're trying to build a
// dynamic type system in a language that doesn't encourage such a thing.
type Value = interface{}

// Resolver is an interface for retrieving the value a link points to
// Resolver is not a value, it's an interface that link values depend on
type Resolver interface {
	Get(ctx context.Context, path string) (res Value, err error)
}

// Link is a complex value that points at the address of another value
// Links retain a pointer to cache the value they refer to, forming a
// singleton complex value: a compound type of only one value
type Link interface {
	Path() string

	// Value returns the value a link points to, and a flag for if a link has been
	// marked as resolved.
	// Value is a cache of link content, not a method for fetching.
	// basically, don't fetch links when Value is called, instead use a resolver
	// to fetch the value of the link, then call Resolved on the link
	Value() (value Value, resolved bool)
	// SetResolved caches the value of the link on the link itself
	Resolved(value Value)
}

// Link represents a link to something
type link struct {
	path     string
	value    Value
	resolved bool
}

// NewLink constructs a new link
func NewLink(path string) Link {
	return &link{
		path: path,
	}
}

// Path is the path this link points to
func (l link) Path() string { return string(l.path) }

func (l link) Value() (value Value, resolved bool) {
	return l.value, l.resolved
}

func (l *link) Resolved(v Value) {
	l.value = v
	l.resolved = true
}

// Map is an associative array value defined through behaviour
// Map is the complex version of a map
type Map interface {
	ValueForKey(key interface{}) (val Value, err error)
	Iterate() Iterator
}

// Array is an ordered set of values defined through behaviour
// Array is the complex version of an array
type Array interface {
	Iterate() Iterator
}

// ByteReader is an alias for an io.ReadCloser
type ByteReader = io.ReadCloser

// Iterator provides a sequence of values to the caller
// Use Next to advance the sequence cursor. Callers must call close when
// finished.
// The caller must call Close when the iterator is no longer needed
// Operations that modify a sequence will fail if it has active iterators.
type Iterator interface {
	Next() bool
	Scan(dest Value) error
	Key() interface{}
	Close() error

	// IsOrdered returns true if the iterator returns advances deterministically
	IsOrdered() bool
}

// iterator is a generic iterator value
type iterator struct {
	i      int
	values []interface{}
}

// NewIterator creates an iterator
func NewIterator(values []Value) Iterator {
	return &iterator{
		i:      -1,
		values: values,
	}
}

// Next advances the iterator, returning false if no iterations remain
func (it *iterator) Next() bool {
	if it.i >= len(it.values)-1 {
		return false
	}
	it.i++
	return true
}

// Scan reads the current iteration value into dest
func (it *iterator) Scan(dest Value) error {
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.CanSet() {
		return fmt.Errorf("expected pointer value for scan")
	}

	if it.values[it.i] == nil {
		v.Set(reflect.Zero(v.Type()))
	} else {
		v.Set(reflect.ValueOf(it.values[it.i]))
	}
	return nil
}

// Key returns the current iteration key
func (it *iterator) Key() Value { return it.i }

// Close terminates the iterator, releasing any associated resources
func (it *iterator) Close() error { return nil }

// IsOrdered returns true if the iterator returns advances deterministically
func (it *iterator) IsOrdered() bool { return true }

// IsValue returns true if v is a qri value
// Checking IsValue is relatively expensive. Avoid using IsValue in complied
// code, and instead use IsValue in tests
func IsValue(v interface{}) bool {
	switch v.(type) {
	// scalar values
	case nil, uint8, int, float64, bool, []byte, string:
		return true
		// compound values
	case []interface{}, map[string]interface{}, map[interface{}]interface{}:
		return true
	}

	// complex values
	if _, ok := v.(Link); ok {
		return true
	}
	if _, ok := v.(Map); ok {
		return true
	}
	if _, ok := v.(Array); ok {
		return true
	}
	if _, ok := v.(Iterator); ok {
		return true
	}
	if _, ok := v.(ByteReader); ok {
		return true
	}

	return false
}
