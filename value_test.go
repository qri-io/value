package value

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIsValue(t *testing.T) {
	cases := []struct {
		v      Value
		expect bool
	}{
		// scalar
		{nil, true},
		{0, true},
		{0x0, true},
		{int64(0), false},
		{float32(0), false},
		{struct{}{}, false},

		// compound
		{map[string]interface{}{}, true},
		{[]interface{}{}, true},
		{map[string]string{}, false},
		{[]struct{}{}, false},

		// complex
		{NewLink("/foo"), true},
		{NewIterator(nil), true},
		{ioutil.NopCloser(&bytes.Buffer{}), true},
	}

	for i, c := range cases {
		got := IsValue(c.v)
		if c.expect != got {
			t.Errorf("case %d input: %#v expected %t", i, c.v, c.expect)
		}
	}
}

func TestLink(t *testing.T) {
	// create a link
	l := NewLink("path")
	p := l.Path()
	if "path" != p {
		t.Errorf("path mismatch. want: %s, got: %s", "path", p)
	}

	// check value before resolving
	v, resolved := l.Value()
	if resolved {
		t.Errorf("expected resolved to be false")
	}
	if nil != v {
		t.Errorf("expected v to be nil. got: %#v", v)
	}

	// resolve the link
	rTo := "this link resolved to a string"
	l.Resolved(rTo)

	v, resolved = l.Value()
	if !resolved {
		t.Errorf("expected resolved to be true")
	}
	if diff := cmp.Diff(rTo, v); diff != "" {
		t.Errorf("result mismatch (-want, +got):\n%s", diff)
	}
}

func TestIterator(t *testing.T) {
	values := []Value{nil, "hello", 2, float64(3), false}
	it := NewIterator(values)
	defer it.Close()

	if !it.IsOrdered() {
		t.Errorf("expected iterator to be ordered")
	}

	var v Value
	i := 0
	for it.Next() {
		k := it.Key()
		if key, ok := k.(int); ok {
			if key != i {
				t.Errorf("key index mismatch. expected %d, got %d", i, key)
			}
		} else {
			t.Errorf("key returned wrong kind. expected int, got: %#v", k)
		}

		if err := it.Scan(v); err == nil {
			t.Errorf("expected error passing non-pointer")
		}

		if err := it.Scan(&v); err != nil {
			t.Error(err)
		}
		expect := values[i]
		if diff := cmp.Diff(expect, v); diff != "" {
			t.Errorf("iteration %d mismatch (-want +got):\n%s", i, diff)
		}
		i++
	}
}
