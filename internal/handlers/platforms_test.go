package handlers

import (
	"testing"

	"easyllm/internal/models"
)

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

func TestValidateAntigravityAccountInput(t *testing.T) {
	err := validateAntigravityAccountInput(models.AntigravityAccount{})
	if err == nil {
		t.Fatalf("expected validation error for empty account")
	}
}
