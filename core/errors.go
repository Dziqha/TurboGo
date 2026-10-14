package core

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

type HTTPError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func NewHTTPError(status int, code, message string) *HTTPError {
	return &HTTPError{Status: status, Code: code, Message: message}
}

func (e *HTTPError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

func (e *HTTPError) WithDetails(details any) *HTTPError {
	e.Details = details
	return e
}

var (
	ErrBadRequest   = NewHTTPError(fasthttp.StatusBadRequest, "BAD_REQUEST", "bad request")
	ErrUnauthorized = NewHTTPError(fasthttp.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
	ErrForbidden    = NewHTTPError(fasthttp.StatusForbidden, "FORBIDDEN", "forbidden")
	ErrNotFound     = NewHTTPError(fasthttp.StatusNotFound, "NOT_FOUND", "not found")
	ErrConflict     = NewHTTPError(fasthttp.StatusConflict, "CONFLICT", "conflict")
	ErrInternal     = NewHTTPError(fasthttp.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
)

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
	Fields  any    `json:"fields,omitempty"`
}

func (c *Context) Error(err error) {
	if err == nil {
		return
	}
	c.Abort()

	switch e := err.(type) {
	case *HTTPError:
		if e.Status == 0 {
			e.Status = fasthttp.StatusInternalServerError
		}
		if e.Code == "" {
			e.Code = "ERROR"
		}
		if e.Message == "" {
			e.Message = e.Error()
		}
		c.JSON(e.Status, errorResponse{Error: errorBody{
			Code:    e.Code,
			Message: e.Message,
			Details: e.Details,
		}})
	case *ValidationError:
		fields := make([]map[string]string, 0, len(e.Errors))
		for _, fe := range e.Errors {
			f := map[string]string{"field": fe.Field, "tag": fe.Tag}
			if fe.Param != "" {
				f["param"] = fe.Param
			}
			fields = append(fields, f)
		}
		c.JSON(fasthttp.StatusBadRequest, errorResponse{Error: errorBody{
			Code:   "VALIDATION_ERROR",
			Message: "validation failed",
			Fields: fields,
		}})
	default:
		c.JSON(fasthttp.StatusInternalServerError, errorResponse{Error: errorBody{
			Code:    "INTERNAL_ERROR",
			Message: "internal server error",
		}})
	}
}
