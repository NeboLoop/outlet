package email

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// A refusal for now is recorded; a limit hold never is, so it waits as long
// as it must. Five days of refusals is the end.
func TestRefusedAtAndGaveUp(t *testing.T) {
	now := time.Now()
	assert.True(t, RefusedAt(&DeliveryError{Temporary: true, Err: errors.New("451")}, now).Valid)
	assert.False(t, RefusedAt(overLimit(now.Add(time.Hour), "daily limit"), now).Valid, "a limit hold is not a refusal")
	assert.False(t, RefusedAt(&DeliveryError{Temporary: false, Err: errors.New("550")}, now).Valid)

	assert.False(t, GaveUp(RefusedAt(&DeliveryError{Temporary: true, Err: errors.New("451")}, now.Add(-4*24*time.Hour)), now))
	assert.True(t, GaveUp(RefusedAt(&DeliveryError{Temporary: true, Err: errors.New("451")}, now.Add(-6*24*time.Hour)), now))
}
