package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/go-amwk/core"
)

type Context struct {
	app core.Application

	state    sync.Map
	index    int
	isAbort  bool
	handlers []core.HandlerFunc

	req core.Request
	res core.Response
}

func NewContext(app core.Application, req core.Request, res core.Response) *Context {
	ctx := new(Context)
	ctx.app = app
	ctx.index = -1
	ctx.isAbort = false
	ctx.req = req
	ctx.res = res
	ctx.state = sync.Map{}

	ctx.handlers = make([]core.HandlerFunc, 0)

	return ctx
}

// Get returns the value associated with the key in the context.
func (ctx *Context) Get(key string) (any, bool) {
	return ctx.state.Load(key)
}

// Set sets the value for the key in the context and returns the previous value.
func (ctx *Context) Set(key string, value any) any {
	oldValue, _ := ctx.state.Swap(key, value)
	return oldValue
}

// Context returns the Context associated with the request.
func (ctx *Context) Context() context.Context {
	return ctx.req.Context()
}

// Application returns the application instance associated with the context.
func (ctx *Context) Application() core.Application {
	return ctx.app
}

// Abort marks the context as aborted, and subsequent handlers will not be executed.
func (ctx *Context) Abort() {
	ctx.isAbort = true
}

// IsAbort checks if the context is marked as aborted.
func (ctx *Context) IsAbort() bool {
	return ctx.isAbort
}

// Next calls the next handler in the handlers chain.
func (ctx *Context) Next() error {
	ctx.index++
	for ctx.index < len(ctx.handlers) && !ctx.isAbort {
		handler := ctx.handlers[ctx.index]
		ctx.index++

		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					if e, ok := r.(error); ok {
						err = e
					} else {
						err = fmt.Errorf("panic: %v", r)
					}
				}
			}()
			err = handler(ctx)
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

// Use adds handlers to the context, which will be executed in the order they are added.
func (ctx *Context) Use(handlers ...core.HandlerFunc) {
	ctx.handlers = append(ctx.handlers, handlers...)
}

// Body returns the request body as a readable stream.
func (ctx *Context) Body() (io.ReadCloser, error) {
	return ctx.req.Body()
}

// ClientIP returns the IP address of the client making the request.
func (ctx *Context) ClientIP() string {
	proxyIp := ctx.Header("X-Forwarded-For")
	if proxyIp != "" {
		ips := strings.Split(proxyIp, ",")
		if len(ips) > 0 {
			proxyIp = strings.TrimSpace(ips[0])
			return proxyIp
		}
	}

	return ctx.req.ClientIP()
}

// ContentLength returns the length of the request body in bytes.
func (ctx *Context) ContentLength() int64 {
	return ctx.req.ContentLength()
}

// ContentType returns the Content-Type header of the request.
func (ctx *Context) ContentType() string {
	contentTypeHeader := ctx.Header("Content-Type")
	if contentTypeHeader == "" {
		return ""
	}

	contentType, _, _ := mime.ParseMediaType(contentTypeHeader)
	return contentType
}

// Cookie retrieves a cookie by name from the request.
func (ctx *Context) Cookie(name string) (*http.Cookie, error) {
	return ctx.req.Cookie(name)
}

// Cookies returns all cookies from the request.
func (ctx *Context) Cookies() []*http.Cookie {
	return ctx.req.Cookies()
}

// Header retrieves a header value by name from the request.
func (ctx *Context) Header(key string) string {
	return ctx.req.Header(key)
}

// HeaderValues retrieves all values for a header by name from the request.
func (ctx *Context) HeaderValues(key string) []string {
	return ctx.req.HeaderValues(key)
}

// Headers returns all headers from the request.
func (ctx *Context) Headers() http.Header {
	return ctx.req.Headers()
}

// Method returns the HTTP method of the request (e.g., GET, POST).
func (ctx *Context) Method() string {
	return ctx.req.Method()
}

// Protocol returns the HTTP protocol version of the request (e.g., HTTP/1.1).
func (ctx *Context) Protocol() string {
	return ctx.req.Protocol()
}

// Path returns the request path.
func (ctx *Context) Path() string {
	return ctx.req.Path()
}

// PathValue retrieves a path parameter value by name from the request.
func (ctx *Context) PathValue(name string) string {
	return ctx.req.PathValue(name)
}

// Resource returns the resource pattern of the request.
func (ctx *Context) Resource() string {
	return ctx.req.Resource()
}

// Query retrieves a query parameter value by name from the request.
func (ctx *Context) Query(key string) string {
	return ctx.req.Queries().Get(key)
}

// QueryValues retrieves all values for a query parameter by name from the request.
func (ctx *Context) QueryValues(key string) []string {
	return ctx.req.Queries()[key]
}

// Queries returns all query parameters from the request as a url.Values.
func (ctx *Context) Queries() url.Values {
	return ctx.req.Queries()
}

// AddHeader adds a header to the response.
func (ctx *Context) AddHeader(key, value string) {
	ctx.res.AddHeader(key, value)
}

// SetHeader sets a header in the response.
func (ctx *Context) SetHeader(key, value string) {
	ctx.res.SetHeader(key, value)
}

// GetHeader retrieves a header value by name from the response.
func (ctx *Context) GetHeader(key string) string {
	return ctx.res.GetHeader(key)
}

// DelHeader removes a header from the response.
func (ctx *Context) DelHeader(key string) {
	ctx.res.DelHeader(key)
}

// Status sets the HTTP status code for the response and returns an error if it fails.
func (ctx *Context) Status(code int) error {
	if code < 100 || code > 999 {
		return errors.New("invalid status code")
	}

	ctx.res.Status(code)
	return nil
}

// Write writes data to the response body.
func (ctx *Context) Write(data []byte) (int, error) {
	return ctx.res.Write(data)
}

// Request returns the wrapped Request object associated with the context.
func (ctx *Context) Request() core.Request {
	return ctx.req
}

// Response returns the wrapped Response object associated with the context.
func (ctx *Context) Response() core.Response {
	return ctx.res
}
