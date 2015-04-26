package zincir

import (
	"github.com/codegangsta/negroni"
	"github.com/julienschmidt/httprouter"
	"github.com/nbio/httpcontext"
	"gopkg.in/tylerb/graceful.v1"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type HandlerType interface{}

type Zincir struct {
	Engine *negroni.Negroni
	Router *httprouter.Router
	logger *log.Logger
}

func New() *Zincir {
	var zincir = new(Zincir)
	zincir.Engine = negroni.New()
	zincir.logger = log.New(os.Stdout, "[zincir] ", 0)
	return zincir
}

func (z *Zincir) Route(method string, path string, handler HandlerType) {
	if z.Router == nil {
		z.Router = httprouter.New()

		z.Engine.UseFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			if handle, p, _ := z.Router.Lookup(r.Method, r.URL.Path); handle != nil {
				httpcontext.Set(r, "zincir-nextFunc", next)
				handle(rw, r, p)
			} else {
				next(rw, r)
			}
		})
	}

	h := Wrap(handler)

	z.Router.Handle(method, path, func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		nextFunc := httpcontext.Get(r, "zincir-nextFunc").(http.HandlerFunc)
		httpcontext.Set(r, "zincir-params", p)
		h.ServeHTTP(rw, r, nextFunc)
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
	z.Engine.Use(Wrap(handler))
}

func (z *Zincir) Mount(prefix string, handler HandlerType) {
	h := Wrap(handler)

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

func Param(r *http.Request, key string) string {
	if p, ok := httpcontext.GetOk(r, "zincir-params"); ok {
		if pp, pok := p.(httprouter.Params); pok {
			return pp.ByName(key)
		}
	}

	return ""
}

func SetVar(r *http.Request, key string, value interface{}) {
	httpcontext.Set(r, "_"+key, value)
}

func GetVar(r *http.Request, key string) interface{} {
	return httpcontext.Get(r, key)
}

func GetVarString(r *http.Request, key string) string {
	return httpcontext.GetString(r, key)
}

func GetVarOk(r *http.Request, key string) (interface{}, bool) {
	return httpcontext.GetOk(r, key)
}

func DelVar(r *http.Request, key string) {
	httpcontext.Delete(r, key)
}

func ClearVars(r *http.Request) {
	httpcontext.Clear(r)
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

func Wrap(h HandlerType) negroni.Handler {
	switch f := h.(type) {
	case http.Handler:
		return negroni.Wrap(f)
	case negroni.Handler:
		return f
	case func(w http.ResponseWriter, r *http.Request):
		return negroni.Wrap(http.HandlerFunc(f))
	case func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc):
		return negroni.HandlerFunc(f)
	default:
		log.Fatalf("Unexpected handler type: %T", f)
		panic("Unexpected handler type.")
	}
}
