package zincir

import (
	"github.com/codegangsta/negroni"
	"github.com/julienschmidt/httprouter"
	"github.com/unrolled/render"
	"gopkg.in/tylerb/graceful.v1"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type HandlerType interface{}
type NextFunc func()
type RenderOptions render.Options

type Options struct {
	Render RenderOptions
}

type Zincir struct {
	Engine  *negroni.Negroni
	Router  *httprouter.Router
	Render  *render.Render
	logger  *log.Logger
	options Options
}

func New(options ...Options) *Zincir {
	var zincir = new(Zincir)

	if len(options) > 0 {
		zincir.options = options[0]
	}

	zincir.Engine = negroni.New()
	zincir.logger = log.New(os.Stdout, "[zincir] ", 0)
	zincir.Render = render.New(render.Options(zincir.options.Render))
	return zincir
}

func (z *Zincir) Route(method string, path string, handler HandlerType) {
	if z.Router == nil {
		z.Router = httprouter.New()

		z.Engine.UseFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			if handle, p, _ := z.Router.Lookup(r.Method, r.URL.Path); handle != nil {
				ctx := z.Context(rw, r)
				ctx.params = p
				ctx.nextFunc = next

				handle(rw, r, p)
			} else {
				next(rw, r)
			}
		})
	}

	h := z.Wrap(handler)

	z.Router.Handle(method, path, func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		h.ServeHTTP(rw, r, z.Context(rw, r).nextFunc)
	})
}

func (z *Zincir) GET(path string, handler HandlerType) {
	z.Route("GET", path, handler)
}

func (z *Zincir) POST(path string, handler HandlerType) {
	z.Route("POST", path, handler)
}

func (z *Zincir) HEAD(path string, handler HandlerType) {
	z.Route("HEAD", path, handler)
}

func (z *Zincir) OPTIONS(path string, handler HandlerType) {
	z.Route("OPTIONS", path, handler)
}

func (z *Zincir) PUT(path string, handler HandlerType) {
	z.Route("PUT", path, handler)
}

func (z *Zincir) PATCH(path string, handler HandlerType) {
	z.Route("PATCH", path, handler)
}

func (z *Zincir) DELETE(path string, handler HandlerType) {
	z.Route("DELETE", path, handler)
}

func (z *Zincir) Use(handler HandlerType) {
	z.Engine.Use(z.Wrap(handler))
}

func (z *Zincir) Mount(prefix string, handler HandlerType) {
	h := z.Wrap(handler)

	z.Engine.UseFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if p := strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			oldPath := r.URL.Path
			r.URL.Path = p

			h.ServeHTTP(rw, r, func(w http.ResponseWriter, r *http.Request) {
				r.URL.Path = oldPath
				next(rw, r)
			})
		} else {
			next(rw, r)
		}
	})
}

func (z *Zincir) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	z.Engine.ServeHTTP(rw, r)
}

func (z *Zincir) Run(addr string) {
	z.logger.Printf("listening on %s", addr)
	z.logger.Fatal(http.ListenAndServe(addr, z))
}

func (z *Zincir) RunGraceful(addr string) {
	z.logger.Printf("listening gracefully on %s", addr)
	graceful.Run(addr, 10*time.Second, z)
}

func (z *Zincir) Wrap(h HandlerType) negroni.Handler {
	switch f := h.(type) {
	case http.Handler:
		return negroni.Wrap(f)
	case negroni.Handler:
		return f
	case func(w http.ResponseWriter, r *http.Request):
		return negroni.Wrap(http.HandlerFunc(f))
	case func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc):
		return negroni.HandlerFunc(f)
	case func(rw http.ResponseWriter, r *http.Request, next NextFunc):
		return negroni.HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			nf := func() {
				next(rw, r)
			}

			f(rw, r, nf)
		})
	case func(c *Ctx):
		return negroni.HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			f(z.Context(rw, r))
			next(rw, r)
		})
	case func(c *Ctx, next http.HandlerFunc):
		return negroni.HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			f(z.Context(rw, r), next)
		})
	case func(c *Ctx, next NextFunc):
		return negroni.HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			nf := func() {
				next(rw, r)
			}

			f(z.Context(rw, r), nf)
		})
	case func(http.Handler) http.Handler:
		nextCaller := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			z.Context(rw, r).nextFunc(rw, r)
		})

		handler := f(nextCaller)

		return negroni.HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			z.Context(rw, r).nextFunc = next
			handler.ServeHTTP(rw, r)
		})
	default:
		log.Fatalf("Unexpected handler type: %T", f)
		panic("Unexpected handler type.")
	}
}

func NewStatic(dir http.FileSystem) *negroni.Static {
	return negroni.NewStatic(dir)
}

func NewLogger() *negroni.Logger {
	return negroni.NewLogger()
}

func NewRecovery() *negroni.Recovery {
	return negroni.NewRecovery()
}
