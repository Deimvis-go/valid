package valid

import "strings"

// onPanic runs fn only if the surrounding code panicked, then re-raises the
// original panic so normal propagation continues. It must be invoked
// directly via `defer onPanic(...)`; wrapping it in another function
// prevents recover from seeing the panic. See
// https://stackoverflow.com/a/49344592.
func onPanic(fn func(r any)) {
	// Since Go 1.21, recover returns a non-nil *runtime.PanicNilError for
	// panic(nil), so a nil check is sufficient to detect any panic.
	// https://pkg.go.dev/runtime#PanicNilError
	if r := recover(); r != nil {
		fn(r)
		panic(r)
	}
}

// escape doubles every occurrence of esc in s. It is used to make path
// components unambiguous when they appear in error messages wrapped in the
// same delimiter.
func escape(s string, esc byte) string {
	if !strings.ContainsRune(s, rune(esc)) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 1)
	for i := 0; i < len(s); i++ {
		if s[i] == esc {
			b.WriteByte(esc)
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
