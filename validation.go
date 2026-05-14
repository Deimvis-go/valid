package valid

import (
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/Deimvis/go-ext/go1.25/ext"
	"github.com/Deimvis/go-ext/go1.25/xstrings"
)

// TODO: support specifying validate function for types as option to Deep() (want to esaily pass like, valid.Deep(queries, {Query: func(a) error { q.ValidateHasNonEmptyContent()  }}))
// - support as additional validate or as overridden validate
// TODO: change up-down validation order to down-up
// TODO: support specifying tag name to lookup for field name (e.g. use `json` tags for writing path)
// TODO: support warnings (custom method providing context in its function which allows to send warnings)
// TODO: add defer ext.OnPanic and add debug information to panic error
// TODO: add option to prevent pointer allocation as optimization (used for isValidatable check)
// TODO: support option to ignore nil root

// TODO: consider renaming method to Validate().
// Originally ValidateSelf was chosen in order to clarify that method
// should not validate recursively and perform validation only on its level.
// So naming it ValidateSelf allowed users to implement another method Validate
// that validates recursively and doesnt interact with this package.
// Plus it indicates that it should validate itself and not some input argument -
// Validate() sound more like a method for validator, not a struct to validate itself.
type Validatable interface {
	ValidateSelf() error
}

var emptyValidator = validator.New(validator.WithRequiredStructEnabled())

// TODO: rename to Should
// Deep validates given object using both
// validation tags and ValidateSelf methods.
// It panics when given struct is nil.
// It goes recursively through container elements, substructs (including embedded structs), etc.
func Deep(obj any) error {
	return XDeep(emptyValidator, obj)
}

// TODO: remove it and just add options to Deep()
// XDeep is the same as Deep but allows to pass custom validator.
func XDeep(v *validator.Validate, obj any) error {
	objV := extractInternalValue(reflect.ValueOf(obj))
	if objV.Kind() == reflect.Struct {
		err := v.Struct(obj)
		if err != nil {
			return err
		}
	}
	c := &context{
		p: &path{
			v: []string{},
		},
	}
	c.p.PushRootObject(obj)
	defer c.p.Pop()
	return validateSelfRecursively(c, objV, false)
}

func validateSelfRecursively(c *context, v reflect.Value, ignoreNil bool) error {
	return validate(c, extractInternalValue(v), ignoreNil)
}

func validate(c *context, v reflect.Value, ignoreNil bool) error {
	defer ext.OnPanic(func(r any) {
		log.Printf("panicked during validate, cur_level = %s\n", v.String())
	})
	var err error
	switch v.Kind() {
	case reflect.Struct:
		err = validateStruct(c, v)
	case reflect.Slice, reflect.Array:
		err = validateSlice(c, v)
	case reflect.Map:
		err = validateMap(c, v)
	case reflect.Pointer: // nil pointer
		if !v.IsNil() {
			panic("bug: validate was called non-nil pointer (internal value was not extracted)")
		}
		if ignoreNil {
			err = nil
		} else {
			err = validateNilPtr(c, v)
		}
	case reflect.Invalid:
		if ignoreNil {
			err = nil
		} else {
			err = errors.New("object is invalid: nil pointer")
		}
	default:
		err = validateSelf(c, v)
	}
	return err
}

func validateStruct(c *context, v reflect.Value) error {
	vt := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if vt.Field(i).IsExported() {
			var err error
			c.p.WithStructField(vt.Field(i), func() {
				err = validateSelfRecursively(c, v.Field(i), true /*ignore nil*/)
			})
			if err != nil {
				return err
			}
		}
	}
	err := validateSelf(c, v)
	if err != nil {
		return err
	}
	return nil
}

func validateSlice(c *context, v reflect.Value) error {
	for i := 0; i < v.Len(); i++ {
		var err error
		c.p.WithSliceElement(i, func() {
			err = validateSelfRecursively(c, v.Index(i), true /*ignore nil*/)
		})
		if err != nil {
			return err
		}
	}
	err := validateSelf(c, v)
	if err != nil {
		return err
	}
	return nil
}

func validateMap(c *context, v reflect.Value) error {
	err := validateSelf(c, v)
	if err != nil {
		return err
	}
	for _, key := range v.MapKeys() {
		var err error
		c.p.WithMapKey(key, func() {
			err = validateSelfRecursively(c, v.MapIndex(key), true /*ignore nil*/)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func validateNilPtr(c *context, v reflect.Value) error {
	switch v.Type().Elem().Kind() {
	case reflect.Slice, reflect.Array:
		return nil
	}
	return errors.New("object is invalid: nil pointer")
}

func validateSelf(c *context, v reflect.Value) error {
	if isValidatable(v) {
		return validateSelfDo(c, v)
	}
	ptr, ok := tryGetPtr(v)
	if ok && isValidatable(ptr) {
		return validateSelfDo(c, ptr)
	}
	return nil
}

func isValidatable(v reflect.Value) bool {
	validatable := reflect.TypeOf((*Validatable)(nil)).Elem()
	isNil := (v.Kind() == reflect.Pointer && v.IsNil())
	return !isNil && v.CanInterface() && v.Type().Implements(validatable)
}

func validateSelfDo(c *context, v reflect.Value) error {
	validatable, ok := v.Interface().(Validatable)
	if !ok {
		panic("value doesn't implement Validatable")
	}
	// TODO: add current level to err msg
	err := validatable.ValidateSelf()
	if err != nil {
		return fmt.Errorf(errMsgFmt, xstrings.Escape(c.p.String(), '`'), err)
	}
	return nil
}

func tryGetPtr(v reflect.Value) (reflect.Value, bool) {
	// note: got from json library, when it checks for pointer methods
	if v.Kind() != reflect.Pointer && v.Type().Name() != "" && v.CanAddr() {
		return v.Addr(), true
	}

	// NOTE: v.CanInterface checks for flagRO, it responds whether field is unexported.
	// If it is exported (flagRO == 0) we can use its value to set to a new pointer.
	// v.CanSet() is not suitable because it checks if both (CanAddr() AND flagRO == 0)
	if !v.CanInterface() {
		return v, false
	}
	ptr := reflect.New(v.Type())
	ptr.Elem().Set(v)
	return ptr, true
}

// extractInternalValue resolves pointers and interfaces until value becomes of different kind.
// https://github.com/go-playground/validator/blob/a947377040f8ebaee09f20d09a745ec369396793/util.go#L15
func extractInternalValue(current reflect.Value) reflect.Value {

BEGIN:
	switch current.Kind() {
	case reflect.Pointer:

		if current.IsNil() {
			return current
		}

		current = current.Elem()
		goto BEGIN

	case reflect.Interface:

		if current.IsNil() {
			return current
		}

		current = current.Elem()
		goto BEGIN

	case reflect.Invalid:
		return current

	default:
		return current
	}
}

var (
	errMsgFmt = "%s: %w"
)
