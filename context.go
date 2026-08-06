package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-amwk/core"
)

type Context struct {
	app core.Application
	req core.Request
	res core.Response

	state    map[string]any
	mu       sync.Mutex
	index    int
	isAbort  atomic.Bool
	handlers []core.HandlerFunc
}

func NewContext(app core.Application, req core.Request, res core.Response) *Context {
	ctx := new(Context)
	ctx.app = app
	ctx.index = 0
	ctx.isAbort.Store(false)
	ctx.req = req
	ctx.res = res
	ctx.state = make(map[string]any)

	ctx.handlers = make([]core.HandlerFunc, 0)

	return ctx
}

// Get returns the value associated with the key in the context.
func (ctx *Context) Get(key string) (any, bool) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	value, ok := ctx.state[key]
	return value, ok
}

// Set sets the value for the key in the context and returns the previous value.
func (ctx *Context) Set(key string, value any) any {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	oldValue := ctx.state[key]
	ctx.state[key] = value
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
	ctx.isAbort.Store(true)
}

// IsAbort checks if the context is marked as aborted.
func (ctx *Context) IsAbort() bool {
	return ctx.isAbort.Load()
}

// Next executes the remaining handlers in the chain sequentially.
//
// Next is designed to be called both at the top level (to start handler chain execution)
// and from within a handler to implement nested middleware patterns. This allows handlers
// to wrap subsequent handlers, executing code both before and after them:
//
//	ctx.Use(func(c core.Context) error {
//	    // pre-processing
//	    c.Next()              // executes all remaining handlers
//	    // post-processing
//	    return nil
//	})
//
// Behavior:
//   - Each handler in the chain is executed exactly once per Next call.
//   - If a handler returns an error, execution stops immediately and that error is returned.
//   - If a handler panics, the panic is propagated up the call stack, and subsequent handlers are
//     not executed.
//   - Calling Next multiple times from the same handler is safe but idempotent:
//     the second and subsequent calls are no-ops (all remaining handlers have already run).
//   - Next is NOT safe for concurrent use from multiple goroutines.
//   - Calling [Context.Abort] before Next starts prevents all handlers from executing.
//   - Calling [Context.Abort] from within a handler prevents subsequent handlers
//     from executing, but the current handler continues to completion.
func (ctx *Context) Next() error {
	for !ctx.isAbort.Load() && ctx.index < len(ctx.handlers) {
		handler := ctx.handlers[ctx.index]
		ctx.index++

		if err := handler(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Use appends handlers to the context's handler chain. Handlers are executed
// in the order they are added when [Context.Next] is called.
//
// Use may be called before Next to register handlers upfront, or from within a
// handler during Next execution to dynamically append additional handlers:
//
//	ctx.Use(func(c core.Context) error {
//	    // conditional middleware injection
//	    ctx.Use(authMiddleware)
//	    return c.Next()
//	})
//
// Handlers added during Next execution will be picked up by the current Next
// loop if the current index has not yet passed them — i.e., dynamically added
// handlers at the tail of the chain will still execute.
//
// Use is NOT safe for concurrent use from multiple goroutines.
func (ctx *Context) Use(handlers ...core.HandlerFunc) core.Context {
	ctx.handlers = append(ctx.handlers, handlers...)
	return ctx
}

// Body returns the request body as a readable stream.
func (ctx *Context) Body() (io.ReadCloser, error) {
	return ctx.req.Body()
}

// ClientIP returns the IP address of the direct TCP connection peer.
func (ctx *Context) ClientIP() string {
	return ctx.req.ClientIP()
}

// ClientIPs collects all available client IP information from the request,
// combining proxy headers with the direct connection IP. The returned slice
// is deduplicated and preserves the following priority order:
//
//  1. X-Forwarded-For chain (original client first, each proxy in order)
//  2. X-Real-IP (only if not already present from X-Forwarded-For)
//  3. Direct connection IP (only if not already present from the headers above)
//
// The result is never nil; at minimum it contains the direct connection IP.
func (ctx *Context) ClientIPs() []string {
	ips := make([]string, 0)
	recorded := make(map[string]struct{})

	xff := ctx.Header("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if ip == "" {
				continue
			}
			if _, ok := recorded[ip]; ok {
				continue
			}
			ips = append(ips, ip)
			recorded[ip] = struct{}{}
		}
	}
	realIP := ctx.Header("X-Real-IP")
	if realIP != "" {
		if _, ok := recorded[realIP]; !ok {
			ips = append(ips, realIP)
			recorded[realIP] = struct{}{}
		}
	}

	clientIP := ctx.ClientIP()
	if _, ok := recorded[clientIP]; !ok {
		ips = append(ips, clientIP)
	}

	// TODO: add trusted proxy validation to filter out untrusted IPs from the headers.
	return ips
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

// Size returns the size of the response body in bytes.
func (ctx *Context) Size() int {
	return ctx.res.Size()
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

// WriteString writes a string to the response body.
func (ctx *Context) WriteString(s string) (int, error) {
	return ctx.res.WriteString(s)
}

// Written returns the data that has been written to the response body so far.
func (ctx *Context) Written() []byte {
	return ctx.res.Written()
}

// String writes a string to the response body.
func (ctx *Context) String(s string) (int, error) {
	if ctx.GetHeader("Content-Type") == "" {
		ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
	}

	return ctx.res.Write([]byte(s))
}

// JSON serializes the given value to JSON and writes it to the response body.
func (ctx *Context) JSON(v any) (int, error) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}

	if ctx.GetHeader("Content-Type") == "" {
		ctx.SetHeader("Content-Type", "application/json")
	}

	return ctx.res.Write(jsonData)
}

// Redirect sets the Location header and writes a redirect response with the specified status code,
// defaulting to 302 Found if no code is provided.
func (ctx *Context) Redirect(link string, code ...int) error {
	statusCode := http.StatusFound
	if len(code) > 0 {
		statusCode = code[0]
	}
	switch statusCode {
	case http.StatusMultipleChoices, http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		// the status code is always valid
		_ = ctx.Status(statusCode)
	default:
		return fmt.Errorf("invalid redirect status code: %d", statusCode)
	}

	ctx.SetHeader("Location", link)

	return nil
}

// Request returns the wrapped Request object associated with the context.
func (ctx *Context) Request() core.Request {
	return ctx.req
}

// Response returns the wrapped Response object associated with the context.
func (ctx *Context) Response() core.Response {
	return ctx.res
}
