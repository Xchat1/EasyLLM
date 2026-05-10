package openai

import (
	"net/http"
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

func intPtr(v int) *int { return &v }

func int64Ptr(v int64) *int64 { return &v }

func floatPtr(v float64) *float64 { return &v }
