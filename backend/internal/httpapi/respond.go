package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// errEnvelope is the single error shape every endpoint returns:
// {"error": {"code": "...", "message": "..."}}. Clients branch on code and
// show message only as a fallback.
type errEnvelope struct {
	Error errBody `json:"error"`
}

type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Stable error codes shared with the frontend (see the API consumption brief).
const (
	codeValidation         = "validation"
	codeUnauthorized       = "unauthorized"
	codeInvalidCredentials = "invalid_credentials"
	codeInvalidToken       = "invalid_token"
	codeForbidden          = "forbidden"
	codeNotFound           = "not_found"
	codeEmailTaken         = "email_taken"
	codeDuplicate          = "duplicate"
	codeDuplicateOrder     = "duplicate_order"
	codeEmptyCart          = "empty_cart"
	codeUnavailable        = "unavailable"
	codeInsufficientPoints = "insufficient_points"
	codeInvalidTransition  = "invalid_transition"
	codeRateLimited        = "rate_limited"
	codePaymentInitFailed  = "payment_init_failed"
	codeInternal           = "internal"
)

// writeJSON encodes v as a JSON response with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			slog.Error("encoding response", "error", err)
		}
	}
}

// writeError emits the standard error envelope.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errEnvelope{Error: errBody{Code: code, Message: message}})
}

// decodeJSON reads a JSON request body into dst, rejecting unknown fields and
// oversized bodies. It returns false (and writes a 400) on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, codeValidation, "invalid request body")
		return false
	}
	return true
}
