package workers

import (
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/services/emailval"
)

func TestValidationRetryDelay(t *testing.T) {
	tests := []struct {
		attempts int
		delay    time.Duration
		retry    bool
	}{
		{0, 0, false},
		{1, 15 * time.Minute, true},
		{2, time.Hour, true},
		{3, 4 * time.Hour, true},
		{4, 0, false},
	}
	for _, tt := range tests {
		delay, retry := validationRetryDelay(tt.attempts)
		if delay != tt.delay || retry != tt.retry {
			t.Errorf("validationRetryDelay(%d) = %v, %v; want %v, %v", tt.attempts, delay, retry, tt.delay, tt.retry)
		}
	}
}

func TestIsTransient(t *testing.T) {
	for _, sub := range []string{emailval.SubGreylisted, emailval.SubMailServerTempError, emailval.SubTimeoutExceeded} {
		if !isTransient(&emailval.Result{Status: emailval.StatusUnknown, SubStatus: sub}) {
			t.Errorf("unknown/%s should be transient", sub)
		}
	}
	if isTransient(&emailval.Result{Status: emailval.StatusUnknown, SubStatus: emailval.SubExceptionOccurred}) {
		t.Error("unknown/exception_occurred should be final")
	}
	if isTransient(&emailval.Result{Status: emailval.StatusInvalid, SubStatus: emailval.SubGreylisted}) {
		t.Error("only unknown results are retried")
	}
}
