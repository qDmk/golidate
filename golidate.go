package golidate

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrNotStruct = errors.New("wrong argument given, should be a struct")
var ErrInvalidValidatorSyntax = errors.New("invalid validator syntax")
var ErrValidateForUnexportedFields = errors.New("validation for unexported field is not allowed")

type ValidationError struct {
	Err error
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 1 {
		return v[0].Err.Error()
	}

	res := ""
	for _, err := range v {
		res += err.Err.Error() + "\n"
	}

	return res
}

func splitArgs(str string) (v string, args []string, err ValidationError) {
	v, unsplitArgs, found := strings.Cut(str, ":")
	if !found {
		err = ValidationError{ErrInvalidValidatorSyntax}
		return
	}

	if unsplitArgs == "" {
		args = []string{}
	} else {
		args = strings.Split(unsplitArgs, ",")
	}
	return
}

type validator func(v reflect.Value, args []string) ValidationError
type intArgValidator func(v reflect.Value, arg int) ValidationError

func fromIntArgValidator(val intArgValidator) validator {
	return func(v reflect.Value, args []string) ValidationError {
		if len(args) != 1 {
			return ValidationError{ErrInvalidValidatorSyntax}
		}

		arg, convErr := strconv.Atoi(args[0])
		if convErr != nil {
			return ValidationError{ErrInvalidValidatorSyntax}
		}

		return val(v, arg)
	}
}

func getValidator(valName string) (val validator, err ValidationError) {
	switch valName {
	case "non-empty":
		val = fromIntArgValidator(func(v reflect.Value, arg int) ValidationError {
			if arg < 1 || v.Kind() != reflect.String {
				return ValidationError{ErrInvalidValidatorSyntax}
			}

			chars := utf8.RuneCountInString(v.String())
			if chars == 0 || chars > arg {
				return ValidationError{errors.New("validator non-empty")}
			}

			return ValidationError{}
		})

	default:
		err = ValidationError{ErrInvalidValidatorSyntax}
	}

	return
}

func Validate(v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	var errs ValidationErrors = []ValidationError{}
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		field := val.Field(i)

		valStr, ok := typeField.Tag.Lookup("validate")
		if !ok {
			continue
		}

		if !typeField.IsExported() {
			errs = append(errs, ValidationError{ErrValidateForUnexportedFields})
			continue
		}

		valName, args, err := splitArgs(valStr)
		if err.Err != nil {
			errs = append(errs, err)
			continue
		}

		validator, err := getValidator(valName)
		if err.Err != nil {
			errs = append(errs, err)
			continue
		}

		err = validator(field, args)
		if err.Err != nil {
			errs = append(errs, err)
		}

	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
