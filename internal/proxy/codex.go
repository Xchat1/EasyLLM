package proxy

import (
	"easyllm/internal/models"
	openaiplatform "easyllm/internal/platforms/openai"
	"easyllm/internal/storage"
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// poolEntry is a unified proxy pool entry from any source
type poolEntry struct {
	id               string
	email            string
	accessToken      string
	chatgptAccountID string // chatgpt-account-id header value (OAuth accounts)
	source           string // "codex" | "openai"
	requests         *int64
	planRank         int
	remainingQuota   *int
	expiresAt        *time.Time
}

// CodexProxy manages the unified proxy pool
type CodexProxy struct {
	mu           sync.RWMutex
	pool         []poolEntry
	tokenIndex   map[string]*poolEntry // token → poolEntry for O(1) lookup
	strategy     string
	currentIndex int64
	codexDB      *storage.CodexStorage
	openaiDB     *storage.OpenAIStorage
	enabled      bool
	httpClient   *http.Client
}

var globalProxy *CodexProxy
var proxyMu sync.Mutex

func GetProxy() *CodexProxy {
	proxyMu.Lock()
	defer proxyMu.Unlock()
	return globalProxy
}

func InitProxy(codexDB *storage.CodexStorage, openaiDB *storage.OpenAIStorage, strategy string) *CodexProxy {
	proxyMu.Lock()
	defer proxyMu.Unlock()
	globalProxy = &CodexProxy{
		codexDB:    codexDB,
		openaiDB:   openaiDB,
		strategy:   strategy,
		enabled:    true,
		tokenIndex: make(map[string]*poolEntry),
		httpClient: &http.Client{
			Timeout: 180 * time.Second,
			Transport: &http.Transport{
				DisableCompression:  true,
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
	globalProxy.Refresh()
	return globalProxy
}

// Refresh reloads the pool from both CodexAccount table and proxy-enabled OpenAI OAuth accounts.
// Existing in-memory request counters are preserved for entries that remain in the pool.
func (p *CodexProxy) Refresh() {
	// Snapshot existing counters before taking the write lock
	p.mu.RLock()
	oldCounters := make(map[string]*int64, len(p.pool))
	for i := range p.pool {
		oldCounters[p.pool[i].id] = p.pool[i].requests
	}
	p.mu.RUnlock()

	var entries []poolEntry
	selectedIDs := loadLocalAccessAccountIDSet()

	// Dedicated Codex pool accounts
	if p.codexDB != nil {
		accounts, err := p.codexDB.LoadEnabledAccounts()
		if err == nil {
			for _, a := range accounts {
				if selectedIDs != nil && !selectedIDs[a.ID] {
					continue
				}
				cnt := a.RequestCount
				if old, ok := oldCounters[a.ID]; ok && old != nil {
					cnt = atomic.LoadInt64(old)
				}
				entries = append(entries, poolEntry{
					id:          a.ID,
					email:       a.Email,
					accessToken: a.AccessToken,
					source:      "codex",
					requests:    &cnt,
				})
			}
		}
	}

	// OpenAI OAuth accounts with proxy_enabled = true
	if p.openaiDB != nil {
		accounts, err := p.openaiDB.ListProxyEnabled()
		if err == nil {
			for _, a := range accounts {
				if a.AccessToken == nil || *a.AccessToken == "" {
					continue
				}
				if selectedIDs != nil && !selectedIDs[a.ID] {
					continue
				}
				cnt := int64(0)
				if old, ok := oldCounters[a.ID]; ok && old != nil {
					cnt = atomic.LoadInt64(old)
				}
				accountID := ""
				if a.ChatGPTAccountID != nil {
					accountID = *a.ChatGPTAccountID
				}
				entries = append(entries, poolEntry{
					id:               a.ID,
					email:            a.Email,
					accessToken:      *a.AccessToken,
					chatgptAccountID: accountID,
					source:           "openai",
					requests:         &cnt,
					planRank:         planRank(a.Plan),
					remainingQuota:   remainingQuota(&a),
					expiresAt:        a.ExpiresAt,
				})
			}
		}
	}

	// Build token→entry index for O(1) lookup in matchIncomingToken / IsKnownToken
	idx := make(map[string]*poolEntry, len(entries))
	for i := range entries {
		idx[entries[i].accessToken] = &entries[i]
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.pool = entries
	p.tokenIndex = idx
}

func (p *CodexProxy) GetPoolStatus() *models.CodexPoolStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var totalRequests int64
	for _, e := range p.pool {
		if e.requests != nil {
			totalRequests += *e.requests
		}
	}

	accts := make([]models.CodexAccount, len(p.pool))
	for i, e := range p.pool {
		cnt := int64(0)
		if e.requests != nil {
			cnt = *e.requests
		}
		accts[i] = models.CodexAccount{
			ID:           e.id,
			Email:        e.email,
			AccessToken:  "",
			Enabled:      true,
			RequestCount: cnt,
		}
	}

	return &models.CodexPoolStatus{
		TotalAccounts:   len(p.pool),
		EnabledAccounts: len(p.pool),
		TotalRequests:   totalRequests,
		Accounts:        accts,
	}
}

// matchIncomingToken checks if the incoming request's Bearer token matches
// any managed account (pool or all OpenAI OAuth accounts). Returns a poolEntry
// for logging purposes, or nil if no match.
func (p *CodexProxy) matchIncomingToken(r *http.Request) *poolEntry {
	auth := r.Header.Get("Authorization")
	if len(auth) <= 7 {
		return nil
	}
	token := auth[7:] // strip "Bearer "

	// O(1) lookup via token index (covers pool entries)
	p.mu.RLock()
	if entry, ok := p.tokenIndex[token]; ok {
		p.mu.RUnlock()
		return entry
	}
	p.mu.RUnlock()

	// Fallback: check all OAuth accounts (the account may not be in the pool)
	if p.openaiDB != nil {
		account, err := p.openaiDB.GetByAccessToken(token)
		if err == nil && account != nil {
			cnt := int64(0)
			accountID := ""
			if account.ChatGPTAccountID != nil {
				accountID = *account.ChatGPTAccountID
			}
			return &poolEntry{
				id:               account.ID,
				email:            account.Email,
				accessToken:      token,
				chatgptAccountID: accountID,
				source:           "openai",
				requests:         &cnt,
			}
		}
	}

	return nil
}

func (p *CodexProxy) pickEntry() *poolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.pool) == 0 {
		return nil
	}

	switch p.strategy {
	case "random":
		idx := rand.Intn(len(p.pool))
		return &p.pool[idx]
	case "least_used":
		least := &p.pool[0]
		leastVal := int64(0)
		if least.requests != nil {
			leastVal = atomic.LoadInt64(least.requests)
		}
		for i := 1; i < len(p.pool); i++ {
			if p.pool[i].requests != nil {
				v := atomic.LoadInt64(p.pool[i].requests)
				if v < leastVal {
					least = &p.pool[i]
					leastVal = v
				}
			}
		}
		return least
	case "auto", "quota_high_first", "quota_low_first", "plan_high_first", "plan_low_first", "expiry_soon_first":
		bestIndex := 0
		for i := 1; i < len(p.pool); i++ {
			if comparePoolEntries(&p.pool[i], &p.pool[bestIndex], p.strategy) < 0 {
				bestIndex = i
			}
		}
		return &p.pool[bestIndex]
	default: // round_robin
		idx := int(atomicAddInt64(&p.currentIndex, 1)-1) % len(p.pool)
		return &p.pool[idx]
	}
}

// pickEntryExcluding returns a random pool entry not in the tried set.
// Callers should maintain tried tokens to avoid repeating invalidated accounts.
func (p *CodexProxy) pickEntryExcluding(tried map[string]bool) *poolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.pool) == 0 {
		return nil
	}
	// Build candidate indexes.
	cands := make([]int, 0, len(p.pool))
	for i := range p.pool {
		tok := p.pool[i].accessToken
		if tok == "" {
			continue
		}
		if tried != nil && tried[tok] {
			continue
		}
		cands = append(cands, i)
	}
	if len(cands) == 0 {
		return nil
	}
	idx := cands[rand.Intn(len(cands))]
	return &p.pool[idx]
}

// IsKnownToken checks if a Bearer token belongs to any managed account.
func (p *CodexProxy) IsKnownToken(token string) bool {
	if token == "" {
		return false
	}
	p.mu.RLock()
	_, found := p.tokenIndex[token]
	p.mu.RUnlock()
	if found {
		return true
	}

	if p.openaiDB != nil {
		account, err := p.openaiDB.GetByAccessToken(token)
		if err == nil && account != nil {
			return true
		}
	}
	return false
}

func (p *CodexProxy) IsEnabled() bool { return p.enabled }
func (p *CodexProxy) SetEnabled(v bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = v
}

func (p *CodexProxy) GetStrategy() string { return p.strategy }
func (p *CodexProxy) SetStrategy(s string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.strategy = s
}

func (p *CodexProxy) PoolSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.pool)
}

func (p *CodexProxy) TotalRequests() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var total int64
	for _, e := range p.pool {
		if e.requests != nil {
			total += atomic.LoadInt64(e.requests)
		}
	}
	return total
}

func loadLocalAccessAccountIDSet() map[string]bool {
	raw, ok := storage.GetSetting("codex_local_access_account_ids")
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			set[id] = true
		}
	}
	return set
}

func planRank(plan *string) int {
	value := ""
	if plan != nil {
		value = strings.ToLower(strings.TrimSpace(*plan))
	}
	switch value {
	case "enterprise":
		return 6
	case "business":
		return 5
	case "team":
		return 4
	case "pro":
		return 3
	case "plus":
		return 2
	case "free":
		return 1
	default:
		return 0
	}
}

func remainingQuota(account *models.OpenAIAccount) *int {
	if account == nil {
		return nil
	}
	values := make([]int, 0, 2)
	if account.Quota5hUsedPercent != nil {
		values = append(values, clampPercent(100-int(*account.Quota5hUsedPercent)))
	}
	if account.Quota7dUsedPercent != nil {
		values = append(values, clampPercent(100-int(*account.Quota7dUsedPercent)))
	}
	if len(values) == 0 && account.QuotaTotal != nil && *account.QuotaTotal > 0 && account.QuotaUsed != nil {
		remaining := int((1 - float64(*account.QuotaUsed)/float64(*account.QuotaTotal)) * 100)
		values = append(values, clampPercent(remaining))
	}
	if len(values) == 0 {
		return nil
	}
	best := values[0]
	for _, value := range values[1:] {
		if value < best {
			best = value
		}
	}
	return &best
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func comparePoolEntries(left, right *poolEntry, strategy string) int {
	switch strategy {
	case "quota_high_first":
		if diff := compareOptionalIntDesc(left.remainingQuota, right.remainingQuota); diff != 0 {
			return diff
		}
		if diff := compareIntDesc(left.planRank, right.planRank); diff != 0 {
			return diff
		}
	case "quota_low_first":
		if diff := compareOptionalIntAsc(left.remainingQuota, right.remainingQuota); diff != 0 {
			return diff
		}
		if diff := compareIntDesc(left.planRank, right.planRank); diff != 0 {
			return diff
		}
	case "plan_high_first":
		if diff := compareIntDesc(left.planRank, right.planRank); diff != 0 {
			return diff
		}
		if diff := compareOptionalIntDesc(left.remainingQuota, right.remainingQuota); diff != 0 {
			return diff
		}
	case "plan_low_first":
		if diff := compareIntAsc(left.planRank, right.planRank); diff != 0 {
			return diff
		}
		if diff := compareOptionalIntDesc(left.remainingQuota, right.remainingQuota); diff != 0 {
			return diff
		}
	case "expiry_soon_first":
		if diff := compareOptionalTimeAsc(left.expiresAt, right.expiresAt); diff != 0 {
			return diff
		}
		if diff := compareIntDesc(left.planRank, right.planRank); diff != 0 {
			return diff
		}
		if diff := compareOptionalIntDesc(left.remainingQuota, right.remainingQuota); diff != 0 {
			return diff
		}
	default: // auto
		if diff := compareIntDesc(left.planRank, right.planRank); diff != 0 {
			return diff
		}
		if diff := compareOptionalIntDesc(left.remainingQuota, right.remainingQuota); diff != 0 {
			return diff
		}
	}
	return strings.Compare(left.id, right.id)
}

func compareIntDesc(left, right int) int {
	if left == right {
		return 0
	}
	if left > right {
		return -1
	}
	return 1
}

func compareIntAsc(left, right int) int {
	if left == right {
		return 0
	}
	if left < right {
		return -1
	}
	return 1
}

func compareOptionalIntDesc(left, right *int) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	return compareIntDesc(*left, *right)
}

func compareOptionalIntAsc(left, right *int) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	return compareIntAsc(*left, *right)
}

func compareOptionalTimeAsc(left, right *time.Time) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	if left.Equal(*right) {
		return 0
	}
	if left.Before(*right) {
		return -1
	}
	return 1
}

// saveRateLimits captures rate-limit headers from upstream and persists them to the OpenAI account.
func (p *CodexProxy) saveRateLimits(entry *poolEntry, resp *http.Response) {
	if entry.source != "openai" || p.openaiDB == nil {
		return
	}
	info := openaiplatform.ParseCodexHeaders(resp.Header)
	if info == nil {
		return
	}
	acc, err := p.openaiDB.Get(entry.id)
	if err != nil || acc == nil {
		return
	}
	acc.QuotaTotal = &info.Total
	used := info.Used
	acc.QuotaUsed = &used
	if info.ResetAt != "" {
		acc.QuotaResetAt = &info.ResetAt
	}
	acc.Quota5hUsedPercent = info.Codex5hUsedPercent
	acc.Quota5hResetSeconds = info.Codex5hResetSeconds
	acc.Quota5hWindowMinutes = info.Codex5hWindowMinutes
	acc.Quota7dUsedPercent = info.Codex7dUsedPercent
	acc.Quota7dResetSeconds = info.Codex7dResetSeconds
	acc.Quota7dWindowMinutes = info.Codex7dWindowMinutes
	now := time.Now()
	acc.QuotaUpdatedAt = &now
	_ = p.openaiDB.Save(acc)
}

// saveLog is intentionally a no-op. EasyLLM does not retain API call logs.
func (p *CodexProxy) saveLog(entry *poolEntry, requestBody []byte, requestPath, lastSSEData string, statusCode int, durationMs int64, userAgent string) {
}

// atomicAddInt64 is a helper that wraps atomic.AddInt64 for cleaner call sites.
func atomicAddInt64(addr *int64, delta int64) int64 {
	return atomic.AddInt64(addr, delta)
}
