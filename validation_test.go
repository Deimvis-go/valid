package valid_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Deimvis-go/valid"
)

// TODO: add test for a nested nil field.

func TestDeep(t *testing.T) {
	testDeep(t, []testCase{
		{
			&A{V: 42},
			nil,
		},
		{
			&A{V: 1},
			errors.New("A: wrong value"),
		},
		{
			&A{K: "key", V: 42},
			errors.New("A: non-empty key"),
		},
		{
			&A{K: "key", V: 1},
			errors.New("A: non-empty key"),
		},
	})
}

func TestDeep_Nil(t *testing.T) {
	require.NotPanics(t, func() {
		_ = valid.Deep(nil)
	})
}

func TestDeep_WithRecursion(t *testing.T) {
	testDeep(t, []testCase{
		{
			&AExported{Valid: true, A: A{V: 42}},
			nil,
		},
		{
			&AExported{Valid: false, A: A{V: 42}},
			errors.New("AExported: not valid"),
		},
		{
			&AExported{Valid: true, A: A{V: 1}},
			errors.New("AExported.A: wrong value"),
		},
		{
			&AExportedPtr{},
			nil,
		},
		{
			&AExportedPtr{AExported: &AExported{Valid: true, A: A{V: 1}}},
			errors.New("AExportedPtr.AExported.A: wrong value"),
		},
		{
			&AUnexported{Valid: true, a: A{V: 42}},
			nil,
		},
		{
			&AUnexported{Valid: false, a: A{V: 42}},
			errors.New("AUnexported: not valid"),
		},
		{
			&AUnexported{Valid: true, a: A{V: 1}},
			nil,
		},
		{
			&AUnexportedPtr{a: ptr(A{V: 42})},
			nil,
		},
		{
			&AUnexportedPtr{a: ptr(A{V: 1})},
			nil,
		},
		{
			&AUnexportedEmbed{aLowercase: A{V: 42}},
			nil,
		},
		{
			&AUnexportedEmbed{aLowercase: A{V: 1}},
			errors.New("AUnexportedEmbed: wrong value"),
		},
		{
			AUnexportedEmbed{aLowercase: A{V: 42}},
			nil,
		},
		{
			AUnexportedEmbed{aLowercase: A{V: 1}},
			errors.New("AUnexportedEmbed: wrong value"),
		},
	})
}

func TestDeep_WithEmbeds(t *testing.T) {
	testDeep(t, []testCase{
		{
			&D{A: A{V: 42}},
			nil,
		},
		{
			&D{A: A{V: 1}},
			errors.New("D.A: wrong value"),
		},
	})
}

func TestDeep_FromValue(t *testing.T) {
	testDeep(t, []testCase{
		{
			ValidatableByValue{Valid: true},
			nil,
		},
		{
			ValidatableByValue{Valid: false},
			errors.New("ValidatableByValue: not valid"),
		},
		{
			ValidatableByPointer{Valid: true},
			nil,
		},
		{
			ValidatableByPointer{Valid: false},
			errors.New("ValidatableByPointer: not valid"),
		},
	})
}

func TestDeep_FromPointer(t *testing.T) {
	testDeep(t, []testCase{
		{
			&ValidatableByValue{Valid: true},
			nil,
		},
		{
			&ValidatableByValue{Valid: false},
			errors.New("ValidatableByValue: not valid"),
		},
		{
			&ValidatableByPointer{Valid: true},
			nil,
		},
		{
			&ValidatableByPointer{Valid: false},
			errors.New("ValidatableByPointer: not valid"),
		},
	})
}

func TestDeep_Slice(t *testing.T) {
	testDeep(t, []testCase{
		{
			[]A{},
			nil,
		},
		{
			[]A{{V: 42}},
			nil,
		},
		{
			[]A{{V: 1}},
			errors.New("<[]valid_test.A>[0]: wrong value"),
		},
		{
			[]A{{V: 42}, {V: 42}},
			nil,
		},
		{
			[]A{{V: 1}, {V: 1}},
			errors.New("<[]valid_test.A>[0]: wrong value"),
		},
		{
			[]A{{V: 42}, {V: 1}},
			errors.New("<[]valid_test.A>[1]: wrong value"),
		},
		{
			[]A{{V: 1}, {V: 42}},
			errors.New("<[]valid_test.A>[0]: wrong value"),
		},
	})
}

func TestDeep_CustomSlice(t *testing.T) {
	testDeep(t, []testCase{
		{
			MySlice{"valid"},
			nil,
		},
		{
			MySlice{"not_valid"},
			errors.New("MySlice: not valid"),
		},
	})
}

func TestDeep_CustomMap(t *testing.T) {
	testDeep(t, []testCase{
		{
			MyMap{"valid": 1},
			nil,
		},
		{
			MyMap{"valid": 0},
			errors.New("MyMap: not valid"),
		},
	})
}

func TestDeep_CustomString(t *testing.T) {
	testDeep(t, []testCase{
		{
			MyString("valid"),
			nil,
		},
		{
			MyString("not valid"),
			errors.New("MyString: not valid"),
		},
	})
}

func TestDeep_ErrorContainsPath(t *testing.T) {
	type mapT = map[string]AlwaysInvalid
	type sliceT = []mapT
	type Root struct {
		Field sliceT
	}
	testDeep(t, []testCase{
		{
			Root{
				Field: sliceT{
					nil,
					mapT{"key": AlwaysInvalid{}},
					nil,
				},
			},
			errors.New(`Root.Field[1]["key"]: always invalid`),
		},
		{
			sliceT{
				nil,
				mapT{"key": AlwaysInvalid{}},
				nil,
			},
			errors.New(`<[]map[string]valid_test.AlwaysInvalid>[1]["key"]: always invalid`),
		},
		{
			mapT{"key": AlwaysInvalid{}},
			errors.New(`<map[string]valid_test.AlwaysInvalid>["key"]: always invalid`),
		},
	})
}

func testDeep(t *testing.T, testCases []testCase) {
	t.Helper()
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			actual := valid.Deep(tc.obj)
			if tc.exp != nil {
				require.NotNil(t, actual, "error should be not nil")
				require.Equal(t, tc.exp.Error(), actual.Error())
			} else {
				require.Nil(t, actual, "error should be nil")
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

type testCase struct {
	obj any
	exp error
}

type A struct {
	K string
	V int
}

func (a *A) ValidateSelf() error {
	if len(a.K) > 0 {
		return errors.New("non-empty key")
	}
	if a.V != 42 {
		return errors.New("wrong value")
	}
	return nil
}

type AExported struct {
	Valid bool
	A     A
}

func (b *AExported) ValidateSelf() error {
	if !b.Valid {
		return errors.New("not valid")
	}
	return nil
}

type AUnexported struct {
	Valid bool
	a     A //nolint:unused // exercised via reflection-based validation
}

func (b *AUnexported) ValidateSelf() error {
	if !b.Valid {
		return errors.New("not valid")
	}
	return nil
}

type AExportedPtr struct {
	AExported *AExported
}

type AUnexportedPtr struct {
	a *A //nolint:unused // exercised via reflection-based validation
}

type aLowercase = A

type AUnexportedEmbed struct {
	aLowercase
}

type D struct {
	A
	other string //nolint:unused // unexported field exists for test coverage
}

type ValidatableByValue struct {
	Valid bool
}

func (v ValidatableByValue) ValidateSelf() error {
	if !v.Valid {
		return errors.New("not valid")
	}
	return nil
}

type ValidatableByPointer struct {
	Valid bool
}

func (v *ValidatableByPointer) ValidateSelf() error {
	if !v.Valid {
		return errors.New("not valid")
	}
	return nil
}

type MySlice []string

func (s MySlice) ValidateSelf() error {
	if len(s) == 0 || s[0] != "valid" {
		return errors.New("not valid")
	}
	return nil
}

type MyMap map[string]int

func (m MyMap) ValidateSelf() error {
	if v, ok := m["valid"]; ok && v != 1 {
		return errors.New("not valid")
	}
	return nil
}

type MyString string

func (s MyString) ValidateSelf() error {
	if s != "valid" {
		return errors.New("not valid")
	}
	return nil
}

type AlwaysInvalid struct{}

func (ai AlwaysInvalid) ValidateSelf() error {
	return errors.New("always invalid")
}
