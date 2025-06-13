package util_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pitabwire/util"
)

type MockJSONRequestHandler struct {
	handler func(req *http.Request) util.JSONResponse
}

func (h *MockJSONRequestHandler) OnIncomingRequest(req *http.Request) util.JSONResponse {
	return h.handler(req)
}

type MockResponse struct {
	Foo string `json:"foo"`
}

func TestMakeJSONAPI(t *testing.T) {
	tests := []struct {
		Return     util.JSONResponse
		ExpectCode int
		ExpectJSON string
	}{
		// MessageResponse return values
		{
			util.MessageResponse(http.StatusInternalServerError, "Everything is broken"),
			http.StatusInternalServerError,
			`{"message":"Everything is broken"}`,
		},
		// interface return values
		{
			util.JSONResponse{http.StatusInternalServerError, MockResponse{"yep"}, nil},
			http.StatusInternalServerError,
			`{"foo":"yep"}`,
		},
		// Error JSON return values which fail to be marshalled should fallback to text
		{util.JSONResponse{http.StatusInternalServerError, struct {
			Foo interface{} `json:"foo"`
		}{func(_, _ string) {}}, nil}, http.StatusInternalServerError, `{"message":"Internal Server Error"}`},
		// With different status codes
		{util.JSONResponse{http.StatusCreated, MockResponse{"narp"}, nil}, http.StatusCreated, `{"foo":"narp"}`},
		// Top-level array success values
		{
			util.JSONResponse{http.StatusOK, []MockResponse{{"yep"}, {"narp"}}, nil},
			http.StatusOK,
			`[{"foo":"yep"},{"foo":"narp"}]`,
		},
	}

	for _, tst := range tests {
		mock := MockJSONRequestHandler{func(_ *http.Request) util.JSONResponse {
			return tst.Return
		}}
		mockReq, _ := http.NewRequest(http.MethodGet, "http://example.com/foo", nil)
		mockWriter := httptest.NewRecorder()
		handlerFunc := util.MakeJSONAPI(&mock)
		handlerFunc(mockWriter, mockReq)
		if mockWriter.Code != tst.ExpectCode {
			t.Errorf("TestMakeJSONAPI wanted HTTP status %d, got %d", tst.ExpectCode, mockWriter.Code)
		}
		actualBody := mockWriter.Body.String()
		if actualBody != tst.ExpectJSON {
			t.Errorf("TestMakeJSONAPI wanted body '%s', got '%s'", tst.ExpectJSON, actualBody)
		}
	}
}

func TestMakeJSONAPICustomHeaders(t *testing.T) {
	mock := MockJSONRequestHandler{func(_ *http.Request) util.JSONResponse {
		headers := make(map[string]any)
		headers["Custom"] = "Thing"
		headers["X-Custom"] = "Things"
		return util.JSONResponse{
			Code:    200,
			JSON:    MockResponse{"yep"},
			Headers: headers,
		}
	}}
	mockReq, _ := http.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	mockWriter := httptest.NewRecorder()
	handlerFunc := util.MakeJSONAPI(&mock)
	handlerFunc(mockWriter, mockReq)
	if mockWriter.Code != 200 {
		t.Errorf("TestMakeJSONAPICustomHeaders wanted HTTP status 200, got %d", mockWriter.Code)
	}
	h := mockWriter.Header().Get("Custom")
	if h != "Thing" {
		t.Errorf("TestMakeJSONAPICustomHeaders wanted header 'Custom: Thing' , got 'Custom: %s'", h)
	}
	h = mockWriter.Header().Get("X-Custom")
	if h != "Things" {
		t.Errorf("TestMakeJSONAPICustomHeaders wanted header 'X-Custom: Things' , got 'X-Custom: %s'", h)
	}
}

func TestMakeJSONAPIRedirect(t *testing.T) {
	mock := MockJSONRequestHandler{func(_ *http.Request) util.JSONResponse {
		return util.RedirectResponse("https://matrix.org")
	}}
	mockReq, _ := http.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	mockWriter := httptest.NewRecorder()
	handlerFunc := util.MakeJSONAPI(&mock)
	handlerFunc(mockWriter, mockReq)
	if mockWriter.Code != 302 {
		t.Errorf("TestMakeJSONAPIRedirect wanted HTTP status 302, got %d", mockWriter.Code)
	}
	location := mockWriter.Header().Get("Location")
	if location != "https://matrix.org" {
		t.Errorf("TestMakeJSONAPIRedirect wanted Location header 'https://matrix.org', got '%s'", location)
	}
}

func TestMakeJSONAPIError(t *testing.T) {
	mock := MockJSONRequestHandler{func(_ *http.Request) util.JSONResponse {
		err := errors.New("oops")
		return util.ErrorResponse(err)
	}}
	mockReq, _ := http.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	mockWriter := httptest.NewRecorder()
	handlerFunc := util.MakeJSONAPI(&mock)
	handlerFunc(mockWriter, mockReq)
	if mockWriter.Code != 500 {
		t.Errorf("TestMakeJSONAPIError wanted HTTP status 500, got %d", mockWriter.Code)
	}
	actualBody := mockWriter.Body.String()
	expect := `{"message":"oops"}`
	if actualBody != expect {
		t.Errorf("TestMakeJSONAPIError wanted body '%s', got '%s'", expect, actualBody)
	}
}

func TestIs2xx(t *testing.T) {
	tests := []struct {
		Code   int
		Expect bool
	}{
		{200, true},
		{201, true},
		{299, true},
		{300, false},
		{199, false},
		{0, false},
		{500, false},
	}
	for _, test := range tests {
		j := util.JSONResponse{
			Code: test.Code,
		}
		actual := j.Is2xx()
		if actual != test.Expect {
			t.Errorf("TestIs2xx wanted %t, got %t", test.Expect, actual)
		}
	}
}

func TestGetLogger(t *testing.T) {
	entry := util.NewLogger(t.Context(), util.DefaultLogOptions()).WithField("test", "yep")
	mockReq, _ := http.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	ctx := util.ContextWithLogger(mockReq.Context(), entry)
	mockReq = mockReq.WithContext(ctx)
	ctxLogger := util.Log(mockReq.Context())
	if ctxLogger != entry {
		t.Errorf("TestGetLogger wanted logger '%v', got '%v'", entry, ctxLogger)
	}

	noLoggerInReq, _ := http.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	ctxLogger = util.Log(noLoggerInReq.Context())
	if ctxLogger == nil {
		t.Errorf("TestGetLogger wanted logger, got nil")
	}
}

func TestProtect(t *testing.T) {
	mockWriter := httptest.NewRecorder()
	mockReq, _ := http.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	mockReq = mockReq.WithContext(
		util.ContextWithLogger(
			mockReq.Context(),
			util.NewLogger(t.Context(), util.DefaultLogOptions()).WithField("test", "yep"),
		),
	)
	h := util.Protect(func(_ http.ResponseWriter, _ *http.Request) {
		panic("oh noes!")
	})

	h(mockWriter, mockReq)

	expectCode := 500
	if mockWriter.Code != expectCode {
		t.Errorf("TestProtect wanted HTTP status %d, got %d", expectCode, mockWriter.Code)
	}

	expectBody := `{"message":"Internal Server Error"}`
	actualBody := mockWriter.Body.String()
	if actualBody != expectBody {
		t.Errorf("TestProtect wanted body %s, got %s", expectBody, actualBody)
	}
}

func TestProtectWithoutLogger(t *testing.T) {
	mockWriter := httptest.NewRecorder()
	mockReq, _ := http.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	h := util.Protect(func(_ http.ResponseWriter, _ *http.Request) {
		panic("oh noes!")
	})

	h(mockWriter, mockReq)

	expectCode := 500
	if mockWriter.Code != expectCode {
		t.Errorf("TestProtect wanted HTTP status %d, got %d", expectCode, mockWriter.Code)
	}

	expectBody := `{"message":"Internal Server Error"}`
	actualBody := mockWriter.Body.String()
	if actualBody != expectBody {
		t.Errorf("TestProtect wanted body %s, got %s", expectBody, actualBody)
	}
}

func TestWithCORSOptions(t *testing.T) {
	mockWriter := httptest.NewRecorder()
	mockReq, _ := http.NewRequest(http.MethodOptions, "http://example.com/foo", nil)
	h := util.WithCORSOptions(func(_ http.ResponseWriter, _ *http.Request) {
		mockWriter.WriteString("yep")
	})
	h(mockWriter, mockReq)
	if mockWriter.Code != 200 {
		t.Errorf("TestWithCORSOptions wanted HTTP status 200, got %d", mockWriter.Code)
	}

	origin := mockWriter.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("TestWithCORSOptions wanted Access-Control-Allow-Origin header '*', got '%s'", origin)
	}

	// OPTIONS request shouldn't hit the handler func
	expectBody := ""
	actualBody := mockWriter.Body.String()
	if actualBody != expectBody {
		t.Errorf("TestWithCORSOptions wanted body %s, got %s", expectBody, actualBody)
	}
}

func TestGetRequestID(t *testing.T) {
	reqID := "alphabetsoup"
	mockReq, _ := http.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	ctx := util.ContextWithRequestID(mockReq.Context(), reqID)
	mockReq = mockReq.WithContext(ctx)
	ctxReqID := util.GetRequestID(mockReq.Context())
	if reqID != ctxReqID {
		t.Errorf("TestGetRequestID wanted request ID '%s', got '%s'", reqID, ctxReqID)
	}

	noReqIDInReq, _ := http.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	ctxReqID = util.GetRequestID(noReqIDInReq.Context())
	if ctxReqID != "" {
		t.Errorf("TestGetRequestID wanted empty request ID, got '%s'", ctxReqID)
	}
}
