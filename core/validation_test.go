package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestBind_AutoDetectJSON(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetContentType("application/json")
	fctx.Request.SetBodyString(`{"name":"bob","age":25}`)
	c := NewContext(&fctx, nil, nil)
	ReleaseContext(c)

	c2 := NewContext(&fctx, nil, nil)
	type payload struct {
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age"`
	}
	var p payload
	err := c2.Bind(&p)
	assert.NoError(t, err)
	assert.Equal(t, "bob", p.Name)
	assert.Equal(t, 25, p.Age)
}

func TestBind_AutoDetectForm(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetContentType("application/x-www-form-urlencoded")
	fctx.Request.SetBodyString("name=alice&age=30")
	c := NewContext(&fctx, nil, nil)

	type payload struct {
		Name string `form:"name"`
		Age  int    `form:"age"`
	}
	var p payload
	err := c.Bind(&p)
	assert.NoError(t, err)
	assert.Equal(t, "alice", p.Name)
	assert.Equal(t, 30, p.Age)
}

func TestBind_AutoDetectQuery(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.SetRequestURI("/search?q=golang&page=2")
	c := NewContext(&fctx, nil, nil)

	type payload struct {
		Q    string `query:"q"`
		Page int    `query:"page"`
	}
	var p payload
	err := c.Bind(&p)
	assert.NoError(t, err)
	assert.Equal(t, "golang", p.Q)
	assert.Equal(t, 2, p.Page)
}

func TestBind_ValidationFails(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetContentType("application/json")
	fctx.Request.SetBodyString(`{"name":""}`)
	c := NewContext(&fctx, nil, nil)

	type payload struct {
		Name string `json:"name" validate:"required"`
	}
	var p payload
	err := c.Bind(&p)
	assert.Error(t, err)

	verr, ok := err.(*ValidationError)
	assert.True(t, ok)
	assert.Len(t, verr.Errors, 1)
	assert.Equal(t, "name", verr.Errors[0].Field)
	assert.Equal(t, "required", verr.Errors[0].Tag)
}

func TestBind_ValidationMultipleErrors(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetContentType("application/json")
	fctx.Request.SetBodyString(`{"name":"","email":"not-an-email"}`)
	c := NewContext(&fctx, nil, nil)

	type payload struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"email"`
	}
	var p payload
	err := c.Bind(&p)
	assert.Error(t, err)

	verr := err.(*ValidationError)
	assert.Len(t, verr.Errors, 2)

	found := map[string]string{}
	for _, fe := range verr.Errors {
		found[fe.Field] = fe.Tag
	}
	assert.Equal(t, "required", found["name"])
	assert.Equal(t, "email", found["email"])
}

func TestBind_ValidateCustomRule(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetContentType("application/json")
	fctx.Request.SetBodyString(`{"age":15}`)
	c := NewContext(&fctx, nil, nil)

	type payload struct {
		Age int `json:"age" validate:"gte=18"`
	}
	var p payload
	err := c.Bind(&p)
	assert.Error(t, err)

	verr := err.(*ValidationError)
	assert.Len(t, verr.Errors, 1)
	assert.Equal(t, "age", verr.Errors[0].Field)
	assert.Equal(t, "gte", verr.Errors[0].Tag)
	assert.Equal(t, "18", verr.Errors[0].Param)
	assert.Equal(t, 15, verr.Errors[0].Value)
}

func TestBind_NestedStructValidation(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetContentType("application/json")
	fctx.Request.SetBodyString(`{"user":{"name":""}}`)
	c := NewContext(&fctx, nil, nil)

	type user struct {
		Name string `json:"name" validate:"required"`
	}
	type payload struct {
		User user `json:"user" validate:"required"`
	}
	var p payload
	err := c.Bind(&p)
	assert.Error(t, err)

	verr := err.(*ValidationError)
	assert.Len(t, verr.Errors, 1)
	assert.Equal(t, "name", verr.Errors[0].Field)
}

func TestValidate_NonStructSkipped(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := NewContext(&fctx, nil, nil)

	m := map[string]string{}
	assert.NoError(t, c.Validate(&m))
}

func TestBind_NilTarget(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := NewContext(&fctx, nil, nil)

	err := c.Bind(nil)
	assert.Error(t, err)
}
