package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

func TestDecodeJSONIsStrictAndCollectionsAreArrays(t *testing.T) {
	type requestBody struct {
		Name string `json:"name"`
	}
	for name, body := range map[string]string{"unknown field": `{"name":"ok","secret":true}`, "trailing value": `{"name":"ok"} {}`, "missing body": ``} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			recorder := httptest.NewRecorder()
			var decoded requestBody
			if err := DecodeJSON(recorder, request, &decoded); err == nil {
				t.Fatal("DecodeJSON accepted invalid body")
			}
		})
	}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
	recorder := httptest.NewRecorder()
	var decoded requestBody
	if err := DecodeJSON(recorder, request, &decoded); err != nil || decoded.Name != "ok" {
		t.Fatalf("decoded=%+v error=%v", decoded, err)
	}
	WriteJSON(recorder, http.StatusOK, NonNilSlice([]int(nil)))
	if !strings.Contains(recorder.Body.String(), "[]") {
		t.Fatalf("body=%q", recorder.Body.String())
	}
}

func TestApplicationErrorsMapWithoutLeakingInternalDetails(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{apperrors.Validation("bad input"), 400},
		{apperrors.NotFound(apperrors.CodeUserNotFound, "user not found", nil), 404},
		{apperrors.Conflict(apperrors.CodeCategoryConflict, "category conflict", nil), 409},
		{errors.New("password=database-secret"), 500},
	}
	for _, test := range tests {
		status, response := MapApplicationError(test.err)
		if status != test.status {
			t.Fatalf("MapApplicationError(%v) status=%d want=%d", test.err, status, test.status)
		}
		encoded, _ := json.Marshal(response)
		if bytes.Contains(encoded, []byte("database-secret")) || bytes.Contains(encoded, []byte("password=")) {
			t.Fatalf("response leaks cause: %s", encoded)
		}
	}
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	recorder := httptest.NewRecorder()
	internal := apperrors.WrapInternal("load budget report", errors.New("driver detail"))
	WriteApplicationError(recorder, httptest.NewRequest(http.MethodGet, "/v1/test", nil), logger, internal)
	if strings.Contains(recorder.Body.String(), "driver detail") || strings.Contains(logs.String(), "driver detail") {
		t.Fatalf("detail leaked response=%q logs=%q", recorder.Body.String(), logs.String())
	}
	if !strings.Contains(logs.String(), "operation=\"load budget report\"") || !strings.Contains(logs.String(), "cause_type=*errors.errorString") {
		t.Fatalf("diagnostic context missing: %q", logs.String())
	}
}

func TestBearerAPIKeyProtectsV1WithoutDisclosingKeys(t *testing.T) {
	const configured = "configured-secret"
	handler, err := NewHandler(configured, func(router *Router) {
		router.HandleFunc(http.MethodGet, "/v1/test", func(w http.ResponseWriter, _ *http.Request) { WriteJSON(w, 200, map[string]bool{"ok": true}) })
	})
	if err != nil {
		t.Fatal(err)
	}
	live := httptest.NewRecorder()
	handler.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/live", nil))
	if live.Code != 200 {
		t.Fatalf("live status=%d", live.Code)
	}
	for name, authorization := range map[string]string{"missing": "", "malformed": "Basic configured-secret", "incorrect": "Bearer supplied-secret"} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
			if authorization != "" {
				request.Header.Set("Authorization", authorization)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != 401 {
				t.Fatalf("status=%d", recorder.Code)
			}
			body := recorder.Body.String()
			if strings.Contains(body, configured) || strings.Contains(body, "supplied-secret") {
				t.Fatalf("credential leaked: %q", body)
			}
		})
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	request.Header.Set("Authorization", "Bearer "+configured)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != 200 {
		t.Fatalf("authorized status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if _, err := NewHandler("", nil); err == nil {
		t.Fatal("NewHandler accepted empty API key")
	}
}

func TestRouterProvidesJSONNotFoundMethodAndPathParsing(t *testing.T) {
	router := NewRouter()
	router.HandleFunc(http.MethodGet, "/v1/items/{id}", func(w http.ResponseWriter, request *http.Request) {
		id, err := ParsePathID(request, "id")
		if err != nil {
			WriteValidationError(w, err.Error())
			return
		}
		WriteJSON(w, 200, map[string]int64{"id": id})
	})
	request := httptest.NewRequest(http.MethodGet, "/v1/items/42", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "42") {
		t.Fatalf("get status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/items/42", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 405 || recorder.Header().Get("Allow") != "GET" || !strings.Contains(recorder.Body.String(), "method_not_allowed") {
		t.Fatalf("method status=%d allow=%q body=%q", recorder.Code, recorder.Header().Get("Allow"), recorder.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/v1/missing", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 404 || !strings.Contains(recorder.Body.String(), "not_found") {
		t.Fatalf("missing status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestServerAppliesTimeoutDefaultsAndOverrides(t *testing.T) {
	server, err := NewServer(Config{APIKey: "key", WriteTimeout: 9 * time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if server.Addr != ":8080" || server.ReadHeaderTimeout != 5*time.Second || server.ReadTimeout != 15*time.Second || server.WriteTimeout != 9*time.Second || server.IdleTimeout != 60*time.Second {
		t.Fatalf("server=%+v", server)
	}
	if _, err := NewServer(Config{}, nil); err == nil {
		t.Fatal("NewServer accepted empty API key")
	}
}
