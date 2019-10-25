package filter

import (
	"context"
	"fmt"

	"github.com/qri-io/value"
)

func newStream(in interface{}) (res *valueStream, err error) {
	res = &valueStream{}

	switch v := in.(type) {
	case *valueStream:
		res.val = v
		return res, err
	case []interface{}:
		res.vals = v
		return res, err
	case nil, bool, byte, int, float64, string, []byte, map[interface{}]interface{}, map[string]interface{}:
		res.val = in
		return res, err
	}

	// TODO (b5) - handle link

	// if vs, ok := in.(vals.ValueStream); ok {
	// 	res.val = vs
	// 	return res, err
	// }

	// if kvs, ok := in.(vals.KeyValueStream); ok {
	// 	res.val = kvs
	// 	return res, err
	// }

	return nil, fmt.Errorf("unrecognized type: %T", in)
}

type valueStream struct {
	i    int
	done bool
	// only one of val, vals, wrap will be set
	val  interface{}
	vals []interface{}
}

// var _ vals.ValueStream = (*valueStream)(nil)

func (it *valueStream) Next(v *interface{}) (more bool) {
	if it.val == nil && it.vals == nil {
		return false
	}

	if it.val != nil {
		*v = it.val
		it.val = nil
		return true
	}

	if it.i == len(it.vals) || it.done {
		return false
	}

	*v = it.vals[it.i]
	it.i++
	return true
}

func (it *valueStream) Close() error {
	it.done = true
	return nil
}

func (it *valueStream) ValueForIndex(i int) (v interface{}, err error) {
	return it.vals[i], nil
}

func applyToStream(ctx context.Context, r value.Resolver, vs *valueStream, f filter) (res interface{}, err error) {
	var vals []interface{}
	var v interface{}
	for vs.Next(&v) {
		if v, err = f.apply(ctx, r, v); err != nil {
			return res, err
		}
		vals = append(vals, v)
	}
	return vals, nil
}

// type keyValueStream struct {
// 	i    int
// 	done bool
// 	vals []struct {
// 		key string
// 		val interface{}
// 	}
// }

// var _ vals.KeyValueStream = (*keyValueStream)(nil)

// func (it *keyValueStream) Next(key *string, v *interface{}) (more bool) {
// 	defer func() { it.i++ }()
// 	if it.i == len(it.vals) || it.done {
// 		return false
// 	}

// 	*key = it.vals[it.i].key
// 	*v = it.vals[it.i].val
// 	return true
// }

// func (it *keyValueStream) Close() error {
// 	it.done = true
// 	return nil
// }

// func (it *keyValueStream) MapIndex(key string) (v interface{}, err error) {
// 	return nil, fmt.Errorf("mapInded of keyvalue", a ...interface{})
// 	// return it.v.Index(i).Interface(), nil
// }
