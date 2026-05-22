package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateStruct(data interface{}) error {
	err := validate.Struct(data)
	if err != nil {
		var errorMessages []string

		for _, e := range err.(validator.ValidationErrors) {
			field := e.Field()
			tag := e.Tag()
			param := e.Param()

			switch tag {
			case "required":
				errorMessages = append(errorMessages, fmt.Sprintf("%s tidak boleh kosong", field))
			case "email":
				errorMessages = append(errorMessages, fmt.Sprintf("%s harus berupa format email yang valid", field))
			case "min":
				errorMessages = append(errorMessages, fmt.Sprintf("%s minimal harus %s karakter", field, param))
			case "max":
				errorMessages = append(errorMessages, fmt.Sprintf("%s maksimal %s karakter", field, param))
			default:
				errorMessages = append(errorMessages, fmt.Sprintf("%s tidak valid (gagal pada aturan '%s')", field, tag))
			}
		}

		return errors.New(strings.Join(errorMessages, ", "))
	}

	return nil
}

func ValidateUUID(id string, paramName string) error {
	err := validate.Var(id, "required,uuid")
	if err != nil {
		return fmt.Errorf("kolom '%s' gagal pada validasi 'uuid'", paramName)
	}
	return nil
}
