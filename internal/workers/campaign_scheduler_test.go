package workers

import (
	"sync"
	"testing"
	"time"
)

func TestDefaultCampaignSchedulerConfig(t *testing.T) {
	config := DefaultCampaignSchedulerConfig()

	if config.ScheduleInterval != 10*time.Second {
		t.Errorf("ScheduleInterval should be 10s, got %v", config.ScheduleInterval)
	}
	if config.SendPollInterval != 2*time.Second {
		t.Errorf("SendPollInterval should be 2s, got %v", config.SendPollInterval)
	}
	if config.Workers != 10 {
		t.Errorf("Workers should be 10, got %d", config.Workers)
	}
	if config.RateLimit != 14 {
		t.Errorf("RateLimit should be 14, got %d", config.RateLimit)
	}
	if config.BatchSize != 1000 {
		t.Errorf("BatchSize should be 1000, got %d", config.BatchSize)
	}
	if config.ErrorThreshold != 100 {
		t.Errorf("ErrorThreshold should be 100, got %d", config.ErrorThreshold)
	}
	if config.PoolSize != 20 {
		t.Errorf("PoolSize should be 20, got %d", config.PoolSize)
	}
}

func TestNewCampaignPipe(t *testing.T) {
	pipe := NewCampaignPipe("campaign-123")

	if pipe.CampaignID != "campaign-123" {
		t.Errorf("CampaignID should be campaign-123, got %s", pipe.CampaignID)
	}
	if pipe.sent.Load() != 0 {
		t.Errorf("sent should be 0, got %d", pipe.sent.Load())
	}
	if pipe.errors.Load() != 0 {
		t.Errorf("errors should be 0, got %d", pipe.errors.Load())
	}
	if pipe.stopped.Load() {
		t.Error("stopped should be false")
	}
	if pipe.paused.Load() {
		t.Error("paused should be false")
	}
	if pipe.errorsPaused.Load() {
		t.Error("errorsPaused should be false")
	}
}

func TestCampaignPipe_RecordSent(t *testing.T) {
	pipe := NewCampaignPipe("test")

	pipe.RecordSent()
	pipe.RecordSent()
	pipe.RecordSent()

	if pipe.sent.Load() != 3 {
		t.Errorf("sent should be 3, got %d", pipe.sent.Load())
	}
	if pipe.lastMinuteSent.Load() != 3 {
		t.Errorf("lastMinuteSent should be 3, got %d", pipe.lastMinuteSent.Load())
	}
}

func TestCampaignPipe_RecordSent_ResetsErrors(t *testing.T) {
	pipe := NewCampaignPipe("test")

	// Accumulate errors
	pipe.RecordError()
	pipe.RecordError()
	pipe.RecordError()

	if pipe.errors.Load() != 3 {
		t.Errorf("errors should be 3, got %d", pipe.errors.Load())
	}

	// Successful send resets consecutive errors
	pipe.RecordSent()

	if pipe.errors.Load() != 0 {
		t.Errorf("errors should be reset to 0 after success, got %d", pipe.errors.Load())
	}
}

func TestCampaignPipe_RecordError(t *testing.T) {
	pipe := NewCampaignPipe("test")

	count1 := pipe.RecordError()
	count2 := pipe.RecordError()
	count3 := pipe.RecordError()

	if count1 != 1 {
		t.Errorf("First error count should be 1, got %d", count1)
	}
	if count2 != 2 {
		t.Errorf("Second error count should be 2, got %d", count2)
	}
	if count3 != 3 {
		t.Errorf("Third error count should be 3, got %d", count3)
	}
}

func TestCampaignPipe_ShouldPause(t *testing.T) {
	pipe := NewCampaignPipe("test")

	// Should not pause initially
	if pipe.ShouldPause(5) {
		t.Error("Should not pause with 0 errors")
	}

	// Add errors but stay under threshold
	pipe.RecordError()
	pipe.RecordError()
	pipe.RecordError()
	pipe.RecordError()

	if pipe.ShouldPause(5) {
		t.Error("Should not pause with 4 errors (threshold 5)")
	}

	// Hit threshold
	pipe.RecordError()

	if !pipe.ShouldPause(5) {
		t.Error("Should pause with 5 errors (threshold 5)")
	}

	// Above threshold
	pipe.RecordError()

	if !pipe.ShouldPause(5) {
		t.Error("Should pause with 6 errors (threshold 5)")
	}
}

func TestCampaignPipe_Stats(t *testing.T) {
	pipe := NewCampaignPipe("test")

	// Initial stats
	sent, errors, _ := pipe.Stats()
	if sent != 0 || errors != 0 {
		t.Errorf("Initial stats should be 0/0, got %d/%d", sent, errors)
	}

	// After some activity
	pipe.RecordSent()
	pipe.RecordSent()
	pipe.RecordSent()
	pipe.RecordError()
	pipe.RecordError()

	sent, errors, _ = pipe.Stats()
	// Note: RecordSent resets errors, so after 3 sends and 2 errors, errors = 2
	if sent != 3 {
		t.Errorf("sent should be 3, got %d", sent)
	}
	if errors != 2 {
		t.Errorf("errors should be 2, got %d", errors)
	}
}

func TestCampaignPipe_Concurrent(t *testing.T) {
	pipe := NewCampaignPipe("test")
	var wg sync.WaitGroup

	// Concurrently record sends and errors
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			pipe.RecordSent()
		}()
		go func() {
			defer wg.Done()
			pipe.RecordError()
		}()
	}

	wg.Wait()

	sent, _, _ := pipe.Stats()
	if sent != 100 {
		t.Errorf("sent should be 100, got %d", sent)
	}
}

func TestCampaignPipe_PausedFlags(t *testing.T) {
	pipe := NewCampaignPipe("test")

	// Test paused flag
	pipe.paused.Store(true)
	if !pipe.paused.Load() {
		t.Error("paused should be true")
	}

	// Test errorsPaused flag
	pipe.errorsPaused.Store(true)
	if !pipe.errorsPaused.Load() {
		t.Error("errorsPaused should be true")
	}

	// Test stopped flag
	pipe.stopped.Store(true)
	if !pipe.stopped.Load() {
		t.Error("stopped should be true")
	}
}
