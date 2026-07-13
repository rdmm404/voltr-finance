package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"rdmm404/voltr-finance/internal/api"
)

func BearerAPIKey(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if !validBearer(request.Header.Values("Authorization"), apiKey) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			WriteJSON(w, http.StatusUnauthorized, api.ErrorResponse{Error: api.Error{Code: "authentication_error", Message: "authentication required"}})
			return
		}
		next.ServeHTTP(w, request)
	})
}

func validBearer(headers []string, configured string) bool {
	if len(headers) != 1 || configured == "" {
		return false
	}
	parts := strings.Fields(headers[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(parts[1]), []byte(configured)) == 1
}
