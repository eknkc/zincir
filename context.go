package zincir

import (
	"github.com/julienschmidt/httprouter"
	"github.com/unrolled/render"
	"io"
	"net/http"
)

type H map[string]interface{}
type L []interface{}

type Ctx struct {
	Request  *http.Request
	Writer   http.ResponseWriter
	engine   *Zincir
	params   httprouter.Params
	vars     map[interface{}]interface{}
	nextFunc http.HandlerFunc
}

func (c *Ctx) Param(key string) string {
	if c.params == nil {
		return ""
	}

	return c.params.ByName(key)
}

func (c *Ctx) Get(key interface{}) interface{} {
	if c.vars == nil {
		return nil
	}

	return c.vars[key]
}

func (c *Ctx) Set(key interface{}, value interface{}) {
	if c.vars == nil {
		c.vars = make(map[interface{}]interface{})
	}

	c.vars[key] = value
}

func (c *Ctx) Del(key interface{}) {
	if c.vars != nil {
		delete(c.vars, key)
	}
}

func (c *Ctx) Render() *render.Render {
	return c.engine.render
}

func (c *Ctx) JSON(status int, v interface{}) {
	c.engine.render.JSON(c.Writer, status, v)
}

func (c *Ctx) JSONP(status int, cbname string, v interface{}) {
	c.engine.render.JSONP(c.Writer, status, cbname, v)
}

func (c *Ctx) XML(status int, v interface{}) {
	c.engine.render.XML(c.Writer, status, v)
}

func (c *Ctx) Data(status int, v []byte) {
	c.engine.render.Data(c.Writer, status, v)
}

func (c *Ctx) HTML(status int, name string, binding interface{}, htmlOpt ...render.HTMLOptions) {
	c.engine.render.HTML(c.Writer, status, name, binding, htmlOpt...)
}

func makeCtx(z *Zincir, w http.ResponseWriter, r *http.Request) *Ctx {
	ctx := new(Ctx)
	ctx.engine = z
	ctx.Request = r
	ctx.Writer = w
	return ctx
}

type CtxReadCloser interface {
	io.ReadCloser
	Context() *Ctx
}

type ctxReadCloser struct {
	io.ReadCloser
	context *Ctx
}

func (c *ctxReadCloser) Context() *Ctx {
	return c.context
}

func (z *Zincir) Context(w http.ResponseWriter, r *http.Request) *Ctx {
	ctx, ok := r.Body.(CtxReadCloser)
	if !ok {
		ctx = &ctxReadCloser{
			ReadCloser: r.Body,
			context:    makeCtx(z, w, r),
		}

		r.Body = ctx
	}
	return ctx.Context()
}
