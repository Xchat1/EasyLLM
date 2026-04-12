package handlers

import (
	"testing"
	"time"

	"easyllm/internal/models"
)

func TestMergePlatformAccountUpdatePreservesHiddenFields(t *testing.T) {
	cookie := "cookie-value"
	metadata := `{"source":"import"}`
	resetAt := time.Now().Add(2 * time.Hour)

	existing := &models.PlatformAccount{
		CookieToken:  &cookie,
		MetadataJSON: &metadata,
		QuotaResetAt: &resetAt,
	}
	incoming := models.PlatformAccount{
		Email: "user@example.com",
	}

	merged := mergePlatformAccountUpdate(existing, incoming)
	if merged.CookieToken == nil || *merged.CookieToken != cookie {
		t.Fatalf("expected cookie token to be preserved")
	}
	if merged.MetadataJSON == nil || *merged.MetadataJSON != metadata {
		t.Fatalf("expected metadata json to be preserved")
	}
	if merged.QuotaResetAt == nil || !merged.QuotaResetAt.Equal(resetAt) {
		t.Fatalf("expected quota reset time to be preserved")
	}
}

func TestMergePlatformInstanceUpdatePreservesRuntimeFields(t *testing.T) {
	startedAt := time.Now().Add(-time.Hour)
	stoppedAt := time.Now()
	existing := &models.PlatformInstance{
		LastStartedAt: &startedAt,
		LastStoppedAt: &stoppedAt,
	}

	merged := mergePlatformInstanceUpdate(existing, models.PlatformInstance{Name: "workspace-a"})
	if merged.LastStartedAt == nil || !merged.LastStartedAt.Equal(startedAt) {
		t.Fatalf("expected last_started_at to be preserved")
	}
	if merged.LastStoppedAt == nil || !merged.LastStoppedAt.Equal(stoppedAt) {
		t.Fatalf("expected last_stopped_at to be preserved")
	}
}

func TestMergeWakeupTaskUpdatePreservesRuntimeStatus(t *testing.T) {
	lastRunAt := time.Now().Add(-30 * time.Minute)
	nextRunAt := time.Now().Add(30 * time.Minute)
	status := "ok"
	message := "last run completed"
	existing := &models.WakeupTask{
		LastRunAt:   &lastRunAt,
		NextRunAt:   &nextRunAt,
		LastStatus:  &status,
		LastMessage: &message,
	}

	merged := mergeWakeupTaskUpdate(existing, models.WakeupTask{Name: "daily"})
	if merged.LastRunAt == nil || !merged.LastRunAt.Equal(lastRunAt) {
		t.Fatalf("expected last_run_at to be preserved")
	}
	if merged.NextRunAt == nil || !merged.NextRunAt.Equal(nextRunAt) {
		t.Fatalf("expected next_run_at to be preserved")
	}
	if merged.LastStatus == nil || *merged.LastStatus != status {
		t.Fatalf("expected last_status to be preserved")
	}
	if merged.LastMessage == nil || *merged.LastMessage != message {
		t.Fatalf("expected last_message to be preserved")
	}
}
