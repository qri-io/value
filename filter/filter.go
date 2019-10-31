package filter

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/qri-io/value"
)

// Filter applies piplelined transformations to an input values
type Filter struct {
	src      string
	resolver value.Resolver
}

// New creates a new Filter
func New(filterStr string, resolver value.Resolver) *Filter {
	return &Filter{
		src:      filterStr,
		resolver: resolver,
	}
}

// Apply executes a filter string against a given source, returning a filtered result
func (filt *Filter) Apply(ctx context.Context, source interface{}) (val interface{}, err error) {
	// fmt.Printf("parse %s\n", filterStr)
	r := strings.NewReader(filt.src)
	p := parser{s: newScanner(r)}
	filters, err := p.filters()
	if err != nil {
		return nil, err
	}

	val = source
	for _, f := range filters {
		// fmt.Printf("run filter: %#v\n", f)
		if val, err = f.apply(ctx, filt.resolver, val); err != nil {
			// panic(err)
			return val, err
		}
		// fmt.Printf("result: %#v\n", val)
	}

	return unpackValueStreams(val)
}

type filter interface {
	apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error)
}

func unpackValueStreams(in interface{}) (val interface{}, err error) {
	if vs, ok := in.(*valueStream); ok {
		vals := []interface{}{}
		var v interface{}
		for vs.Next(&v) {
			if val, err = unpackValueStreams(v); err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}
		return vals, nil
	}

	return in, nil
}

type fStringLiteral string

func (f fStringLiteral) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {
	if v, ok := in.(*valueStream); ok {
		return applyToStream(ctx, r, v, f)
	}
	return string(f), nil
}

type fNumericLiteral float64

func (f fNumericLiteral) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {
	return f, nil
}

type fLength byte

func (f fLength) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {

	if it, ok := in.(value.Iterator); ok {
		i := 0
		for it.Next() {
			i++
		}
		return i, it.Close()
	}

	switch v := in.(type) {
	case *valueStream:
		return applyToStream(ctx, r, v, f)
	case string:
		return len(v), nil
	case []byte:
		return len(v), nil
	case map[interface{}]interface{}:
		return len(v), nil
	case map[string]interface{}:
		return len(v), nil
	case []interface{}:
		return len(v), nil

	case nil, bool, byte, int, float64:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected type: %T", in)
	}
}

type selector interface {
	filter
	isSelector()
}

type fSelector []selector

func (f fSelector) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {
	out = in
	for _, sel := range f {
		out, err = sel.apply(ctx, r, out)
		if err != nil {
			return out, err
		}
	}
	return out, err
}

// fIdentity is the identity filter, it returns whatever it's given
type fIdentity byte

func (f fIdentity) isSelector() {}

func (f fIdentity) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {
	return in, nil
}

type fKeySelector string

func (f fKeySelector) isSelector() {}

func (f fKeySelector) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {

	if link, ok := in.(value.Link); ok {
		in, err = r.Resolve(ctx, link)
		if err != nil {
			return nil, err
		}
	}

	if m, ok := in.(value.Map); ok {
		v, _ := m.ValueForKey(string(f))
		return v, nil
	}

	switch v := in.(type) {
	case *valueStream:
		return applyToStream(ctx, r, v, f)
	case map[interface{}]interface{}:
		return v[string(f)], err
	case map[string]interface{}:
		return v[string(f)], err
	case []interface{}:
		res := make([]interface{}, len(v))
		for i, d := range v {
			res[i], err = f.apply(ctx, r, d)
			if err != nil {
				return nil, err
			}
		}
		return res, nil

	case nil, bool, byte, int, float64, string, []byte:
		// TODO (b5) - should we error here?
		return nil, nil
	}

	// if vr, ok := in.(vals.ValueStream); ok {
	// 	vals := []interface{}{}
	// 	var v interface{}
	// 	for vr.Next(&v) {
	// 		val, err := f.apply(v)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		vals = append(vals, val)
	// 	}
	// 	return vals, nil
	// }

	// if kvs, ok := in.(vals.KeyValueStream); ok {
	// 	var v interface{}
	// 	var key string
	// 	s := string(f)
	// 	for kvs.Next(&key, &v) {
	// 		if key == s {
	// 			return v, kvs.Close()
	// 		}
	// 	}
	// 	return nil, nil
	// }

	// if keyable, ok := in.(vals.Keyable); ok {
	// 	return keyable.MapIndex(string(f)), nil
	// }

	return nil, fmt.Errorf("unexpected type: %T", in)
}

type fIndexSelector int

func (f fIndexSelector) isSelector() {}

func (f fIndexSelector) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {

	if link, ok := in.(value.Link); ok {
		in, err = r.Resolve(ctx, link)
		if err != nil {
			return nil, err
		}
	}

	if it, ok := in.(value.Iterator); ok {
		i := 0
		for it.Next() {
			if i == int(f) {
				if err = it.Scan(&out); err != nil {
					return nil, err
				}
				break
			}
			i++
		}
		return out, it.Close()
	}

	switch v := in.(type) {
	case *valueStream:
		return applyToStream(ctx, r, v, f)
	case string:
		return v[int(f)], nil
	case []byte:
		return v[int(f)], nil
	case []interface{}:
		return v[int(f)], nil

	case nil, bool, byte, int, float64, map[string]interface{}, map[interface{}]interface{}:
		// TODO (b5) - should we error here?
		return nil, nil
	}

	// if vr, ok := in.(vals.ValueStream); ok {
	// 	var v interface{}

	// 	i := 0
	// 	for vr.Next(&v) {
	// 		if i == int(f) {
	// 			return v, nil
	// 		}
	// 		i++
	// 	}
	// 	return nil, nil
	// }

	// TODO (b5) - what do about a KeyValueStream here?
	// also, ordered KeyValueStream? Too much?

	return nil, fmt.Errorf("unexpected type: %T", in)
}

type fIterateAllSeletor bool

func (f fIterateAllSeletor) isSelector() {}

func (f fIterateAllSeletor) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {
	if link, ok := in.(value.Link); ok {
		in, err = r.Resolve(ctx, link)
		if err != nil {
			return nil, err
		}
	}

	if it, ok := in.(value.Iterator); ok {
		return it, nil
	}

	return newStream(in)
}

type fIndexRangeSelector struct {
	start int
	stop  int
	all   bool
}

func (f *fIndexRangeSelector) isSelector() {}

func (f *fIndexRangeSelector) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {
	if link, ok := in.(value.Link); ok {
		in, err = r.Resolve(ctx, link)
		if err != nil {
			return nil, err
		}
	}

	if it, ok := in.(value.Iterator); ok {
		res := []interface{}{}
		offset := f.start
		limit := f.stop
		for it.Next() {
			if offset--; offset > 0 {
				continue
			}
			var v interface{}
			if err = it.Scan(&v); err != nil {
				return nil, err
			}
			res = append(res, v)
			if limit--; limit == 0 {
				break
			}
		}
		return res, it.Close()
	}

	if rdr, ok := in.(io.ReadCloser); ok {
		defer rdr.Close()

		var buf []byte
		if f.start > 0 {
			buf = make([]byte, f.start)
			if _, err = rdr.Read(buf); err != nil {
				return nil, err
			}
		}
		if f.stop == 0 {
			return ioutil.ReadAll(rdr)
		}
		buf = make([]byte, f.stop)
		_, err = rdr.Read(buf)
		return buf, err
	}

	switch v := in.(type) {
	case *valueStream:
		return applyToStream(ctx, r, v, f)
	case string:
		if f.all {
			return v, nil
		}
		return v[f.start:f.stop], nil
	case []byte:
		if f.all {
			return v, nil
		}
		return v[f.start:f.stop], nil
	case []interface{}:
		if f.all {
			return v, nil
		}
		if f.stop == 0 {
			return v[f.start:], nil
		}
		return v[f.start:f.stop], nil

	case nil, bool, byte, int, float64, map[string]interface{}, map[interface{}]interface{}:
		// TODO (b5) - should we error here?
		return nil, nil
	}

	return nil, fmt.Errorf("unexpected type: %T", in)
}

type fBinaryOp struct {
	left  filter
	op    tokenType
	right filter
}

func (f fBinaryOp) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {
	left, err := f.left.apply(ctx, r, in)
	if err != nil {
		return nil, err
	}
	left, lk := normalizeValue(left)

	right, err := f.right.apply(ctx, r, in)
	if err != nil {
		return nil, err
	}
	right, rk := normalizeValue(right)

	switch f.op {
	case tStar:
		if lk == reflect.Float64 && rk == reflect.Float64 {
			return left.(float64) * right.(float64), nil
		}
	case tPlus:
		if lk == reflect.Float64 && rk == reflect.Float64 {
			return left.(float64) + right.(float64), nil
		}
	}

	return nil, fmt.Errorf("binary operations are not finished cannot %#v %s %#v", left, f.op, right)
}

func normalizeValue(in interface{}) (out interface{}, rk reflect.Kind) {
	if nl, ok := in.(fNumericLiteral); ok {
		return float64(nl), reflect.Float64
	} else if sl, ok := in.(fStringLiteral); ok {
		return string(sl), reflect.String
	}

	rk = reflect.TypeOf(in).Kind()
	switch rk {
	case reflect.Int:
		return float64(in.(int)), reflect.Float64
	case reflect.Float64:
		return in, rk
	}

	return in, rk
}

// fSlics is a group of filters
type fSlice []filter

func (fSlice) isSelector() {}

func (f fSlice) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {
	if link, ok := in.(value.Link); ok {
		in, err = r.Resolve(ctx, link)
		if err != nil {
			return nil, err
		}
	}

	if v, ok := in.(*valueStream); ok {
		return applyToStream(ctx, r, v, f)
	}

	vals := make([]interface{}, len(f))
	for i, fi := range f {
		if vals[i], err = fi.apply(ctx, r, in); err != nil {
			return nil, err
		}
	}
	return vals, nil
}

type fObjectMapping map[string]filter

func (f fObjectMapping) apply(ctx context.Context, r value.Resolver, in interface{}) (out interface{}, err error) {
	if v, ok := in.(*valueStream); ok {
		return applyToStream(ctx, r, v, f)
	}

	vals := map[string]interface{}{}
	for key, f := range f {
		if vals[key], err = f.apply(ctx, r, in); err != nil {
			return nil, err
		}
	}
	return vals, nil
}
