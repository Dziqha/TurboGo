package core

import (
	"github.com/Dziqha/TurboGo/internal/kafka"
	"github.com/Dziqha/TurboGo/internal/rabbitmq"
	"github.com/Dziqha/TurboGo/internal/redis"
	"encoding/json"

	"github.com/valyala/fasthttp"
)

type Context struct {
	Ctx   *fasthttp.RequestCtx
	Redis *redis.Engine
	Kafka *kafka.Engine
	Queue *rabbitmq.Engine
	index    int
	handlers []Handler
	aborted bool

}

func NewContext(ctx *fasthttp.RequestCtx, redis *redis.Engine, kafka *kafka.Engine, queue *rabbitmq.Engine, handlers []Handler) *Context {
	return &Context{
		Ctx:      ctx,
		Redis:    redis,
		Kafka:    kafka,
		Queue:    queue,
		handlers: handlers,
		index:    -1, // ⬅️  agar Next() mulai dari 0
	}
}


// JSON writes a JSON response
func (c *Context) JSON(code int, data any) {
	c.Ctx.SetStatusCode(code)
	c.Ctx.SetContentType("application/json")
	res, err := json.Marshal(data)
	if err != nil {
		c.Ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		c.Ctx.SetBodyString(`{"error":"failed to marshal json"}`)
		return
	}
	c.Ctx.SetBody(res)
}

// Text writes a plain text response
func (c *Context) Text(code int, msg string) {
	c.Ctx.SetStatusCode(code)
	c.Ctx.SetContentType("text/plain")
	c.Ctx.SetBodyString(msg)
}

// Param returns a URL parameter (from user-defined routing system)
func (c *Context) Param(key string) string {
	// implementasi tergantung routing kamu
	val := c.Ctx.UserValue(key)
	if val == nil {
		return ""
	}
	return val.(string)
}

// Query returns a query parameter
func (c *Context) Query(key string) string {
	return string(c.Ctx.QueryArgs().Peek(key))
}

// Body binds JSON body to a struct
func (c *Context) BindJSON(dest any) error {
	return json.Unmarshal(c.Ctx.PostBody(), dest)
}

func (c *Context) Header(key string) string {
    return string(c.Ctx.Request.Header.Peek(key))
}

func (c *Context) Next() {
	c.index++
	if c.index < len(c.handlers) && !c.aborted {
		c.handlers[c.index](c)
	}
}

func (c *Context) Abort() {
	c.aborted = true
}


