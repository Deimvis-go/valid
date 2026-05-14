package hi

import (
	"github.com/go-playground/validator/v10"

	"github.com/Deimvis-go/valid"
)

var val = validator.New(validator.WithRequiredStructEnabled())

func Validate(obj interface{}) error {
	return valid.XDeep(val, obj)
}
