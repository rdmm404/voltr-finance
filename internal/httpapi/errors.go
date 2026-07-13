package httpapi

import (
	"log/slog"
	"net/http"

	"rdmm404/voltr-finance/internal/api"
	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

func WriteApplicationError(w http.ResponseWriter, request *http.Request, logger *slog.Logger, err error) {
	status, response := MapApplicationError(err)
	if status == http.StatusInternalServerError && logger != nil {
		logger.Error("request failed", "method", request.Method, "path", request.URL.Path, "code", response.Error.Code)
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
