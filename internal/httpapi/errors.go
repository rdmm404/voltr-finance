package httpapi

import (
	"log/slog"
	"net/http"

	"rdmm404/voltr-finance/internal/api"
	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

type HandlerSupport struct{ logger *slog.Logger }

func NewHandlerSupport(logger *slog.Logger) *HandlerSupport {
	if logger == nil {
		logger = slog.Default()
	}
	return &HandlerSupport{logger: logger}
}

func HandlerSupportOrDefault(support ...*HandlerSupport) *HandlerSupport {
	if len(support) > 0 && support[0] != nil {
		return support[0]
	}
	return NewHandlerSupport(nil)
}

func (s *HandlerSupport) Decode(w http.ResponseWriter, request *http.Request, value any) bool {
	if err := DecodeJSON(w, request, value); err != nil {
		WriteValidationError(w, err.Error())
		return false
	}
	return true
}

func (s *HandlerSupport) Fail(w http.ResponseWriter, request *http.Request, err error) {
	WriteApplicationError(w, request, s.logger, err)
}

func WriteApplicationError(w http.ResponseWriter, request *http.Request, logger *slog.Logger, err error) {
	status, response := MapApplicationError(err)
	if status == http.StatusInternalServerError {
		if logger == nil {
			logger = slog.Default()
		}
		operation, causeType := apperrors.Diagnostic(err)
		logger.Error("request failed", "method", request.Method, "path", request.URL.Path, "code", response.Error.Code, "operation", operation, "cause_type", causeType)
	}
	WriteJSON(w, status, response)
}

func MapApplicationError(err error) (int, api.ErrorResponse) {
	appError, ok := apperrors.As(err)
	if !ok {
		return safeInternalError()
	}
	response := api.ErrorResponse{Error: api.Error{Code: string(appError.Code), Message: appError.Message}}
	switch appError.Kind {
	case apperrors.KindValidation:
		return http.StatusBadRequest, response
	case apperrors.KindNotFound:
		return http.StatusNotFound, response
	case apperrors.KindConflict:
		return http.StatusConflict, response
	default:
		return safeInternalError()
	}
}

func WriteValidationError(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: api.Error{Code: string(apperrors.CodeValidation), Message: message}})
}

func WriteNotFound(w http.ResponseWriter) {
	WriteJSON(w, http.StatusNotFound, api.ErrorResponse{Error: api.Error{Code: "not_found", Message: "route not found"}})
}

func WriteMethodNotAllowed(w http.ResponseWriter, allowed string) {
	if allowed != "" {
		w.Header().Set("Allow", allowed)
	}
	WriteJSON(w, http.StatusMethodNotAllowed, api.ErrorResponse{Error: api.Error{Code: "method_not_allowed", Message: "method not allowed"}})
}

func safeInternalError() (int, api.ErrorResponse) {
	return http.StatusInternalServerError, api.ErrorResponse{Error: api.Error{Code: string(apperrors.CodeInternal), Message: "internal error"}}
}
