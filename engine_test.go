package engine_test

import (
	"context"
	"io"
	"net/http"
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
	body          io.ReadCloser
	clientIP      string
	contentLength int64
	cookies       []*http.Cookie
	ctx           context.Context
	headers       http.Header
	method        string
	protocol      string
	path          string
	pathValues    map[string]string
	resource      string
	queryValues   url.Values
}

func NewRequest() *Request {
	return &Request{
		body:          nil,
		clientIP:      "127.0.0.1",
		contentLength: 123,
		cookies:       []*http.Cookie{},
		ctx:           context.Background(),
		headers:       make(http.Header),
		method:        "GET",
		protocol:      "HTTP/1.1",
		path:          "/test",
		pathValues:    make(map[string]string),
		queryValues:   make(url.Values),
	}
}

func (r *Request) Body() (io.ReadCloser, error) { return r.body, nil }
func (r *Request) ClientIP() string             { return r.clientIP }
func (r *Request) ContentLength() int64         { return r.contentLength }
func (r *Request) Cookie(name string) (*http.Cookie, error) {
	for _, c := range r.cookies {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, http.ErrNoCookie
}
func (r *Request) Cookies() []*http.Cookie           { return r.cookies }
func (r *Request) Context() context.Context          { return r.ctx }
func (r *Request) Header(name string) string         { return r.headers.Get(name) }
func (r *Request) HeaderValues(name string) []string { return r.headers.Values(name) }
func (r *Request) Headers() http.Header              { return r.headers }
func (r *Request) Method() string                    { return r.method }
func (r *Request) Protocol() string                  { return r.protocol }
func (r *Request) Path() string                      { return r.path }
func (r *Request) PathValue(key string) string       { return r.pathValues[key] }
func (r *Request) SetPathValue(key, value string) {
	r.pathValues[key] = value
}
func (r *Request) Resource() string                { return r.resource }
func (r *Request) SetResource(resource string)     { r.resource = resource }
func (r *Request) Query(key string) string         { return r.queryValues.Get(key) }
func (r *Request) QueryValues(key string) []string { return r.queryValues[key] }
func (r *Request) Queries() url.Values             { return r.queryValues }
func (r *Request) Request() any                    { return r }

type Response struct {
	headers    http.Header
	statusCode int
	body       []byte
}

func NewResponse() *Response {
	return &Response{
		headers:    make(http.Header),
		statusCode: 200,
		body:       make([]byte, 0),
	}
}

func (r *Response) AddHeader(key, value string) { r.headers.Add(key, value) }
func (r *Response) SetHeader(key, value string) { r.headers.Set(key, value) }
func (r *Response) GetHeader(key string) string { return r.headers.Get(key) }
func (r *Response) DelHeader(key string)        { r.headers.Del(key) }
func (r *Response) Headers() http.Header        { return r.headers }
func (r *Response) Write(d []byte) (int, error) {
	r.body = append(r.body, d...)
	return len(d), nil
}
func (r *Response) Status(code int) { r.statusCode = code }
func (r *Response) StatusCode() int { return r.statusCode }
func (r *Response) Response() any   { return r }
