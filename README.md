# valid

`valid` is a small extension for [go-playground/validator/v10][validator]
that lets a struct define its own validation via a `ValidateSelf() error`
method — no global registration required. `valid.Deep` then walks the value
recursively, combining struct-tag validation with any `ValidateSelf` methods
it finds along the way, and returns an error annotated with the path to the
failing field.

[validator]: https://pkg.go.dev/github.com/go-playground/validator/v10

## Install

```bash
go get github.com/Deimvis-go/valid
```

## Quick start

```go
package main

import (
    "errors"
    "fmt"

    "github.com/Deimvis-go/valid"
)

type User struct {
    Name string `validate:"required"`
    Age  int
}

func (u *User) ValidateSelf() error {
    if u.Age < 0 {
        return errors.New("age must be non-negative")
    }
    return nil
}

func main() {
    fmt.Println(valid.Deep(&User{Name: "alice", Age: -1}))
    // User: age must be non-negative
}
```

A slightly larger example that recurses through nested structs and slices
lives in [`examples/quickstart`](./examples/quickstart).

## How it works

* `valid.Deep(obj)` validates `obj` using the default validator and any
  `ValidateSelf` methods found on it.
* `valid.DeepWith(v, obj)` does the same but with a caller-provided
  `*validator.Validate`, which is useful when you have pre-registered custom
  tag validators.
* Traversal is recursive over struct fields (including embedded structs),
  slice/array elements, and map values. Unexported fields are skipped
  because reflection cannot safely invoke methods on them.
* `ValidateSelf` may be defined on either a value or a pointer receiver —
  `valid` will find and call it either way.
* Errors are wrapped with the path to the offending value, e.g.
  `Team.Members[1]: age must be non-negative`. Path components that contain
  the delimiter backtick are escaped.

## Behaviour notes

* A nil root object returns an error, except when it is a nil slice or
  array (which are considered valid).
* A nil pointer *inside* a parent is silently skipped rather than treated as
  an error — this makes it easy to model optional fields.
* Maps are validated before their values so that a map-level `ValidateSelf`
  can short-circuit checks on entries.
* `valid.Deep(slice)` is **not** equivalent to calling
  `valid.Deep(slice[i])` for each element: the slice form skips nil entries,
  whereas calling directly on a nil struct returns an error.

## Compatibility

Go 1.22+. Only depends on
[`github.com/go-playground/validator/v10`](https://github.com/go-playground/validator).

