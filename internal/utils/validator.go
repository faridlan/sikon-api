package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateStruct(data any) error {
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
				errorMessages = append(errorMessages, fmt.Sprintf("%s minimal harus %s", field, param))
			case "max":
				errorMessages = append(errorMessages, fmt.Sprintf("%s maksimal %s", field, param))
			case "oneof":
				errorMessages = append(errorMessages, fmt.Sprintf("%s harus salah satu dari: %s", field, param))
			case "uuid":
				errorMessages = append(errorMessages, fmt.Sprintf("%s harus berupa format UUID yang valid", field))
			case "gt":
				errorMessages = append(errorMessages, fmt.Sprintf("%s harus lebih besar dari %s", field, param))
			case "gte":
				errorMessages = append(errorMessages, fmt.Sprintf("%s harus lebih besar atau sama dengan %s", field, param))
			case "dive":
				errorMessages = append(errorMessages, fmt.Sprintf("terdapat kesalahan pada detail %s", field))
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
		return fmt.Errorf("kolom '%s' harus berupa UUID yang valid", paramName)
	}
	return nil
}
