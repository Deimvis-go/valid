package valid

import (
	"fmt"
	"reflect"
	"strings"
)

// vcontext carries validation state across the recursive traversal.
// Today it only holds the current path; it exists so that new state
// (options, warnings, custom tag lookups, …) can be added without
// changing function signatures.
type vcontext struct {
	p *path
}

// path tracks the current position inside the value being validated,
// rendered as a dotted/bracketed string (e.g. `Root.Field[1]["key"]`).
type path struct {
	v []string
}

func (p *path) String() string {
	return strings.Join(p.v, "")
}

func (p *path) WithStructField(sf reflect.StructField, fn func()) {
	p.PushStructField(sf)
	defer p.Pop()
	fn()
}

func (p *path) WithSliceElement(ind int, fn func()) {
	p.PushSliceElement(ind)
	defer p.Pop()
	fn()
}

func (p *path) WithMapKey(key reflect.Value, fn func()) {
	p.PushMapKey(key)
	defer p.Pop()
	fn()
}

func (p *path) PushRootObject(obj any) {
	if len(p.v) != 0 {
		panic("valid: bug: object is not root")
	}
	var name string
	v := extractInternalValue(reflect.ValueOf(obj))
	if v.Kind() == reflect.Invalid {
		name = "nil"
	} else {
		vt := v.Type()
		if vt.Name() != "" {
			name = vt.Name()
		} else {
			// reflect.Type.String() can include a package prefix (e.g.
			// "[]valid.A"); wrap it to keep the root distinguishable from
			// named types.
			name = fmt.Sprintf("<%s>", vt.String())
		}
	}
	p.v = append(p.v, name)
}

func (p *path) PushStructField(sf reflect.StructField) {
	p.v = append(p.v, "."+sf.Name)
}

func (p *path) PushSliceElement(ind int) {
	p.v = append(p.v, fmt.Sprintf("[%d]", ind))
}

func (p *path) PushMapKey(key reflect.Value) {
	p.v = append(p.v, fmt.Sprintf("[%s]", p.formatMapKey(key)))
}

func (p *path) Pop() {
	p.v = p.v[:len(p.v)-1]
}

func (p *path) formatMapKey(key reflect.Value) string {
	if key.CanInterface() {
		if s, ok := key.Interface().(fmt.Stringer); ok {
			return s.String()
		}
	}
	if key.Kind() == reflect.String {
		return fmt.Sprintf("%q", key.String())
	}
	return key.String()
}
