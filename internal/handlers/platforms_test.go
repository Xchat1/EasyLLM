package handlers

import (
	"testing"

	"easyllm/internal/models"
)

func TestMergeCursorAccountUpdatePreservesManagedFields(t *testing.T) {
	cookie := "cookie"
	plan := "pro"
	existing := &models.CursorAccount{
		CookieToken: &cookie,
		Plan:        &plan,
		Active:      true,
	}
	incoming := models.CursorAccount{
		Email:       "user@example.com",
		AccessToken: "token",
	}

	merged := mergeCursorAccountUpdate(existing, incoming)
	if merged.CookieToken == nil || *merged.CookieToken != cookie {
		t.Fatalf("expected cookie token to be preserved")
	}
	if merged.Plan == nil || *merged.Plan != plan {
		t.Fatalf("expected plan to be preserved")
	}
	if !merged.Active {
		t.Fatalf("expected active flag to be preserved")
	}
}

func TestMergeAntigravityAccountUpdatePreservesManagedFields(t *testing.T) {
	plan := "starter"
	quota := int64(100)
	usedQuota := int64(55)
	existing := &models.AntigravityAccount{
		Plan:      &plan,
		Quota:     &quota,
		UsedQuota: &usedQuota,
		Active:    true,
	}
	incoming := models.AntigravityAccount{
		Email:       "user@example.com",
		AccessToken: "token",
	}

	merged := mergeAntigravityAccountUpdate(existing, incoming)
	if merged.Plan == nil || *merged.Plan != plan {
		t.Fatalf("expected plan to be preserved")
	}
	if merged.Quota == nil || *merged.Quota != quota {
		t.Fatalf("expected quota to be preserved")
	}
	if merged.UsedQuota == nil || *merged.UsedQuota != usedQuota {
		t.Fatalf("expected used quota to be preserved")
	}
	if !merged.Active {
		t.Fatalf("expected active flag to be preserved")
	}
}

func TestValidateCursorAccountInput(t *testing.T) {
	err := validateCursorAccountInput(models.CursorAccount{})
	if err == nil {
		t.Fatalf("expected validation error for empty account")
	}
}

func TestValidateAntigravityAccountInput(t *testing.T) {
	err := validateAntigravityAccountInput(models.AntigravityAccount{})
	if err == nil {
		t.Fatalf("expected validation error for empty account")
	}
}
