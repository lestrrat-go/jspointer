package jspointer

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

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

// New creates a new JSON pointer for given path spec. If the path fails
// to be parsed, an error is returned
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

// String returns the stringified version of this JSON pointer
func (p JSPointer) String() string {
	pat := ""
	for _, token := range p.tokens {
		p2 := strings.Replace(strings.Replace(token, "~", EncodedTilde, -1), "/", EncodedSlash, -1)
		pat = pat + "/" + p2
	}
	return pat
}

// Get applies the JSON pointer to the given item, and returns
// the result. 
func (p JSPointer) Get(item interface{}) (interface{}, error) {
	ctx := getCtx()
	defer releaseCtx(ctx)

	ctx.tokens = p.tokens
	ctx.apply(item)
	return ctx.result, ctx.err
}

// Set applies the JSON pointer to the given item, and sets the
// value accordingly.
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
	result   interface{}
	set      bool
	setvalue interface{}
	tokens   []string
}

// json element name -> field index
type fieldMap map[string]int

// struct type to -> fieldMap
var struct2FieldMap = map[reflect.Type]fieldMap{}
var fieldmapMutex = sync.Mutex{}

func getStructMap(v reflect.Value) fieldMap {
	fieldmapMutex.Lock()
	defer fieldmapMutex.Unlock()
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
		c.result = item
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
				c.err = ErrNotFound
				return
			}
			f := v.Field(i)
			if tidx == lastidx {
				if c.set {
					if !f.CanSet() {
						c.err = ErrCanNotSet
						return
					}
					f.Set(reflect.ValueOf(c.setvalue))
					return
				}
				c.result = f.Interface()
				return
			}
			node = f.Interface()
		case reflect.Map:
			m := node.(map[string]interface{})
			n, ok := m[token]
			if !ok {
				c.err = ErrNotFound
				return
			}

			if tidx == lastidx {
				if c.set {
					m[token] = c.setvalue
				} else {
					c.result = n
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
				c.err = ErrSliceIndexOutOfBounds
				return
			}

			if tidx == lastidx {
				if c.set {
					m[wantidx] = c.setvalue
				} else {
					c.result = m[wantidx]
				}
				return
			}
			node = m[wantidx]
		default:
			c.err = ErrNotFound
			return
		}
	}

	// If you fell through here, there was a big problem
	c.err = ErrNotFound
}
