// Package valid is an extension for the go-playground/validator/v10 package
// that lets structs provide their own validation via a ValidateSelf method,
// without a separate registration step.
package valid

import (
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// TODO: support specifying a validate function for types as an option to
//       Deep (e.g. valid.Deep(queries, {Query: func(q) error { … }})) —
//       both as an additional validator and as an override.
// TODO: change traversal order from top-down to bottom-up.
// TODO: support specifying the tag name to look up for field names in
//       error paths (e.g. use `json` tags).
// TODO: support warnings (a custom method that receives a context it can
//       use to emit warnings in addition to / instead of errors).
// TODO: include more debug information in the error wrapped by onPanic.
// TODO: add an option to skip pointer allocation as an optimisation
//       (used for the isValidatable check).
// TODO: support an option to tolerate a nil root.

// TODO: consider renaming the method to Validate(). ValidateSelf was
//       originally chosen to make it clear that the method should not
//       recurse and should validate only its own level. That naming also
//       leaves the Validate name free for a user-defined recursive entry
//       point that does not interact with this package. Plus it signals
//       that the method validates the receiver itself — Validate() sounds
//       more like a method on a validator than on a struct.

// Validatable is implemented by types that can validate themselves.
//
// The method is intentionally named ValidateSelf (rather than Validate) to
// signal that an implementation should only check the receiver's own fields
// and must not recurse into nested structs — the package handles recursion.
// This also leaves the Validate name free for users to define their own
// higher-level validation entry points.
type Validatable interface {
	ValidateSelf() error
}

var defaultValidator = validator.New(validator.WithRequiredStructEnabled())

// TODO: rename to Should.

// Deep validates obj using both struct validation tags and any ValidateSelf
// methods found on obj or its nested values.
//
// Deep recurses through struct fields (including embedded structs), slice and
// array elements, and map values. A nil root obj is treated as an error,
// except for nil slices/arrays which are considered valid.
func Deep(obj any) error {
	return DeepWith(defaultValidator, obj)
}

// TODO: remove DeepWith and just expose options on Deep.

// DeepWith behaves like Deep but uses the provided validator for tag-based
// struct validation. This is useful when you have pre-registered custom
// validators or want to reuse a shared *validator.Validate instance.
func DeepWith(v *validator.Validate, obj any) error {
	objV := extractInternalValue(reflect.ValueOf(obj))
	if objV.Kind() == reflect.Struct {
		if err := v.Struct(obj); err != nil {
			return err
		}
	}
	c := &vcontext{
		p: &path{},
	}
	c.p.PushRootObject(obj)
	defer c.p.Pop()
	return validateRecursively(c, objV, false)
}

func validateRecursively(c *vcontext, v reflect.Value, ignoreNil bool) error {
	return validate(c, extractInternalValue(v), ignoreNil)
}

func validate(c *vcontext, v reflect.Value, ignoreNil bool) error {
	defer onPanic(func(r any) {
		log.Printf("valid: panicked during validate, cur_level = %s\n", v.String())
	})

	switch v.Kind() {
	case reflect.Struct:
		return validateStruct(c, v)
	case reflect.Slice, reflect.Array:
		return validateSlice(c, v)
	case reflect.Map:
		return validateMap(c, v)
	case reflect.Pointer:
		// extractInternalValue only leaves a pointer here when it is nil.
		if !v.IsNil() {
			panic("valid: bug: validate called with non-nil pointer (internal value was not extracted)")
		}
		if ignoreNil {
			return nil
		}
		return validateNilPtr(v)
	case reflect.Invalid:
		if ignoreNil {
			return nil
		}
		return errors.New("object is invalid: nil pointer")
	default:
		return validateSelf(c, v)
	}
}

func validateStruct(c *vcontext, v reflect.Value) error {
	vt := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if !vt.Field(i).IsExported() {
			continue
		}
		var err error
		c.p.WithStructField(vt.Field(i), func() {
			err = validateRecursively(c, v.Field(i), true /* ignoreNil */)
		})
		if err != nil {
			return err
		}
	}
	return validateSelf(c, v)
}

func validateSlice(c *vcontext, v reflect.Value) error {
	for i := 0; i < v.Len(); i++ {
		var err error
		c.p.WithSliceElement(i, func() {
			err = validateRecursively(c, v.Index(i), true /* ignoreNil */)
		})
		if err != nil {
			return err
		}
	}
	return validateSelf(c, v)
}

func validateMap(c *vcontext, v reflect.Value) error {
	if err := validateSelf(c, v); err != nil {
		return err
	}
	for _, key := range v.MapKeys() {
		var err error
		c.p.WithMapKey(key, func() {
			err = validateRecursively(c, v.MapIndex(key), true /* ignoreNil */)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func validateNilPtr(v reflect.Value) error {
	switch v.Type().Elem().Kind() {
	case reflect.Slice, reflect.Array:
		return nil
	}
	return errors.New("object is invalid: nil pointer")
}

func validateSelf(c *vcontext, v reflect.Value) error {
	if isValidatable(v) {
		return validateSelfDo(c, v)
	}
	if ptr, ok := tryGetPtr(v); ok && isValidatable(ptr) {
		return validateSelfDo(c, ptr)
	}
	return nil
}

var validatableType = reflect.TypeOf((*Validatable)(nil)).Elem()

func isValidatable(v reflect.Value) bool {
	isNil := v.Kind() == reflect.Pointer && v.IsNil()
	return !isNil && v.CanInterface() && v.Type().Implements(validatableType)
}

func validateSelfDo(c *vcontext, v reflect.Value) error {
	validatable, ok := v.Interface().(Validatable)
	if !ok {
		panic("valid: value doesn't implement Validatable")
	}
	// TODO: add current level to the error message.
	if err := validatable.ValidateSelf(); err != nil {
		return fmt.Errorf("%s: %w", escape(c.p.String(), '`'), err)
	}
	return nil
}

func tryGetPtr(v reflect.Value) (reflect.Value, bool) {
	// Mirrors encoding/json: addressable, named, non-pointer values can have
	// pointer methods reached through Addr().
	if v.Kind() != reflect.Pointer && v.Type().Name() != "" && v.CanAddr() {
		return v.Addr(), true
	}

	// v.CanInterface checks flagRO — it reports whether the field is
	// unexported. If it is exported (flagRO == 0) we can copy its value into
	// a new pointer. v.CanSet is not suitable here because it also requires
	// addressability.
	if !v.CanInterface() {
		return v, false
	}
	ptr := reflect.New(v.Type())
	ptr.Elem().Set(v)
	return ptr, true
}

// extractInternalValue resolves pointers and interfaces until the value is
// neither. Adapted from go-playground/validator:
// https://github.com/go-playground/validator/blob/a947377040f8ebaee09f20d09a745ec369396793/util.go#L15
func extractInternalValue(current reflect.Value) reflect.Value {
	for {
		switch current.Kind() {
		case reflect.Pointer, reflect.Interface:
			if current.IsNil() {
				return current
			}
			current = current.Elem()
		default:
			return current
		}
	}
}
