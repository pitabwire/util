package util

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
)

// JSONResponse represents an HTTP response which contains a JSON body.
type JSONResponse struct {
	// HTTP status code.
	Code int
	// JSON represents the JSON that should be serialised and sent to the client
	JSON interface{}
	// Headers represent any headers that should be sent to the client
	Headers map[string]any
}

// Is2xx returns true if the Code is between 200 and 299.
func (r JSONResponse) Is2xx() bool {
	return r.Code/100 == Status2xx
}

// RedirectResponse returns a JSONResponse which 302s the client to the given location.
func RedirectResponse(location string) JSONResponse {
	headers := make(map[string]any)
	headers["Location"] = location
	return JSONResponse{
		Code:    StatusFound, // 302
		JSON:    struct{}{},
		Headers: headers,
	}
}

// MessageResponse returns a JSONResponse with a 'message' key containing the given text.
func MessageResponse(code int, msg string) JSONResponse {
	return JSONResponse{
		Code: code,
		JSON: struct {
			Message string `json:"message"`
		}{msg},
	}
}

// ErrorResponse returns an HTTP 500 JSONResponse with the stringified form of the given error.
func ErrorResponse(err error) JSONResponse {
	return MessageResponse(StatusInternalServerError, err.Error())
}

// MatrixErrorResponse is a function that returns error responses in the standard Matrix Error format (errcode / error).
func MatrixErrorResponse(httpStatusCode int, errCode, message string) JSONResponse {
	return JSONResponse{
		Code: httpStatusCode,
		JSON: struct {
			ErrCode string `json:"errcode"`
			Error   string `json:"error"`
		}{errCode, message},
	}
}

// JSONRequestHandler represents an interface that must be satisfied in order to respond to incoming
// HTTP requests with JSON.
type JSONRequestHandler interface {
	OnIncomingRequest(req *http.Request) JSONResponse
}

// jsonRequestHandlerWrapper is a wrapper to allow in-line functions to conform to util.JSONRequestHandler.
type jsonRequestHandlerWrapper struct {
	function func(req *http.Request) JSONResponse
}

// OnIncomingRequest implements util.JSONRequestHandler.
func (r *jsonRequestHandlerWrapper) OnIncomingRequest(req *http.Request) JSONResponse {
	return r.function(req)
}

// NewJSONRequestHandler converts the given OnIncomingRequest function into a JSONRequestHandler.
func NewJSONRequestHandler(f func(req *http.Request) JSONResponse) JSONRequestHandler {
	return &jsonRequestHandlerWrapper{f}
}

// Protect panicking HTTP requests from taking down the entire process, and log them using
// the correct logger, returning a 500 with a JSON response rather than abruptly closing the
// connection. The http.Request MUST have a ctxValueLogger.
func Protect(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				logger := Log(req.Context())
				logger.WithField("panic", r).Error(
					"Request panicked!\n%s", debug.Stack(),
				)
				respond(w, req, MessageResponse(StatusInternalServerError, "Internal Server Error"))
			}
		}()
		handler(w, req)
	}
}

// RequestWithLogging sets up standard logging for http.Requests.
// http.Requests will have a logger (with a request ID/method/path logged) attached to the Context.
// This can be accessed via GetLogger(Context).
func RequestWithLogging(req *http.Request) *http.Request {
	reqID := RandomString(DefaultRequestIDLength)
	// Set a Logger and request ID on the context
	ctx := ContextWithLogger(req.Context(), Log(req.Context()).
		WithField("req.method", req.Method).
		WithField("req.path", req.URL.Path).
		WithField("req.id", reqID))
	ctx = context.WithValue(ctx, ctxValueRequestID, reqID)
	req = req.WithContext(ctx)

	if req.Method != http.MethodOptions {
		logger := Log(req.Context())
		logger.Trace("Incoming request")
	}

	return req
}

// MakeJSONAPI creates an HTTP handler which always responds to incoming requests with JSON responses.
// Incoming http.Requests will have a logger (with a request ID/method/path logged) attached to the Context.
// This can be accessed via GetLogger(Context).
func MakeJSONAPI(handler JSONRequestHandler) http.HandlerFunc {
	return Protect(func(w http.ResponseWriter, req *http.Request) {
		req = RequestWithLogging(req)

		if req.Method == http.MethodOptions {
			SetCORSHeaders(w)
			w.WriteHeader(http.StatusOK)
			return
		}
		res := handler.OnIncomingRequest(req)

		// Set common headers returned regardless of the outcome of the request
		w.Header().Set("Content-Type", "application/json")
		SetCORSHeaders(w)

		respond(w, req, res)
	})
}

func respond(w http.ResponseWriter, req *http.Request, res JSONResponse) {
	logger := Log(req.Context())

	// Set custom headers
	if res.Headers != nil {
		for h, val := range res.Headers {
			var headerValues []any

			// Check if the value is already a headerValues
			if reflect.TypeOf(val).Kind() == reflect.Slice {
				v := reflect.ValueOf(val)
				for i := range v.Len() {
					headerValues = append(headerValues, v.Index(i).Interface())
				}
			} else {
				// If not a headerValues, wrap it in a headerValues
				headerValues = []any{val}
			}

			// Iterate over the headerValues and validate each element
			for _, item := range headerValues {
				switch v := item.(type) {
				case string:
					w.Header().Add(h, v)
				case *http.Cookie:
					http.SetCookie(w, v)
				default:
					w.Header().Add(h, fmt.Sprintf("%v", v))
				}
			}
		}
	}

	// Marshal JSON response into raw bytes to send as the HTTP body
	resBytes, err := json.Marshal(res.JSON)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal JSONResponse")
		// this should never fail to be marshalled so drop err to the floor
		res = MessageResponse(StatusInternalServerError, "Internal Server Error")
		resBytes, _ = json.Marshal(res.JSON)
	}

	// Set status code and write the body
	w.WriteHeader(res.Code)
	if req.Method != http.MethodOptions {
		logger.WithField("code", res.Code).WithField("bytes", len(resBytes)).Trace("Responding")
	}
	_, _ = w.Write(resBytes)
}

// WithCORSOptions intercepts all OPTIONS requests and responds with CORS headers. The request handler
// is not invoked when this happens.
func WithCORSOptions(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodOptions {
			SetCORSHeaders(w)
			return
		}
		handler(w, req)
	}
}

// SetCORSHeaders sets unrestricted origin Access-Control headers on the response writer.
func SetCORSHeaders(w http.ResponseWriter) {
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
}

const (
	StatusFound               = 302
	StatusInternalServerError = 500
	DefaultRequestIDLength    = 12
	Status2xx                 = 2
)
