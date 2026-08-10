package services_test

import (
	"testing"

	"plumelauncher/internal/services"
)

func TestServiceErrorFormat(t *testing.T) {
	err := services.NewValidationError("name is required", "name")
	if err.Code != services.ErrCodeValidation {
		t.Errorf("Code = %q, want %q", err.Code, services.ErrCodeValidation)
	}
	if err.Message != "name is required" {
		t.Errorf("Message = %q", err.Message)
	}
	if len(err.Fields) != 1 || err.Fields[0] != "name" {
		t.Errorf("Fields = %v", err.Fields)
	}
}

func TestServiceErrorString(t *testing.T) {
	err := services.NewNotFoundError("instance not found")
	s := err.Error()
	if s != "[NOT_FOUND] instance not found" {
		t.Errorf("Error() = %q", s)
	}
}

func TestAllErrorCodes(t *testing.T) {
	tests := []struct {
		code services.ErrorCode
	}{
		{services.ErrCodeValidation},
		{services.ErrCodeNotFound},
		{services.ErrCodeConflict},
		{services.ErrCodeUnsupported},
		{services.ErrCodeIncompatible},
		{services.ErrCodeIntegrity},
		{services.ErrCodeNetwork},
		{services.ErrCodeUpstream},
		{services.ErrCodeCancelled},
		{services.ErrCodeInternal},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			var err *services.ServiceError
			switch tt.code {
			case services.ErrCodeValidation:
				err = services.NewValidationError("test")
			case services.ErrCodeNotFound:
				err = services.NewNotFoundError("test")
			case services.ErrCodeConflict:
				err = services.NewConflictError("test")
			case services.ErrCodeUnsupported:
				err = services.NewUnsupportedError("test")
			case services.ErrCodeIncompatible:
				err = services.NewIncompatibleError("test")
			case services.ErrCodeIntegrity:
				err = services.NewIntegrityError("test")
			case services.ErrCodeNetwork:
				err = services.NewNetworkError("test")
			case services.ErrCodeUpstream:
				err = services.NewUpstreamError("test")
			case services.ErrCodeCancelled:
				err = services.NewCancelledError("op-1")
			case services.ErrCodeInternal:
				err = services.NewInternalError("test")
			}
			if err == nil {
				t.Fatal("expected non-nil error")
			}
			if err.Code != tt.code {
				t.Errorf("Code = %q, want %q", err.Code, tt.code)
			}
		})
	}
}
