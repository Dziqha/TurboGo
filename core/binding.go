package core

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func (c *Context) BindQuery(dest any) error {
	args := c.Ctx.QueryArgs()
	return bindStruct(dest, func(name string) []string {
		var vals []string
		args.VisitAll(func(k, v []byte) {
			if string(k) == name {
				vals = append(vals, string(v))
			}
		})
		return vals
	})
}

func (c *Context) BindForm(dest any) error {
	return bindStruct(dest, func(name string) []string {
		form, err := c.Ctx.MultipartForm()
		if err == nil && form != nil {
			if vals, ok := form.Value[name]; ok {
				return vals
			}
		}
		var vals []string
		c.Ctx.PostArgs().VisitAll(func(k, v []byte) {
			if string(k) == name {
				vals = append(vals, string(v))
			}
		})
		return vals
	})
}

func bindStruct(dest any, get func(name string) []string) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return errors.New("bind target must be a non-nil struct pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errors.New("bind target must point to a struct")
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := bindStruct(v.Field(i).Addr().Interface(), get); err != nil {
				return err
			}
			continue
		}

		name := fieldName(field)
		if name == "" || name == "-" {
			continue
		}

		vals := get(name)
		if len(vals) == 0 {
			continue
		}

		if err := setField(v.Field(i), vals); err != nil {
			return fmt.Errorf("field %q: %w", field.Name, err)
		}
	}
	return nil
}

func fieldName(f reflect.StructField) string {
	for _, tag := range []string{"query", "form", "json"} {
		if v, ok := f.Tag.Lookup(tag); ok {
			return strings.Split(v, ",")[0]
		}
	}
	return f.Name
}

func setField(fv reflect.Value, vals []string) error {
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		fv = fv.Elem()
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(vals[0])
		return nil
	case reflect.Bool:
		b, err := strconv.ParseBool(vals[0])
		if err != nil {
			return err
		}
		fv.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(vals[0], 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(vals[0], 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(vals[0], 64)
		if err != nil {
			return err
		}
		fv.SetFloat(f)
		return nil
	case reflect.Slice:
		if fv.Type().Elem().Kind() == reflect.String {
			fv.Set(reflect.ValueOf(append([]string(nil), vals...)))
			return nil
		}
		slice := reflect.MakeSlice(fv.Type(), 0, len(vals))
		for _, s := range vals {
			el := reflect.New(fv.Type().Elem()).Elem()
			if err := setField(el, []string{s}); err != nil {
				return err
			}
			slice = reflect.Append(slice, el)
		}
		fv.Set(slice)
		return nil
	default:
		return fmt.Errorf("unsupported field type %s", fv.Kind())
	}
}