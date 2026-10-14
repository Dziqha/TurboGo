package core

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func init() {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		for _, tag := range []string{"json", "form", "query"} {
			if v, ok := fld.Tag.Lookup(tag); ok {
				return strings.Split(v, ",")[0]
			}
		}
		return fld.Name
	})
}

type FieldError struct {
	Field string
	Tag   string
	Param string
	Value any
}

type ValidationError struct {
	Errors []FieldError
}

func (e *ValidationError) Error() string {
	var parts []string
	for _, fe := range e.Errors {
		parts = append(parts, fmt.Sprintf("%s: %s", fe.Field, fe.Tag))
	}
	return "validation failed: " + strings.Join(parts, ", ")
}

func (c *Context) Bind(dest any) error {
	if err := c.bind(dest); err != nil {
		return err
	}
	return c.Validate(dest)
}

func (c *Context) bind(dest any) error {
	contentType := string(c.Ctx.Request.Header.ContentType())
	switch {
	case strings.Contains(contentType, "application/json"):
		return c.BindJSON(dest)
	case strings.Contains(contentType, "application/x-www-form-urlencoded"),
		strings.Contains(contentType, "multipart/form-data"):
		return c.BindForm(dest)
	default:
		return c.BindQuery(dest)
	}
}

func (c *Context) Validate(dest any) error {
	rv := reflect.ValueOf(dest)
	if !rv.IsValid() {
		return errors.New("validate target must be a non-nil struct pointer")
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return errors.New("validate target must be a non-nil struct pointer")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	err := validate.Struct(dest)
	if err == nil {
		return nil
	}

	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return invalid
	}

	verr := &ValidationError{}
	var vErrors validator.ValidationErrors
	if errors.As(err, &vErrors) {
		for _, ve := range vErrors {
			verr.Errors = append(verr.Errors, FieldError{
				Field: ve.Field(),
				Tag:   ve.Tag(),
				Param: ve.Param(),
				Value: ve.Value(),
			})
		}
		return verr
	}

	return err
}
