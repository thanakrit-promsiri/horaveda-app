package logger

import (
	"testing"
)

func TestNewLogger(t *testing.T) {
	log := NewLogger()
	if log == nil {
		t.Fatal("expected non-nil SugaredLogger")
	}
	log.Info("test logger initialized")
}
