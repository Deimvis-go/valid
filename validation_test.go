package valid

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/Deimvis/go-ext/go1.25/xptr"
)

// TODO: add test for nested nil field

func TestDeep(t *testing.T) {
	testCases := []testCase{
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
	}
	testDeep(t, testCases)
}

func TestDeep_Nil(t *testing.T) {
	require.NotPanics(t, func() {
		Deep(nil)
	})
}

func TestDeep_WithRecursion(t *testing.T) {
	testCases := []testCase{
		{
			&A_Exported{Valid: true, A: A{V: 42}},
			nil,
		},
		{
			&A_Exported{Valid: false, A: A{V: 42}},
			errors.New("A_Exported: not valid"),
		},
		{
			&A_Exported{Valid: true, A: A{V: 1}},
			errors.New("A_Exported.A: wrong value"),
		},
		{
			&A_ExportedPtr{},
			nil,
		},
		{
			&A_ExportedPtr{A_Exported: &A_Exported{Valid: true, A: A{V: 1}}},
			errors.New("A_ExportedPtr.A_Exported.A: wrong value"),
		},
		{
			&A_Unexported{Valid: true, a: A{V: 42}},
			nil,
		},
		{
			&A_Unexported{Valid: false, a: A{V: 42}},
			errors.New("A_Unexported: not valid"),
		},
		{
			&A_Unexported{Valid: true, a: A{V: 1}},
			nil,
		},
		{
			&A_UnexportedPtr{a: xptr.T(A{V: 42})},
			nil,
		},
		{
			&A_UnexportedPtr{a: xptr.T(A{V: 1})},
			nil,
		},
		{
			&A_UnexportedEmbed{a_lowercase: A{V: 42}},
			nil,
		},
		{
			&A_UnexportedEmbed{a_lowercase: A{V: 1}},
			errors.New("A_UnexportedEmbed: wrong value"),
		},
		{
			A_UnexportedEmbed{a_lowercase: A{V: 42}},
			nil,
		},
		{
			A_UnexportedEmbed{a_lowercase: A{V: 1}},
			errors.New("A_UnexportedEmbed: wrong value"),
		},
	}
	testDeep(t, testCases)
}

func TestDeep_WithEmbeds(t *testing.T) {
	testCases := []testCase{
		{
			&D{A: A{V: 42}},
			nil,
		},
		{
			&D{A: A{V: 1}},
			errors.New("D.A: wrong value"),
		},
	}
	testDeep(t, testCases)
}

func TestDeep_FromValue(t *testing.T) {
	testCases := []testCase{
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
	}
	testDeep(t, testCases)
}

func TestDeep_FromPointer(t *testing.T) {
	testCases := []testCase{
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
	}
	testDeep(t, testCases)
}

func TestDeep_Slice(t *testing.T) {
	testCases := []testCase{
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
			errors.New("<[]valid.A>[0]: wrong value"),
		},
		{
			[]A{{V: 42}, {V: 42}},
			nil,
		},
		{
			[]A{{V: 1}, {V: 1}},
			errors.New("<[]valid.A>[0]: wrong value"),
		},
		{
			[]A{{V: 42}, {V: 1}},
			errors.New("<[]valid.A>[1]: wrong value"),
		},
		{
			[]A{{V: 1}, {V: 42}},
			errors.New("<[]valid.A>[0]: wrong value"),
		},
	}
	testDeep(t, testCases)
}

func TestDeep_CustomSlice(t *testing.T) {
	testCases := []testCase{
		{
			MySlice{"valid"},
			nil,
		},
		{
			MySlice{"not_valid"},
			errors.New("MySlice: not valid"),
		},
	}
	testDeep(t, testCases)
}

func TestDeep_CustomMap(t *testing.T) {
	testCases := []testCase{
		{
			MyMap{"valid": 1},
			nil,
		},
		{
			MyMap{"valid": 0},
			errors.New("MyMap: not valid"),
		},
	}
	testDeep(t, testCases)
}

func TestDeep_CustomString(t *testing.T) {
	tcs := []testCase{
		{
			MyString("valid"),
			nil,
		},
		{
			MyString("not valid"),
			errors.New("MyString: not valid"),
		},
	}
	testDeep(t, tcs)
}

func TestDeep_ErrorContainsPath(t *testing.T) {
	type mapT = map[string]AlwaysInvalid
	type sliceT = []mapT
	type Root struct {
		Field sliceT
	}
	tcs := []testCase{
		{
			Root{
				Field: sliceT{
					nil,
					mapT{
						"key": AlwaysInvalid{},
					},
					nil,
				},
			},
			errors.New(`Root.Field[1]["key"]: always invalid`),
		},
		{
			sliceT{
				nil,
				mapT{
					"key": AlwaysInvalid{},
				},
				nil,
			},
			errors.New(`<[]map[string]valid.AlwaysInvalid>[1]["key"]: always invalid`),
		},
		{
			mapT{
				"key": AlwaysInvalid{},
			},
			errors.New(`<map[string]valid.AlwaysInvalid>["key"]: always invalid`),
		},
	}
	testDeep(t, tcs)
}

func testDeep(t *testing.T, testCases []testCase) {
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			actual := Deep(tc.obj)
			if tc.exp != nil {
				require.NotNil(t, actual, "error should be not nil")
				require.Equal(t, tc.exp.Error(), actual.Error())
			} else {
				require.Nil(t, actual, "error should be nil")
			}
		})
	}
}

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

type A_Exported struct {
	Valid bool
	A     A
}

func (b *A_Exported) ValidateSelf() error {
	if !b.Valid {
		return errors.New("not valid")
	}
	return nil
}

type A_Unexported struct {
	Valid bool
	a     A
}

func (b *A_Unexported) ValidateSelf() error {
	if !b.Valid {
		return errors.New("not valid")
	}
	return nil
}

type A_ExportedPtr struct {
	A_Exported *A_Exported
}

type A_UnexportedPtr struct {
	a *A
}

type a_lowercase = A
type A_UnexportedEmbed struct {
	a_lowercase
}

type D struct {
	A
	other string
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
