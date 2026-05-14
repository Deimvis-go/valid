package hi

import (
	"github.com/go-playground/validator/v10"

	"gitlab.corp.mail.ru/ai/godzen/ml_infra/lib/exp/valid"
)

var val = validator.New(validator.WithRequiredStructEnabled())

func Validate(obj interface{}) error {
	return valid.XDeep(val, obj)
}
