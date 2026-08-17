package apierr

import (
	"encoding/json"
	"net/http"
)

type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type Conflict struct {
	ConflictingSessionID string `json:"conflictingSessionId,omitempty"`
	ConflictingTitle     string `json:"conflictingTitle,omitempty"`
	CurrentVersion       int    `json:"currentVersion,omitempty"`
	CurrentState         any    `json:"currentState,omitempty"`
}

type Body struct {
	Code        string       `json:"code"`
	Reason      string       `json:"reason"`
	FieldErrors []FieldError `json:"fieldErrors,omitempty"`
	Conflict    *Conflict    `json:"conflict,omitempty"`
}

func Write(w http.ResponseWriter, status int, body Body) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func Simple(w http.ResponseWriter, status int, code, reason string) {
	Write(w, status, Body{Code: code, Reason: reason})
}

func Validation(w http.ResponseWriter, reason string, fields ...FieldError) {
	Write(w, http.StatusBadRequest, Body{
		Code:        "VALIDATION_ERROR",
		Reason:      reason,
		FieldErrors: fields,
	})
}
