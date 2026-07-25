package engine_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-amwk/core"
	"github.com/go-amwk/engine"
)

func TestContext(t *testing.T) {
	app := NewApplication()
	req := NewRequest(nil)
	res := NewResponse(nil)

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
	ctx := getDefaultContext(nil, nil)
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
	ctx := getDefaultContext(nil, nil)
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
	ctx := getDefaultContext(httptest.NewRequest("GET", "/test", nil), nil)

	if !reflect.DeepEqual(ctx.Context(), ctx.Request().Context()) {
		t.Errorf("Expected Context to return the same context as Request.Context(), got %v", ctx.Context())
	}
}

func TestContext_Abort(t *testing.T) {
	ctx := getDefaultContext(nil, nil)
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
	ctx := getDefaultContext(nil, nil)
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
	ctx := getDefaultContext(nil, nil)
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
	ctx := getDefaultContext(nil, nil)
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
	ctx := getDefaultContext(nil, nil)
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
	ctx := getDefaultContext(nil, nil)
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
	ctx := getDefaultContext(nil, nil)
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
	ctx := getDefaultContext(httptest.NewRequest(http.MethodGet, "/test", bytes.NewBufferString("hello")), nil)

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
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := getDefaultContext(req, nil)

	// ClientIP returns the direct connection IP, independent of proxy headers.
	if ip := ctx.ClientIP(); ip != "192.0.2.1" {
		t.Errorf("Expected ClientIP '192.0.2.1', got %v", ip)
	}

	// Setting proxy headers does NOT affect ClientIP.
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	req.Header.Set("X-Real-IP", "198.51.100.1")
	if ip := ctx.ClientIP(); ip != "192.0.2.1" {
		t.Errorf("Expected ClientIP to remain '192.0.2.1' regardless of proxy headers, got %v", ip)
	}

	// Changing the underlying request's client IP is reflected.
	req.RemoteAddr = "192.168.1.1:2345"
	if ip := ctx.ClientIP(); ip != "192.168.1.1" {
		t.Errorf("Expected ClientIP '192.168.1.1', got %v", ip)
	}

	// Changing the underlying request's RemoteAddr to ipv6
	req.RemoteAddr = "[2001:db8::1]:12345"
	if ip := ctx.ClientIP(); ip != "2001:db8::1" {
		t.Errorf("Expected ClientIP '2001:db8::1', got %v", ip)
	}
}

func TestContext_ClientIPs(t *testing.T) {
	tests := []struct {
		name     string
		xff      string
		realIP   string
		clientIP string
		want     []string
	}{
		{
			name:     "no proxy headers",
			xff:      "",
			realIP:   "",
			clientIP: "10.0.0.1:1234",
			want:     []string{"10.0.0.1"},
		},
		{
			name:     "X-Forwarded-For only",
			xff:      "203.0.113.1, 198.51.100.2, 10.0.0.1",
			realIP:   "",
			clientIP: "10.0.0.1:1234",
			want:     []string{"203.0.113.1", "198.51.100.2", "10.0.0.1"},
		},
		{
			name:     "X-Real-IP only",
			xff:      "",
			realIP:   "203.0.113.1",
			clientIP: "10.0.0.1:1234",
			want:     []string{"203.0.113.1", "10.0.0.1"},
		},
		{
			name:     "both headers deduplicated",
			xff:      "203.0.113.1, 198.51.100.2",
			realIP:   "203.0.113.1",
			clientIP: "198.51.100.2:1234",
			want:     []string{"203.0.113.1", "198.51.100.2"},
		},
		{
			name:     "XFF with whitespace",
			xff:      " 203.0.113.1 ,  198.51.100.2 ",
			realIP:   "",
			clientIP: "10.0.0.1:1234",
			want:     []string{"203.0.113.1", "198.51.100.2", "10.0.0.1"},
		},
		{
			name:     "XFF duplicated",
			xff:      " 203.0.113.1 , 203.0.113.1 , 198.51.100.2 ",
			realIP:   "",
			clientIP: "10.0.0.1:1234",
			want:     []string{"203.0.113.1", "198.51.100.2", "10.0.0.1"},
		},
		{
			name:     "XFF with empty entries",
			xff:      "203.0.113.1, , 198.51.100.2",
			realIP:   "",
			clientIP: "10.0.0.1:1234",
			want:     []string{"203.0.113.1", "198.51.100.2", "10.0.0.1"},
		},
		{
			name:     "all three distinct",
			xff:      "203.0.113.1",
			realIP:   "198.51.100.2",
			clientIP: "10.0.0.1:1234",
			want:     []string{"203.0.113.1", "198.51.100.2", "10.0.0.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := getDefaultContext(req, nil)
			req.RemoteAddr = tt.clientIP
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}

			got := ctx.ClientIPs()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientIPs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContext_ContentLength(t *testing.T) {
	data := "test data"
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(data))
	ctx := getDefaultContext(req, nil)

	if l := ctx.ContentLength(); l != int64(len(data)) {
		t.Errorf("Expected ContentLength %d, got %d", len(data), l)
	}
}

func TestContext_ContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := getDefaultContext(req, nil)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if ct := ctx.ContentType(); ct != "application/json" {
		t.Errorf("Expected ContentType 'application/json', got %v", ct)
	}
	req.Header.Del("Content-Type")
	if ct := ctx.ContentType(); ct != "" {
		t.Errorf("Expected empty ContentType, got %v", ct)
	}
}

func TestContext_Header(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := getDefaultContext(req, nil)

	req.Header.Set("X-Test-Header", "v")
	if got := ctx.Header("X-Test-Header"); got != "v" {
		t.Errorf("Expected header 'v', got %v", got)
	}
}

func TestContext_HeaderValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := getDefaultContext(req, nil)
	req.Header.Add("X-Test", "a")
	req.Header.Add("X-Test", "b")
	hv := ctx.HeaderValues("X-Test")
	if !reflect.DeepEqual(hv, []string{"a", "b"}) {
		t.Errorf("Expected HeaderValues ['a','b'], got %v", hv)
	}
}

func TestContext_Headers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := getDefaultContext(req, nil)
	req.Header.Set("A", "1")
	headers := ctx.Headers()
	if headers.Get("A") != "1" {
		t.Errorf("Expected Headers to contain A=1, got %v", headers)
	}
}

func TestContext_Cookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := getDefaultContext(req, nil)

	req.AddCookie(&http.Cookie{Name: "sid", Value: "abc"})
	c, err := ctx.Cookie("sid")
	if err != nil || c.Value != "abc" {
		t.Fatalf("Cookie lookup failed: %v, %v", c, err)
	}
}

func TestContext_Cookies(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := getDefaultContext(req, nil)
	req.AddCookie(&http.Cookie{Name: "s", Value: "1"})
	cs := ctx.Cookies()
	if len(cs) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(cs))
	}
}

func TestContext_Query(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?q=v", nil)
	ctx := getDefaultContext(req, nil)
	if q := ctx.Query("q"); q != "v" {
		t.Errorf("Expected Query 'v', got %v", q)
	}
}

func TestContext_QueryValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?m=x&m=y", nil)
	ctx := getDefaultContext(req, nil)
	if !reflect.DeepEqual(ctx.QueryValues("m"), []string{"x", "y"}) {
		t.Errorf("Expected QueryValues ['x','y'], got %v", ctx.QueryValues("m"))
	}
}

func TestContext_Queries(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?a=b", nil)
	ctx := getDefaultContext(req, nil)
	qs := ctx.Queries()
	if qs.Get("a") != "b" {
		t.Errorf("Expected Queries to contain a=b, got %v", qs)
	}
}

func TestContext_PathValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := getDefaultContext(req, nil)
	req.SetPathValue("id", "123")
	if pv := ctx.PathValue("id"); pv != "123" {
		t.Errorf("Expected PathValue '123', got %v", pv)
	}
}

func TestContext_Resource(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := getDefaultContext(req, nil)
	ctx.Request().SetResource("/r")
	if r := ctx.Resource(); r != "/r" {
		t.Errorf("Expected Resource '/r', got %v", r)
	}
}

func TestContext_Method_Protocol_Path(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	ctx := getDefaultContext(req, nil)
	if ctx.Method() != "POST" || ctx.Protocol() != "HTTP/1.1" || ctx.Path() != "/test" {
		t.Errorf("Method/Protocol/Path mismatch")
	}
}

func TestContext_AddHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	ctx.AddHeader("X-Test", "v")
	resp.send()

	if rr.Header().Get("X-Test") != "v" {
		t.Errorf("Expected AddHeader to set X-Test=v, got %v", rr.Header().Get("X-Test"))
	}
}

func TestContext_SetHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	ctx.SetHeader("X-Test", "v")
	ctx.SetHeader("X-Test", "v2")

	resp.send()

	if rr.Header().Get("X-Test") != "v2" {
		t.Errorf("Expected SetHeader to overwrite X-Test=v2, got %v", rr.Header().Get("X-Test"))
	}
}

func TestContext_GetHeader(t *testing.T) {
	resp := NewResponse(httptest.NewRecorder())
	ctx := getDefaultContext(nil, resp)
	ctx.SetHeader("X-Test", "v")
	if h := ctx.GetHeader("X-Test"); h != "v" {
		t.Errorf("Expected GetHeader to return 'v', got %v", h)
	}
}

func TestContext_DelHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	ctx.SetHeader("X-Test", "v")
	ctx.DelHeader("X-Test")
	if h := ctx.GetHeader("X-Test"); h != "" {
		t.Errorf("Expected DelHeader to remove X-Test, got %v", h)
	}

	resp.send()

	if rr.Header().Get("X-Test") != "" {
		t.Errorf("Expected sent header X-Test to be deleted, got %v", rr.Header().Get("X-Test"))
	}
}

func TestContext_Status(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	if err := ctx.Status(99); err == nil {
		t.Errorf("Expected Status to return error for invalid code 99")
	}
	if err := ctx.Status(1000); err == nil {
		t.Errorf("Expected Status to return error for invalid code 1000")
	}
	if err := ctx.Status(201); err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if ctx.Response().StatusCode() != 201 {
		t.Errorf("Expected status code 201, got %d", ctx.Response().StatusCode())
	}

	resp.send()

	if rr.Code != 201 {
		t.Errorf("Expected sent status code 201, got %d", rr.Code)
	}
}

func TestContext_Write(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	n, err := ctx.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected Write to write 5 bytes, got %d", n)
	}

	resp.send()

	if rr.Body.String() != "hello" {
		t.Errorf("Expected sent body 'hello', got %v", rr.Body.String())
	}
}

func TestContext_String(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	n, err := ctx.String("hello")
	if err != nil {
		t.Fatalf("String returned error: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected String to write 5 bytes, got %d", n)
	}

	resp.send()

	if rr.Body.String() != "hello" {
		t.Errorf("Expected sent body 'hello', got %v", rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/plain; charset=utf-8', got %v", rr.Header().Get("Content-Type"))
	}
}

func TestContext_String_PreserveContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	n, err := ctx.String("<p>hello</p>")
	if err != nil {
		t.Fatalf("String returned error: %v", err)
	}
	if n != 12 {
		t.Errorf("Expected String to write 12 bytes, got %d", n)
	}

	resp.send()

	if rr.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type to be preserved as 'text/html; charset=utf-8', got %v", rr.Header().Get("Content-Type"))
	}
}

func TestContext_JSON(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	data := map[string]string{"key": "value"}
	n, err := ctx.JSON(data)
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}
	expectedJSON := `{"key":"value"}`
	if n != len(expectedJSON) {
		t.Errorf("Expected JSON to write %d bytes, got %d", len(expectedJSON), n)
	}

	resp.send()

	if rr.Body.String() != expectedJSON {
		t.Errorf("Expected sent body '%s', got '%s'", expectedJSON, rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %v", rr.Header().Get("Content-Type"))
	}
}

func TestContext_JSON_PreserveContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
	data := map[string]string{"key": "value"}
	_, err := ctx.JSON(data)
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	resp.send()

	if rr.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type to be preserved as 'application/json; charset=utf-8', got %v", rr.Header().Get("Content-Type"))
	}
}

func TestContext_JSON_MarshalError(t *testing.T) {
	resp := NewResponse(httptest.NewRecorder())
	ctx := getDefaultContext(nil, resp)
	// channel and func types cannot be marshaled to JSON
	_, err := ctx.JSON(make(chan int))
	if err == nil {
		t.Fatalf("Expected JSON to return error for unmarshalable type")
	}
}

func TestContext_String_Empty(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	n, err := ctx.String("")
	if err != nil {
		t.Fatalf("String returned error: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected String to write 0 bytes, got %d", n)
	}

	resp.send()

	if rr.Body.String() != "" {
		t.Errorf("Expected empty body, got '%s'", rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/plain; charset=utf-8', got %v", rr.Header().Get("Content-Type"))
	}
}

func TestContext_JSON_Nil(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)
	n, err := ctx.JSON(nil)
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}
	expectedJSON := "null"
	if n != len(expectedJSON) {
		t.Errorf("Expected JSON to write %d bytes, got %d", len(expectedJSON), n)
	}

	resp.send()

	if rr.Body.String() != expectedJSON {
		t.Errorf("Expected sent body 'null', got '%s'", rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %v", rr.Header().Get("Content-Type"))
	}
}

func TestContext_JSON_MarshalError_NoContentTypeSideEffect(t *testing.T) {
	resp := NewResponse(httptest.NewRecorder())
	ctx := getDefaultContext(nil, resp)
	// channel types cannot be marshaled — Content-Type must NOT be set on failure
	_, err := ctx.JSON(make(chan int))
	if err == nil {
		t.Fatalf("Expected JSON to return error for unmarshalable type")
	}
	if ctx.GetHeader("Content-Type") != "" {
		t.Errorf("Expected no Content-Type header to be set on marshal error, got '%s'", ctx.GetHeader("Content-Type"))
	}
}

func TestContext_Redirect(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)

	err := ctx.Redirect("https://example.com", http.StatusMovedPermanently)
	if err != nil {
		t.Fatalf("Redirect returned error: %v", err)
	}

	resp.send()

	if rr.Header().Get("Location") != "https://example.com" {
		t.Errorf("Expected Location header to be set to 'https://example.com', got %v", rr.Header().Get("Location"))
	}
	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("Expected status code %d, got %d", http.StatusMovedPermanently, rr.Code)
	}
}

func TestContext_Redirect_DefaultStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)

	err := ctx.Redirect("https://example.com")
	if err != nil {
		t.Fatalf("Redirect returned error: %v", err)
	}

	resp.send()

	if rr.Header().Get("Location") != "https://example.com" {
		t.Errorf("Expected Location header to be set to 'https://example.com', got %v", rr.Header().Get("Location"))
	}
	if rr.Code != http.StatusFound {
		t.Errorf("Expected status code %d, got %d", http.StatusFound, rr.Code)
	}
}

func TestContext_Redirect_InvalidCode(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	ctx := getDefaultContext(nil, resp)

	err := ctx.Redirect("https://example.com", http.StatusBadRequest)
	if err == nil {
		t.Fatalf("Expected Redirect to return error for non-redirect status code 400")
	}

	err = ctx.Redirect("https://example.com", 200)
	if err == nil {
		t.Fatalf("Expected Redirect to return error for non-redirect status code 200")
	}

	err = ctx.Redirect("https://example.com", 500)
	if err == nil {
		t.Fatalf("Expected Redirect to return error for non-redirect status code 500")
	}

	// Verify no headers were set on error
	if rr.Header().Get("Location") != "" {
		t.Errorf("Expected no Location header to be set on error")
	}
}

func TestContext_Redirect_BodyOnStatus(t *testing.T) {
	tests := []struct {
		code       int
		statusText string
	}{
		{http.StatusMultipleChoices, "Multiple Choices"},
		{http.StatusMovedPermanently, "Moved Permanently"},
		{http.StatusFound, "Found"},
		{http.StatusSeeOther, "See Other"},
		{http.StatusTemporaryRedirect, "Temporary Redirect"},
		{http.StatusPermanentRedirect, "Permanent Redirect"},
	}

	for _, tt := range tests {
		t.Run(tt.statusText, func(t *testing.T) {
			rr := httptest.NewRecorder()
			resp := NewResponse(rr)
			ctx := getDefaultContext(nil, resp)

			err := ctx.Redirect("https://example.com/path", tt.code)
			if err != nil {
				t.Fatalf("Redirect returned error: %v", err)
			}

			resp.send()

			if rr.Code != tt.code {
				t.Errorf("Expected status code %d, got %d", tt.code, rr.Code)
			}
		})
	}
}

func getDefaultContext(r *http.Request, res *Response) *engine.Context {
	if r == nil {
		r = httptest.NewRequest(http.MethodGet, "/", nil)
	}
	if res == nil {
		res = NewResponse(httptest.NewRecorder())
	}
	app := NewApplication()
	req := NewRequest(r)

	return engine.NewContext(app, req, res)
}
