package jspointer_test

import (
	"encoding/json"
	"testing"

	"github.com/lestrrat/jspointer"
	"github.com/stretchr/testify/assert"
)

var src = `{
"foo": ["bar", "baz"],
"obj": { "a":1, "b":2, "c":[3,4], "d":[ {"e":9}, {"f":[50,51]} ] },
"": 0,
"a/b": 1,
"c%d": 2,
"e^f": 3,
"g|h": 4,
"i\\j": 5,
"k\"l": 6,
" ": 7,
"m~n": 8
}`
var target map[string]interface{}

func init() {
	if err := json.Unmarshal([]byte(src), &target); err != nil {
		panic(err)
	}
}

func TestEscaping(t *testing.T) {
	data := []string{
		`/a~1b`,
		`/m~0n`,
		`/a~1b/m~0n`,
	}
	for _, pat := range data {
		p, err := jspointer.New(pat)
		if !assert.NoError(t, err, "jspointer.New should succeed for '%s'", pat) {
			return
		}

		if !assert.Equal(t, pat, p.Expression(), "input pattern and generated expression should match") {
			return
		}
	}
}

func runmatch(t *testing.T, pat string, m interface{}) (jspointer.Result, error) {
	p, err := jspointer.New(pat)
	if !assert.NoError(t, err, "jspointer.New should succeed for '%s'", pat) {
		return jspointer.Result{}, err
	}

	return p.Get(m)
}

func TestFullDocument(t *testing.T) {
	res, err := runmatch(t, ``, target)
	if !assert.NoError(t, err, "jsonpointer.Get should succeed") {
		return
	}
	if !assert.Equal(t, res.Item, target, "res.Item should be equal to target") {
		return
	}
}

func TestGetObject(t *testing.T) {
	pats := map[string]interface{}{
		`/obj/a`:       float64(1),
		`/obj/b`:       float64(2),
		`/obj/c/0`:     float64(3),
		`/obj/c/1`:     float64(4),
		`/obj/d/1/f/0`: float64(50),
	}
	for pat, expected := range pats {
		res, err := runmatch(t, pat, target)
		if !assert.NoError(t, err, "jsonpointer.Get should succeed") {
			return
		}

		if !assert.Equal(t, res.Item, expected, "res.Item should be equal to expected") {
			return
		}
	}
}

func TestGetArray(t *testing.T) {
	foo := target["foo"].([]interface{})
	pats := map[string]interface{}{
		`/foo/0`: foo[0],
		`/foo/1`: foo[1],
	}
	for pat, expected := range pats {
		res, err := runmatch(t, pat, target)
		if !assert.NoError(t, err, "jsonpointer.Get should succeed") {
			return
		}

		if !assert.Equal(t, res.Item, expected, "res.Item should be equal to expected") {
			return
		}
	}
}

func TestSet(t *testing.T) {
	var m interface{}
	json.Unmarshal([]byte(`{
"a": [{"b": 1, "c": 2}], "d": 3
}`), &m)

	p, err := jspointer.New(`/a/0/c`)
	if !assert.NoError(t, err, "jspointer.New should succeed") {
		return
	}

	if !assert.NoError(t, p.Set(m, 999), "jspointer.Set should succeed") {
		return
	}

	res, err := runmatch(t, `/a/0/c`, m)
	if !assert.NoError(t, err, "jsonpointer.Get should succeed") {
		return
	}

	if !assert.Equal(t, res.Item, 999, "res.Item should be equal to expected") {
		return
	}
}

func TestStruct(t *testing.T) {
	var s struct {
		Foo string `json:"foo"`
		Bar map[string]interface{} `json:"bar"`
		Baz int
		quux int
	}

	if !assert.NoError(t, json.Unmarshal([]byte(`{
"foo": "foooooooo",
"bar": {"a": 0, "b": 1},
"baz": 2
}`), &s), "json.Unmarshal succeeds") {
		return
	}

	res, err := runmatch(t, `/bar/b`, s)
	if !assert.NoError(t, err, "jsonpointer.Get should succeed") {
		return
	}

	if !assert.Equal(t, res.Item, float64(1), "res.Item should be equal to expected value") {
		return
	}
}
