package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func makeCtx() *Context {
	var fctx fasthttp.RequestCtx
	return NewContext(&fctx, nil, nil)
}

func TestNewContext_ReusesPool(t *testing.T) {
	c1 := makeCtx()
	c1.SetSession("key", "val")
	ReleaseContext(c1)

	var fctx2 fasthttp.RequestCtx
	c2 := NewContext(&fctx2, nil, nil)
	assert.Nil(t, c2.GetSession("key"))
	assert.Equal(t, &fctx2, c2.Ctx)
	ReleaseContext(c2)
}

func TestNewContext_ClearsParams(t *testing.T) {
	c := makeCtx()
	c.SetParam("name", "bob")
	ReleaseContext(c)

	c2 := makeCtx()
	assert.Equal(t, "", c2.Param("name"))
}

func TestContext_Params(t *testing.T) {
	c := makeCtx()
	p := c.Params()
	assert.NotNil(t, p)
	p["id"] = "42"

	assert.Equal(t, "42", c.Param("id"))
}

func TestContext_SetParamAndParam(t *testing.T) {
	c := makeCtx()
	c.SetParam("name", "alice")
	assert.Equal(t, "alice", c.Param("name"))
}

func TestContext_Param_Missing(t *testing.T) {
	c := makeCtx()
	assert.Equal(t, "", c.Param("nonexistent"))
}

func TestContext_GetAllParams(t *testing.T) {
	c := makeCtx()
	c.SetParam("a", "1")
	c.SetParam("b", "2")

	all := c.GetAllParams()
	assert.Equal(t, "1", all["a"])
	assert.Equal(t, "2", all["b"])
	assert.Len(t, all, 2)
}

func TestContext_Next_ExecutesHandlersInOrder(t *testing.T) {
	var order []int
	h1 := func(c *Context) { order = append(order, 1); c.Next() }
	h2 := func(c *Context) { order = append(order, 2); c.Next() }
	h3 := func(c *Context) { order = append(order, 3) }

	c := makeCtx()
	c.SetHandlers([]Handler{h1, h2, h3})
	c.Next()

	assert.Equal(t, []int{1, 2, 3}, order)
}

func TestContext_Next_AbortStopsChain(t *testing.T) {
	var order []int
	h1 := func(c *Context) { order = append(order, 1); c.Next() }
	h2 := func(c *Context) { order = append(order, 2); c.Abort() }
	h3 := func(c *Context) { order = append(order, 3) }

	c := makeCtx()
	c.SetHandlers([]Handler{h1, h2, h3})
	c.Next()

	assert.Equal(t, []int{1, 2}, order)
}

func TestContext_AbortAndAborted(t *testing.T) {
	c := makeCtx()
	assert.False(t, c.Aborted())
	c.Abort()
	assert.True(t, c.Aborted())
}

func TestContext_JSON_WritesResponse(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := NewContext(&fctx, nil, nil)
	c.JSON(200, map[string]string{"msg": "ok"})

	assert.Equal(t, 200, fctx.Response.StatusCode())
	assert.Equal(t, "application/json", string(fctx.Response.Header.ContentType()))

	var body map[string]string
	json.Unmarshal(fctx.Response.Body(), &body)
	assert.Equal(t, "ok", body["msg"])
}

func TestContext_JSON_MarshalError(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := NewContext(&fctx, nil, nil)
	c.JSON(200, make(chan int))

	assert.Equal(t, fasthttp.StatusInternalServerError, fctx.Response.StatusCode())
}

func TestContext_Text(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := NewContext(&fctx, nil, nil)
	c.Text(404, "not found")

	assert.Equal(t, 404, fctx.Response.StatusCode())
	assert.Equal(t, "text/plain", string(fctx.Response.Header.ContentType()))
	assert.Equal(t, "not found", string(fctx.Response.Body()))
}

func TestContext_StatusAndSendString(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := NewContext(&fctx, nil, nil)
	c.Status(201).SendString("created")

	assert.Equal(t, 201, fctx.Response.StatusCode())
	assert.Equal(t, "created", string(fctx.Response.Body()))
}

func TestContext_Session_SetAndGet(t *testing.T) {
	c := makeCtx()
	c.SetSession("user", "alice")
	assert.Equal(t, "alice", c.GetSession("user"))
}

func TestContext_Session_Missing(t *testing.T) {
	c := makeCtx()
	assert.Nil(t, c.GetSession("nobody"))
}

func TestContext_Session_IsClearedOnRelease(t *testing.T) {
	c := makeCtx()
	c.SetSession("secret", "value")
	ReleaseContext(c)

	c2 := makeCtx()
	assert.Nil(t, c2.GetSession("secret"))
	ReleaseContext(c2)
}

func TestContext_SetHandlers(t *testing.T) {
	c := makeCtx()
	h1 := func(c *Context) {}
	h2 := func(c *Context) {}

	c.SetHandlers([]Handler{h1, h2})
	assert.Equal(t, int32(-1), c.index.Load())
}

func TestContext_SetQueueAndMustQueue(t *testing.T) {
	c := makeCtx()
	assert.Panics(t, func() { c.MustQueue() })
}

func TestContext_SetPubsubAndMustPubsub(t *testing.T) {
	c := makeCtx()
	assert.Panics(t, func() { c.MustPubsub() })
}

func TestContext_Header(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("X-Custom", "val")
	c := NewContext(&fctx, nil, nil)

	assert.Equal(t, "val", c.Header("X-Custom"))
}

func TestContext_Query(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.SetRequestURI("/test?q=hello")
	c := NewContext(&fctx, nil, nil)

	assert.Equal(t, "hello", c.Query("q"))
}

func TestContext_BindJSON(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.SetBodyString(`{"name":"alice"}`)
	c := NewContext(&fctx, nil, nil)

	var dest struct {
		Name string `json:"name"`
	}
	err := c.BindJSON(&dest)
	assert.NoError(t, err)
	assert.Equal(t, "alice", dest.Name)
}

func TestContext_ChainMethods(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := NewContext(&fctx, nil, nil)
	result := c.Status(200).SendString("ok")

	assert.Equal(t, c, result)
}

func TestNewEngineContext(t *testing.T) {
	ec := NewEngineContext()
	assert.NotNil(t, ec)
}

func TestContext_Writer_Reused(t *testing.T) {
	c1 := makeCtx()
	c1.Writer.WriteString("hello")
	ReleaseContext(c1)

	c2 := makeCtx()
	assert.Equal(t, 0, c2.Writer.Len())
}

func TestContext_SlotsReadPriority(t *testing.T) {
	c := makeCtx()
	c.pslots[0] = ParamSlot{K: "id", V: "42"}
	c.nparams = 1
	assert.Equal(t, "42", c.Param("id"))
	assert.Equal(t, "", c.Param("missing"))
}

func TestContext_SetParamOverridesSlot(t *testing.T) {
	c := makeCtx()
	c.pslots[0] = ParamSlot{K: "id", V: "42"}
	c.nparams = 1
	c.SetParam("id", "99")
	assert.Equal(t, "99", c.Param("id"))
	// Map materialization must prefer the override.
	assert.Equal(t, "99", c.Params()["id"])
}

func TestContext_ParamsMaterializesSlots(t *testing.T) {
	c := makeCtx()
	c.pslots[0] = ParamSlot{K: "a", V: "1"}
	c.pslots[1] = ParamSlot{K: "b", V: "2"}
	c.nparams = 2
	p := c.Params()
	assert.Equal(t, "1", p["a"])
	assert.Equal(t, "2", p["b"])
	// Idempotent: second call keeps values.
	assert.Equal(t, "1", c.Params()["a"])
}

func TestContext_SetParamsClearsSlots(t *testing.T) {
	c := makeCtx()
	c.pslots[0] = ParamSlot{K: "id", V: "42"}
	c.nparams = 1
	c.SetParams(map[string]string{"x": "1"})
	assert.Equal(t, "", c.Param("id"))
	assert.Equal(t, "1", c.Param("x"))
}

func TestContext_GetAllParamsMergesSlots(t *testing.T) {
	c := makeCtx()
	c.pslots[0] = ParamSlot{K: "a", V: "1"}
	c.nparams = 1
	c.SetParam("b", "2")
	all := c.GetAllParams()
	assert.Equal(t, "1", all["a"])
	assert.Equal(t, "2", all["b"])
	assert.Len(t, all, 2)
}

func TestNewContext_ResetsSlots(t *testing.T) {
	c := makeCtx()
	c.pslots[0] = ParamSlot{K: "id", V: "42"}
	c.nparams = 1
	ReleaseContext(c)
	c2 := makeCtx()
	assert.Equal(t, "", c2.Param("id"))
	assert.Equal(t, 0, len(c2.GetAllParams()))
	ReleaseContext(c2)
}
