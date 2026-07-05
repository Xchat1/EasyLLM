package openai

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestParseQuotaFromUsageClassifiesWeeklyPrimaryWindow(t *testing.T) {
	plan := "chatgpt_plus"
	usage := &usageResponse{
		PlanType: &plan,
		RateLimit: &rateLimitInfo{
			PrimaryWindow: &windowInfo{
				UsedPercent:        intPtr(12),
				LimitWindowSeconds: int64Ptr(604800),
				ResetAfterSeconds:  int64Ptr(603281),
			},
		},
		CodeReviewRateLimit: &rateLimitInfo{
			PrimaryWindow: &windowInfo{
				UsedPercent:        intPtr(0),
				LimitWindowSeconds: int64Ptr(604800),
				ResetAfterSeconds:  int64Ptr(604800),
			},
		},
	}

	info := parseQuotaFromUsage(usage)
	if info.Codex7dUsedPercent == nil || *info.Codex7dUsedPercent != 12 {
		t.Fatalf("expected weekly quota to map into 7d slot, got %#v", info.Codex7dUsedPercent)
	}
	if info.Codex5hUsedPercent != nil {
		t.Fatalf("expected 5h slot to stay empty when response only contains weekly windows, got %#v", info.Codex5hUsedPercent)
	}
	if info.PlanType == nil || *info.PlanType != "plus" {
		t.Fatalf("expected plus plan to be normalized from usage payload, got %#v", info.PlanType)
	}
}

func TestParseCodexHeadersClassifiesSingleWeeklyPrimaryWindow(t *testing.T) {
	headers := http.Header{
		"X-Codex-Primary-Used-Percent":        []string{"18"},
		"X-Codex-Primary-Reset-After-Seconds": []string{"603188"},
		"X-Codex-Primary-Window-Minutes":      []string{"10080"},
	}

	info := ParseCodexHeaders(headers)
	if info.Codex7dUsedPercent == nil || *info.Codex7dUsedPercent != 18 {
		t.Fatalf("expected weekly primary window to map into 7d slot, got %#v", info.Codex7dUsedPercent)
	}
	if info.Codex5hUsedPercent != nil {
		t.Fatalf("expected 5h slot to remain empty for weekly-only headers, got %#v", info.Codex5hUsedPercent)
	}
}

func TestMergeQuotaInfoFillsMissing5hFromCodexHeaders(t *testing.T) {
	plan := "plus"
	usageInfo := &QuotaInfo{
		Codex7dUsedPercent:   floatPtr(18),
		Codex7dResetSeconds:  int64Ptr(603188),
		Codex7dWindowMinutes: int64Ptr(10080),
		Total:                100,
		Used:                 18,
		Remaining:            82,
		ResetAt:              "6d23h",
		PlanType:             &plan,
	}
	headerInfo := &QuotaInfo{
		Codex5hUsedPercent:   floatPtr(64),
		Codex5hResetSeconds:  int64Ptr(13024),
		Codex5hWindowMinutes: int64Ptr(300),
	}

	merged := mergeQuotaInfo(usageInfo, headerInfo)
	if merged.Codex5hUsedPercent == nil || *merged.Codex5hUsedPercent != 64 {
		t.Fatalf("expected 5h quota to be filled from headers, got %#v", merged.Codex5hUsedPercent)
	}
	if merged.Codex7dUsedPercent == nil || *merged.Codex7dUsedPercent != 18 {
		t.Fatalf("expected 7d quota to stay from usage payload, got %#v", merged.Codex7dUsedPercent)
	}
	if merged.PlanType == nil || *merged.PlanType != "plus" {
		t.Fatalf("expected plan to stay from usage payload, got %#v", merged.PlanType)
	}
}

func TestNormalizePlanTypeK12(t *testing.T) {
	for _, input := range []string{"k12", "K12", "K-12", "chatgpt_k12"} {
		if got := NormalizePlanType(input); got != "k12" {
			t.Fatalf("NormalizePlanType(%q) = %q, want k12", input, got)
		}
	}
}

func TestCombineQuotaFetchResultsUsesCodexHeadersWhenUsageIsForbidden(t *testing.T) {
	usageInfo := &QuotaInfo{IsForbidden: true}
	headerInfo := &QuotaInfo{
		Codex5hUsedPercent:   floatPtr(42),
		Codex5hResetSeconds:  int64Ptr(1200),
		Codex5hWindowMinutes: int64Ptr(300),
	}

	got, err := combineQuotaFetchResults(usageInfo, nil, headerInfo, nil)
	if err != nil {
		t.Fatalf("combineQuotaFetchResults returned error: %v", err)
	}
	if got == nil || got.IsForbidden {
		t.Fatalf("expected Codex header quota to override forbidden usage response, got %#v", got)
	}
	if got.Codex5hUsedPercent == nil || *got.Codex5hUsedPercent != 42 {
		t.Fatalf("expected 5h quota from headers, got %#v", got.Codex5hUsedPercent)
	}
}

func TestCombineQuotaFetchResultsKeepsForbiddenWhenHeadersUnavailable(t *testing.T) {
	usageInfo := &QuotaInfo{
		IsForbidden:     true,
		HTTPStatus:      http.StatusForbidden,
		ForbiddenReason: `HTTP 403: {"code":"deactivated_workspace"}`,
	}

	got, err := combineQuotaFetchResults(usageInfo, nil, nil, errors.New("headers unavailable"))
	if err != nil {
		t.Fatalf("combineQuotaFetchResults returned error: %v", err)
	}
	if got == nil || !got.IsForbidden {
		t.Fatalf("expected forbidden usage result when headers are unavailable, got %#v", got)
	}
	if got.HTTPStatus != http.StatusForbidden {
		t.Fatalf("expected HTTP status to be preserved, got %d", got.HTTPStatus)
	}
	if !strings.Contains(got.ForbiddenReason, "deactivated_workspace") {
		t.Fatalf("expected forbidden reason to be preserved, got %q", got.ForbiddenReason)
	}
}

func intPtr(v int) *int { return &v }

func int64Ptr(v int64) *int64 { return &v }

func floatPtr(v float64) *float64 { return &v }
