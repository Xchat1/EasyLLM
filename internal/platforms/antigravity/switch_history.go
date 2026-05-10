// Package antigravity — switch history tracking.
// Logic ported from cockpit-tools-main/src-tauri/src/modules/antigravity_switch_history.rs.
package antigravity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	maxHistoryItems = 200
)

var historyMu sync.Mutex

// ---- public types (mirrors AntigravitySwitchHistoryItem) ----

// AutoSwitchHitGroup mirrors AntigravityAutoSwitchHitGroup.
type AutoSwitchHitGroup struct {
	GroupID    string `json:"group_id"`
	GroupName  string `json:"group_name"`
	Percentage int32  `json:"percentage"`
}

// AutoSwitchReason mirrors AntigravityAutoSwitchReason.
type AutoSwitchReason struct {
	Rule                    string               `json:"rule"`
	Threshold               int32                `json:"threshold"`
	ScopeMode               string               `json:"scope_mode"`
	CreditsEnabled          bool                 `json:"credits_enabled"`
	CreditsThreshold        *int32               `json:"credits_threshold,omitempty"`
	CreditsTriggered        bool                 `json:"credits_triggered"`
	CurrentCreditsRemaining *float64             `json:"current_credits_remaining,omitempty"`
	SelectedGroupIDs        []string             `json:"selected_group_ids"`
	SelectedGroupNames      []string             `json:"selected_group_names"`
	HitGroups               []AutoSwitchHitGroup `json:"hit_groups"`
	CandidateCount          int                  `json:"candidate_count"`
	SelectedPolicy          string               `json:"selected_policy"`
}

// SwitchHistoryItem mirrors AntigravitySwitchHistoryItem.
type SwitchHistoryItem struct {
	ID                    string            `json:"id"`
	Timestamp             int64             `json:"timestamp"`
	AccountID             string            `json:"account_id"`
	TargetEmail           string            `json:"target_email"`
	TriggerType           string            `json:"trigger_type"`
	TriggerSource         string            `json:"trigger_source"`
	LocalOK               bool              `json:"local_ok"`
	SeamlessOK            bool              `json:"seamless_ok"`
	Success               bool              `json:"success"`
	LocalDurationMS       uint64            `json:"local_duration_ms"`
	SeamlessDurationMS    *uint64           `json:"seamless_duration_ms,omitempty"`
	TotalDurationMS       uint64            `json:"total_duration_ms"`
	ErrorStage            *string           `json:"error_stage,omitempty"`
	ErrorCode             *string           `json:"error_code,omitempty"`
	ErrorMessage          *string           `json:"error_message,omitempty"`
	SeamlessEffectiveMode *string           `json:"seamless_effective_mode,omitempty"`
	SeamlessFromEmail     *string           `json:"seamless_from_email,omitempty"`
	SeamlessToEmail       *string           `json:"seamless_to_email,omitempty"`
	SeamlessExecutionID   *string           `json:"seamless_execution_id,omitempty"`
	SeamlessFinishedAt    *string           `json:"seamless_finished_at,omitempty"`
	AutoSwitchReason      *AutoSwitchReason `json:"auto_switch_reason,omitempty"`
}

// NewSwitchHistoryItem creates a new item with defaults.
func NewSwitchHistoryItem(accountID, targetEmail string) SwitchHistoryItem {
	return SwitchHistoryItem{
		ID:            fmt.Sprintf("sw_%d", time.Now().UnixMilli()),
		Timestamp:     time.Now().Unix(),
		AccountID:     accountID,
		TargetEmail:   targetEmail,
		TriggerType:   "manual",
		TriggerSource: "tools.account.switch",
	}
}

// ---- storage ----

func historyFilePath(dataDir string) string {
	return filepath.Join(dataDir, "antigravity_switch_history.json")
}

// LoadHistory loads the switch history from disk.
// Mirrors load_history in antigravity_switch_history.rs.
func LoadHistory(dataDir string) ([]SwitchHistoryItem, error) {
	path := historyFilePath(dataDir)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []SwitchHistoryItem{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read antigravity switch history: %w", err)
	}
	if len(data) == 0 {
		return []SwitchHistoryItem{}, nil
	}
	var items []SwitchHistoryItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse antigravity switch history: %w", err)
	}
	return items, nil
}

func saveHistory(dataDir string, items []SwitchHistoryItem) error {
	path := historyFilePath(dataDir)
	tmpPath := path + ".tmp"
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal antigravity switch history: %w", err)
	}
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write antigravity switch history tmp: %w", err)
	}
	return os.Rename(tmpPath, path)
}

// AddHistoryItem appends a new item to the switch history (thread-safe).
// Mirrors add_history_item in antigravity_switch_history.rs.
func AddHistoryItem(dataDir string, item SwitchHistoryItem) error {
	historyMu.Lock()
	defer historyMu.Unlock()

	existing, err := LoadHistory(dataDir)
	if err != nil {
		existing = []SwitchHistoryItem{}
	}

	// remove duplicate by ID
	filtered := existing[:0]
	for _, e := range existing {
		if e.ID != item.ID {
			filtered = append(filtered, e)
		}
	}
	filtered = append(filtered, item)

	// sort descending by timestamp
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp > filtered[j].Timestamp
	})

	// truncate
	if len(filtered) > maxHistoryItems {
		filtered = filtered[:maxHistoryItems]
	}

	return saveHistory(dataDir, filtered)
}

// ClearHistory removes all switch history entries.
// Mirrors clear_history in antigravity_switch_history.rs.
func ClearHistory(dataDir string) error {
	historyMu.Lock()
	defer historyMu.Unlock()
	return saveHistory(dataDir, []SwitchHistoryItem{})
}
