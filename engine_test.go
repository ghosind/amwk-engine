package engine_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/go-amwk/core"
)

type Application struct {
	handlers []core.HandlerFunc
}

func NewApplication() *Application {
	return &Application{
		handlers: make([]core.HandlerFunc, 0),
	}
}

func (a *Application) Start() error { return nil }
func (a *Application) Use(handlers ...core.HandlerFunc) core.Application {
	a.handlers = append(a.handlers, handlers...)
	return a
}
func (a *Application) Close() error                       { return nil }
func (a *Application) Shutdown(ctx context.Context) error { return nil }

type Request struct {
	req      *http.Request
	resource string
}

func NewRequest(req *http.Request) *Request {
	if req == nil {
		req = httptest.NewRequest(http.MethodGet, "/", nil)
	}
	return &Request{
		req: req,
	}
}

func (r *Request) Application() core.Application { return nil }
func (r *Request) Body() (io.ReadCloser, error)  { return r.req.Body, nil }
func (r *Request) ClientIP() string {
	ip, _, err := net.SplitHostPort(r.req.RemoteAddr)
	if err != nil {
		return r.req.RemoteAddr
	}
	return ip
}
func (r *Request) ContentLength() int64                     { return r.req.ContentLength }
func (r *Request) Cookie(name string) (*http.Cookie, error) { return r.req.Cookie(name) }
func (r *Request) Cookies() []*http.Cookie                  { return r.req.Cookies() }
func (r *Request) Context() context.Context                 { return r.req.Context() }
func (r *Request) Header(name string) string                { return r.req.Header.Get(name) }
func (r *Request) HeaderValues(name string) []string        { return r.req.Header.Values(name) }
func (r *Request) Headers() http.Header                     { return r.req.Header }
func (r *Request) Method() string                           { return r.req.Method }
func (r *Request) Protocol() string                         { return r.req.Proto }
func (r *Request) Path() string                             { return r.req.URL.Path }
func (r *Request) PathValue(key string) string              { return r.req.PathValue(key) }
func (r *Request) SetPathValue(key, value string)           { r.req.SetPathValue(key, value) }
func (r *Request) Resource() string                         { return r.resource }
func (r *Request) SetResource(resource string)              { r.resource = resource }
func (r *Request) Query(key string) string                  { return r.req.URL.Query().Get(key) }
func (r *Request) QueryValues(key string) []string          { return r.req.URL.Query()[key] }
func (r *Request) Queries() url.Values                      { return r.req.URL.Query() }
func (r *Request) Request() any                             { return r }

type Response struct {
	rw   http.ResponseWriter
	body *bytes.Buffer
	code int
}

func NewResponse(rw http.ResponseWriter) *Response {
	if rw == nil {
		rw = httptest.NewRecorder()
	}
	return &Response{
		rw:   rw,
		body: &bytes.Buffer{},
	}
}

func (r *Response) Application() core.Application { return nil }
func (r *Response) AddHeader(key, value string)   { r.rw.Header().Add(key, value) }
func (r *Response) SetHeader(key, value string)   { r.rw.Header().Set(key, value) }
func (r *Response) GetHeader(key string) string   { return r.rw.Header().Get(key) }
func (r *Response) DelHeader(key string)          { r.rw.Header().Del(key) }
func (r *Response) Headers() http.Header          { return r.rw.Header() }
func (r *Response) Write(d []byte) (int, error) {
	return r.body.Write(d)
}
func (r *Response) Status(code int) { r.code = code }
func (r *Response) StatusCode() int { return r.code }
func (r *Response) Response() any   { return r }
func (r *Response) send() {
	if r.code != 0 {
		r.rw.WriteHeader(r.code)
	} else {
		r.rw.WriteHeader(http.StatusOK)
	}

	r.rw.Write(r.body.Bytes())
}
