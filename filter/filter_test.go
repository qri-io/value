package filter

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// d for "data", this quick test function makes for cleaner test writing
func d(in string) interface{} {
	var v interface{}
	if err := json.Unmarshal([]byte(in), &v); err != nil {
		panic(err)
	}
	return v
}

type goodCase struct {
	filter string
	source interface{}
	value  interface{}
}

func runGoodCases(t *testing.T, cases []goodCase) {
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s", c.filter), func(t *testing.T) {
			filt := New(c.filter, nil)
			got, err := filt.Apply(context.Background(), c.source)
			if err != nil {
				t.Fatalf("error: %s", err)
			}
			if diff := cmp.Diff(c.value, got); diff != "" {
				t.Errorf("\n%s\nvalue mismatch (-want +got):\n%s", c.filter, diff)
			}
		})
	}
}

func TestApply(t *testing.T) {
	cases := []goodCase{
		{".", d(`[{"a": "b"}]`), d(`[{"a": "b"}]`)},
		{`"swoosh"`, d(`{"a": "b"}`), d(`"swoosh"`)},
		{".apples", d(`[{"a": "b"}]`), d(`[null]`)},
		{".a", d(`[{"a":"b"}]`), d(`["b"]`)},
		{".bar", d(`[{"bar": "b", "baz": 10}]`), d(`["b"]`)},
		{".a.bar", d(`{"a": { "bar": "b", "bat": 0}}`), d(`"b"`)},
		// TODO (b5) -
		// {"[1]", []interface{}{"a", "b", "c"}, []interface{}{1}},

		{".[1]", []interface{}{"a", "b", "c"}, "b"},
		{".[0:2]", []interface{}{"a", "b", "c"}, []interface{}{"a", "b"}},
		{".bar[0:2]", map[string]interface{}{"bar": []interface{}{"a", "b", "c"}}, []interface{}{"a", "b"}},

		{".bar.a",
			map[string]interface{}{
				"bar": []interface{}{
					map[string]interface{}{"a": "a"},
					map[string]interface{}{"a": "b"},
					map[string]interface{}{"a": "c"}}}, []interface{}{"a", "b", "c"}},
		{".bar * 5", map[string]interface{}{"bar": 5}, float64(25)},

		// {"( .bar | length ) x 5", map[string]interface{}{ "bar": []string{"a","b","c"} }, 15},
	}

	runGoodCases(t, cases)
}

func TestPipe(t *testing.T) {
	cases := []goodCase{
		{".a | length", map[string]interface{}{"a": map[string]interface{}{"bar": "b", "baz": 0}}, 2},
	}

	runGoodCases(t, cases)
}

func TestIteration(t *testing.T) {
	cases := []goodCase{
		{".[:]", d(`["a","b","c"]`), []interface{}{"a", "b", "c"}},
		{`.[] | "swoosh"`, d(`[{"a": "b"}]`), d(`["swoosh"]`)},
		{`.[][]`, d(`["a"]`), d(`[["a"]]`)},
	}

	runGoodCases(t, cases)
}

func TestArrayMapping(t *testing.T) {
	cases := []goodCase{
		{`[.]`, d(`["a","b","c"]`), d(`[["a","b","c"]]`)},
		{"[ .foo, .bar ]", map[string]interface{}{"bar": "a", "foo": "b", "camp": "lucky"}, []interface{}{"b", "a"}},

		// TODO (b5) - implicit array mapping
		// {".foo, .bar", map[string]interface{}{"bar": "a", "foo": "b", "camp": "lucky"}, []interface{}{"b", "a"}},

		// TODO (b5) - current parser will choke on floating point literals in first position
		// {`[34.5, .]`, d("a"), d(`[34.5, "a"]`)},
	}

	runGoodCases(t, cases)
}

func TestObjectMapping(t *testing.T) {
	cases := []goodCase{
		{`{ foo: . }`, d(`["a","b","c"]`), d(`{ "foo": ["a","b","c"] }`)},
		{`{ foo: .[0], bar: .[1:] }`, d(`["a","b","c"]`), d(`{ "foo": "a", "bar": ["b","c"]}`)},
		{`.[] | {"value": .}`, d(`["a","b","c"]`), d(`[{"value":"a"},{"value":"b"},{"value":"c"}]`)},
	}

	runGoodCases(t, cases)
}

func TestLength(t *testing.T) {
	cases := []goodCase{
		{`length`, d(`"abcde"`), 5},
		{`length`, d(`[0,1,2,3,4]`), 5},
		{`length`, d(`{ "a": 0, "b": 1, "c": 2, "d": 3, "e": 4 }`), 5},
		{`length`, []byte{0, 1, 2, 3, 4}, 5},
		{`length`, map[interface{}]interface{}{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}, 5},
		{`.[] | length`, d(`["abcde", "fg"]`), []interface{}{5, 2}},
	}

	runGoodCases(t, cases)
}
