package engine_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/go-amwk/core"
	"github.com/go-amwk/engine"
)

func TestContext(t *testing.T) {
	app := NewApplication()
	req := NewRequest()
	res := NewResponse()

	ctx := engine.NewContext(app, req, res)
	if !reflect.DeepEqual(ctx.Application(), app) {
		t.Errorf("Expected Application to be %v, got %v", app, ctx.Application())
	}
	if !reflect.DeepEqual(ctx.Request(), req) {
		t.Errorf("Expected Request to be %v, got %v", req, ctx.Request())
	}
	if !reflect.DeepEqual(ctx.Response(), res) {
		t.Errorf("Expected Response to be %v, got %v", res, ctx.Response())
	}
}

func TestContext_Get(t *testing.T) {
	ctx := getDefaultContext()
	key := "test_key"
	expected := "test_value"

	_, ok := ctx.Get(key)
	if ok {
		t.Errorf("Expected Get to return false for non-existent key, got true")
	}
	ctx.Set(key, expected)
	value, ok := ctx.Get(key)
	if !ok {
		t.Errorf("Expected Get to return true for existing key, got false")
	}
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("Expected Get to return %v, got %v", expected, value)
	}
}

func TestContext_Set(t *testing.T) {
	ctx := getDefaultContext()
	key := "test_key"
	value1 := "test_value1"
	value2 := "test_value2"

	oldValue := ctx.Set(key, value1)
	if oldValue != nil {
		t.Errorf("Expected Set to return nil for new key, got %v", oldValue)
	}
	oldValue = ctx.Set(key, value2)
	if !reflect.DeepEqual(oldValue, value1) {
		t.Errorf("Expected Set to return previous value %v, got %v", value1, oldValue)
	}
	value, _ := ctx.Get(key)
	if !reflect.DeepEqual(value, value2) {
		t.Errorf("Expected Get to return updated value %v, got %v", value2, value)
	}
}

func TestContext_Context(t *testing.T) {
	ctx := getDefaultContext()

	if !reflect.DeepEqual(ctx.Context(), ctx.Request().Context()) {
		t.Errorf("Expected Context to return the same context as Request.Context(), got %v", ctx.Context())
	}
}

func TestContext_Abort(t *testing.T) {
	ctx := getDefaultContext()
	seq := []int{}

	ctx.Use(func(c core.Context) error {
		seq = append(seq, 1)
		return nil
	})
	ctx.Use(func(c core.Context) error {
		seq = append(seq, 2)
		c.Abort()
		return nil
	})
	ctx.Use(func(c core.Context) error {
		seq = append(seq, 3)
		return nil
	})

	if err := ctx.Next(); err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if !reflect.DeepEqual(seq, []int{1, 2}) {
		t.Errorf("Expected handler sequence [1 2], got %v", seq)
	}
	if !ctx.IsAbort() {
		t.Errorf("Expected IsAbort true after abort")
	}
}

func TestContext_Next(t *testing.T) {
	ctx := getDefaultContext()
	seq := []int{}

	ctx.Use(func(c core.Context) error {
		seq = append(seq, 1)
		c.Next()
		seq = append(seq, 1)
		return nil
	})
	ctx.Use(func(c core.Context) error {
		seq = append(seq, 2)
		c.Next()
		seq = append(seq, 2)
		return nil
	})
	ctx.Use(func(c core.Context) error {
		seq = append(seq, 3)
		c.Next()
		seq = append(seq, 3)
		return nil
	})

	if err := ctx.Next(); err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if !reflect.DeepEqual(seq, []int{1, 2, 3, 3, 2, 1}) {
		t.Errorf("Expected handler sequence [1 2 3 3 2 1], got %v", seq)
	}
	if ctx.IsAbort() {
		t.Errorf("Expected IsAbort false after Next")
	}
}

func TestContext_Next_ReturnError(t *testing.T) {
	ctx := getDefaultContext()
	expectedErr := errors.New("handler error")
	seq := []int{}

	ctx.Use(func(c core.Context) error {
		seq = append(seq, 1)
		return nil
	})
	ctx.Use(func(c core.Context) error {
		seq = append(seq, 2)
		return expectedErr
	})
	ctx.Use(func(c core.Context) error {
		seq = append(seq, 3)
		return nil
	})

	err := ctx.Next()
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected Next to return error %v, got %v", expectedErr, err)
	}
	if !reflect.DeepEqual(seq, []int{1, 2}) {
		t.Errorf("Expected handler sequence [1 2], got %v", seq)
	}
}

func TestContext_Next_Panic(t *testing.T) {
	ctx := getDefaultContext()
	expectedErr := errors.New("handler panic")
	seq := []int{}

	ctx.Use(func(c core.Context) error {
		seq = append(seq, 1)
		return nil
	})
	ctx.Use(func(c core.Context) error {
		seq = append(seq, 2)
		panic(expectedErr)
	})
	ctx.Use(func(c core.Context) error {
		seq = append(seq, 3)
		return nil
	})

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic, but Next did not panic")
		} else if err, ok := r.(error); !ok || !errors.Is(err, expectedErr) {
			t.Errorf("Expected panic error %v, got %v", expectedErr, r)
		}
		if !reflect.DeepEqual(seq, []int{1, 2}) {
			t.Errorf("Expected handler sequence [1 2], got %v", seq)
		}
	}()

	ctx.Next()
}

func TestContext_Next_PanicNonError(t *testing.T) {
	ctx := getDefaultContext()
	expectedPanic := "unexpected panic"
	seq := []int{}

	ctx.Use(func(c core.Context) error {
		seq = append(seq, 1)
		return nil
	})
	ctx.Use(func(c core.Context) error {
		seq = append(seq, 2)
		panic(expectedPanic)
	})
	ctx.Use(func(c core.Context) error {
		seq = append(seq, 3)
		return nil
	})

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic, but Next did not panic")
		} else if err, ok := r.(string); !ok || err != "unexpected panic" {
			t.Errorf("Expected panic error 'unexpected panic', got %v", r)
		}
		if !reflect.DeepEqual(seq, []int{1, 2}) {
			t.Errorf("Expected handler sequence [1 2], got %v", seq)
		}
	}()

	ctx.Next()
}

func TestContext_Use(t *testing.T) {
	ctx := getDefaultContext()
	ctx.Use(func(c core.Context) error {
		c.Set("a", 1)
		return nil
	}, func(c core.Context) error {
		if v, _ := c.Get("a"); v != 1 {
			t.Errorf("Expected to get value 1 for key 'a', got %v", v)
		}
		return nil
	})

	if err := ctx.Next(); err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
}

func TestContext_Use_InNext(t *testing.T) {
	ctx := getDefaultContext()
	outer := false
	inner := false
	ctx.Use(func(c core.Context) error {
		outer = true
		ctx.Use(func(c core.Context) error {
			inner = true
			return nil
		})
		return nil
	})

	if err := ctx.Next(); err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if !outer {
		t.Errorf("Expected outer handler to run")
	}
	if !inner {
		t.Errorf("Expected inner handler to run")
	}
}

func TestContext_Body(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.body = io.NopCloser(bytes.NewReader([]byte("hello")))

	rc, err := ctx.Body()
	if err != nil {
		t.Fatalf("Body returned error: %v", err)
	}
	b, _ := io.ReadAll(rc)
	if string(b) != "hello" {
		t.Errorf("Expected body 'hello', got %s", string(b))
	}
}

func TestContext_ClientIP(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)

	if ip := ctx.ClientIP(); ip != "127.0.0.1" {
		t.Errorf("Expected ClientIP '127.0.0.1', got %v", ip)
	}

	// X-Forwarded-For takes precedence
	req.headers.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	if ip := ctx.ClientIP(); ip != "203.0.113.1" {
		t.Errorf("Expected ClientIP to be '203.0.113.1', got %v", ip)
	}
	req.headers.Del("X-Forwarded-For")
	if ip := ctx.ClientIP(); ip != "127.0.0.1" {
		t.Errorf("Expected ClientIP to fallback to '127.0.0.1', got %v", ip)
	}
}

func TestContext_ContentLength(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.contentLength = 999
	if l := ctx.ContentLength(); l != 999 {
		t.Errorf("Expected ContentLength 999, got %d", l)
	}
}

func TestContext_ContentType(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.headers.Set("Content-Type", "application/json; charset=utf-8")
	if ct := ctx.ContentType(); ct != "application/json" {
		t.Errorf("Expected ContentType 'application/json', got %v", ct)
	}
	req.headers.Del("Content-Type")
	if ct := ctx.ContentType(); ct != "" {
		t.Errorf("Expected empty ContentType, got %v", ct)
	}
}

func TestContext_Header(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.headers.Set("X-Test-Header", "v")
	if got := ctx.Header("X-Test-Header"); got != "v" {
		t.Errorf("Expected header 'v', got %v", got)
	}
}

func TestContext_HeaderValues(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.headers.Add("X-Test", "a")
	req.headers.Add("X-Test", "b")
	hv := ctx.HeaderValues("X-Test")
	if !reflect.DeepEqual(hv, []string{"a", "b"}) {
		t.Errorf("Expected HeaderValues ['a','b'], got %v", hv)
	}
}

func TestContext_Headers(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.headers.Set("A", "1")
	headers := ctx.Headers()
	if headers.Get("A") != "1" {
		t.Errorf("Expected Headers to contain A=1, got %v", headers)
	}
}

func TestContext_Cookie(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.cookies = []*http.Cookie{{Name: "sid", Value: "abc"}}
	c, err := ctx.Cookie("sid")
	if err != nil || c.Value != "abc" {
		t.Fatalf("Cookie lookup failed: %v, %v", c, err)
	}
}

func TestContext_Cookies(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.cookies = []*http.Cookie{{Name: "s", Value: "1"}}
	cs := ctx.Cookies()
	if len(cs) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(cs))
	}
}

func TestContext_Query(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.queryValues.Set("q", "v")
	if q := ctx.Query("q"); q != "v" {
		t.Errorf("Expected Query 'v', got %v", q)
	}
}

func TestContext_QueryValues(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.queryValues["m"] = []string{"x", "y"}
	if !reflect.DeepEqual(ctx.QueryValues("m"), []string{"x", "y"}) {
		t.Errorf("Expected QueryValues ['x','y'], got %v", ctx.QueryValues("m"))
	}
}

func TestContext_Queries(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.queryValues.Set("a", "b")
	qs := ctx.Queries()
	if qs.Get("a") != "b" {
		t.Errorf("Expected Queries to contain a=b, got %v", qs)
	}
}

func TestContext_PathValue(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.SetPathValue("id", "123")
	if pv := ctx.PathValue("id"); pv != "123" {
		t.Errorf("Expected PathValue '123', got %v", pv)
	}
}

func TestContext_Resource(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.SetResource("/r")
	if r := ctx.Resource(); r != "/r" {
		t.Errorf("Expected Resource '/r', got %v", r)
	}
}

func TestContext_Method_Protocol_Path(t *testing.T) {
	ctx := getDefaultContext()
	req := ctx.Request().(*Request)
	req.method = "POST"
	req.protocol = "HTTP/2.0"
	req.path = "/p"
	if ctx.Method() != "POST" || ctx.Protocol() != "HTTP/2.0" || ctx.Path() != "/p" {
		t.Errorf("Method/Protocol/Path mismatch")
	}
}

func TestContext_AddHeader(t *testing.T) {
	ctx := getDefaultContext()
	ctx.AddHeader("X-Test", "v")
	if ctx.Response().Headers().Get("X-Test") != "v" {
		t.Errorf("Expected AddHeader to set X-Test=v, got %v", ctx.Response().Headers().Get("X-Test"))
	}
}

func TestContext_SetHeader(t *testing.T) {
	ctx := getDefaultContext()
	ctx.SetHeader("X-Test", "v")
	if ctx.Response().Headers().Get("X-Test") != "v" {
		t.Errorf("Expected SetHeader to set X-Test=v, got %v", ctx.Response().Headers().Get("X-Test"))
	}
	ctx.SetHeader("X-Test", "v2")
	if ctx.Response().Headers().Get("X-Test") != "v2" {
		t.Errorf("Expected SetHeader to overwrite X-Test=v2, got %v", ctx.Response().Headers().Get("X-Test"))
	}
}

func TestContext_GetHeader(t *testing.T) {
	ctx := getDefaultContext()
	ctx.SetHeader("X-Test", "v")
	if h := ctx.GetHeader("X-Test"); h != "v" {
		t.Errorf("Expected GetHeader to return 'v', got %v", h)
	}
}

func TestContext_DelHeader(t *testing.T) {
	ctx := getDefaultContext()
	ctx.SetHeader("X-Test", "v")
	ctx.DelHeader("X-Test")
	if h := ctx.GetHeader("X-Test"); h != "" {
		t.Errorf("Expected DelHeader to remove X-Test, got %v", h)
	}
}

func TestContext_Status(t *testing.T) {
	ctx := getDefaultContext()
	if err := ctx.Status(201); err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if ctx.Response().StatusCode() != 201 {
		t.Errorf("Expected status code 201, got %d", ctx.Response().StatusCode())
	}
	if err := ctx.Status(99); err == nil {
		t.Errorf("Expected Status to return error for invalid code 99")
	}
	if err := ctx.Status(1000); err == nil {
		t.Errorf("Expected Status to return error for invalid code 1000")
	}
}

func TestContext_Write(t *testing.T) {
	ctx := getDefaultContext()
	n, err := ctx.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected Write to write 5 bytes, got %d", n)
	}
}

func getDefaultContext() *engine.Context {
	app := NewApplication()
	req := NewRequest()
	res := NewResponse()

	return engine.NewContext(app, req, res)
}
