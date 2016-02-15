package jspointer

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var ErrInvalidPointer = errors.New("invalid pointer")
var ErrNoSuchKey = errors.New("no such key in object")

var ctxPool = sync.Pool{
	New: moreCtx,
}

func moreCtx() interface{} {
	return &matchCtx{}
}

func getCtx() *matchCtx {
	return ctxPool.Get().(*matchCtx)
}

func releaseCtx(ctx *matchCtx) {
	ctx.err = nil
	ctx.set = false
	ctx.tokens = nil
	ctx.result = nil
	ctxPool.Put(ctx)
}

const (
	EncodedTilde = "~0"
	EncodedSlash = "~1"
	Separator    = '/'
)

type JSPointer struct {
	tokens []string
}

type Result struct {
	Item interface{}
	Kind reflect.Kind
}

func New(path string) (*JSPointer, error) {
	var p JSPointer
	if err := p.parse(path); err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *JSPointer) parse(s string) error {
	if s == "" {
		return nil
	}

	if s[0] != Separator {
		return ErrInvalidPointer
	}

	prev := 0
	tokens := []string{}
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case Separator:
			tokens = append(tokens, s[prev+1:i])
			prev = i
		}
	}

	if prev != len(s) {
		tokens = append(tokens, s[prev+1:])
	}

	dtokens := make([]string, 0, len(tokens))
	for _, t := range tokens {
		t = strings.Replace(strings.Replace(t, EncodedSlash, "/", -1), EncodedTilde, "~", -1)
		dtokens = append(dtokens, t)
	}

	p.tokens = dtokens
	return nil
}

func (p JSPointer) Expression() string {
	pat := ""
	for _, token := range p.tokens {
		p2 := strings.Replace(strings.Replace(token, "~", EncodedTilde, -1), "/", EncodedSlash, -1)
		pat = pat + "/" + p2
	}
	return pat
}

func (p JSPointer) Get(item interface{}) (*Result, error) {
	ctx := getCtx()
	defer releaseCtx(ctx)

	ctx.tokens = p.tokens
	ctx.apply(item)
	return ctx.result, ctx.err
}

func (p JSPointer) Set(item interface{}, value interface{}) error {
	ctx := getCtx()
	defer releaseCtx(ctx)

	ctx.set = true
	ctx.tokens = p.tokens
	ctx.setvalue = value
	ctx.apply(item)
	return ctx.err
}

type matchCtx struct {
	err      error
	result   *Result
	set      bool
	setvalue interface{}
	tokens   []string
}

// json element name -> field index
type fieldMap map[string]int

// struct type to -> fieldMap
var struct2FieldMap = map[reflect.Type]fieldMap{}

func getStructMap(v reflect.Value) fieldMap {
	t := v.Type()
	fm, ok := struct2FieldMap[t]
	if ok {
		return fm
	}

	fm = fieldMap{}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}

		tag := sf.Tag.Get("json")
		if tag == "" || tag == "-" || tag[0] == ',' {
			fm[sf.Name] = i
			continue
		}

		flen := 0
		for j := 0; j < len(tag); j++ {
			if tag[j] == ',' {
				break
			}
			flen = j
		}
		fm[tag[:flen+1]] = i
	}

	struct2FieldMap[t] = fm

	return fm
}

func (c *matchCtx) apply(item interface{}) {
	if len(c.tokens) == 0 {
		c.result = &Result{
			Kind: reflect.TypeOf(item).Kind(),
			Item: item,
		}
		return
	}

	lastidx := len(c.tokens) - 1
	node := item
	for tidx, token := range c.tokens {
		v := reflect.ValueOf(node)
		switch v.Kind() {
		case reflect.Struct:
			sm := getStructMap(v)
			i, ok := sm[token]
			if !ok {
				c.err = ErrNoSuchKey
				return
			}
			f := v.Field(i)
			if tidx == lastidx {
				if c.set {
					if !f.CanSet() {
						c.err = errors.New("field cannot be set to")
						return
					}
					f.Set(reflect.ValueOf(c.setvalue))
					return
				}
				c.result = &Result{Kind: f.Kind(), Item: f.Interface()}
				return
			}
			node = f.Interface()
		case reflect.Map:
			m := node.(map[string]interface{})
			n, ok := m[token]
			if !ok {
				c.err = ErrNoSuchKey
				return
			}

			if tidx == lastidx {
				if c.set {
					m[token] = c.setvalue
				} else {
					c.result = &Result{
						Kind: v.Kind(),
						Item: n,
					}
				}
				return
			}

			node = n
		case reflect.Slice:
			m := node.([]interface{})
			wantidx, err := strconv.Atoi(token)
			if err != nil {
				c.err = err
				return
			}

			if wantidx < 0 || len(m) <= wantidx {
				c.err = errors.New("array index out of bounds")
				return
			}

			if tidx == lastidx {
				if c.set {
					m[wantidx] = c.setvalue
				} else {
					c.result = &Result{
						Kind: v.Kind(),
						Item: m[wantidx],
					}
				}
				return
			}
			node = m[wantidx]
		default:
			c.err = errors.New("not found")
			return
		}
	}

	// If you fell through here, there was a big problem
	c.err = errors.New("not found")
}
