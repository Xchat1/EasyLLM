package handlers

import (
	"easyllm/internal/models"
	"easyllm/internal/storage"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const cockpitGeneralSettingsKey = "cockpit_general_settings"

type CockpitHandler struct {
	storage    *storage.CockpitStorage
	openai     *storage.OpenAIStorage
	codexStore *storage.CodexStorage
}

func NewCockpitHandler(s *storage.CockpitStorage, openai *storage.OpenAIStorage, codexStore *storage.CodexStorage) *CockpitHandler {
	return &CockpitHandler{
		storage:    s,
		openai:     openai,
		codexStore: codexStore,
	}
}

func (h *CockpitHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/cockpit")
	g.GET("/accounts", h.ListAllAccounts)
	g.GET("/definitions", h.Definitions)
	g.GET("/overview", h.Overview)
	g.GET("/instances", h.ListAllInstances)
	g.GET("/wakeup/tasks", h.ListWakeupTasks)
	g.POST("/wakeup/tasks", h.CreateWakeupTask)
	g.PUT("/wakeup/tasks/:id", h.UpdateWakeupTask)
	g.DELETE("/wakeup/tasks/:id", h.DeleteWakeupTask)
	g.POST("/wakeup/tasks/:id/toggle", h.ToggleWakeupTask)
	g.GET("/settings/general", h.GetGeneralSettings)
	g.PUT("/settings/general", h.UpdateGeneralSettings)

	platform := g.Group("/platforms/:platform")
	platform.GET("/accounts", h.ListAccounts)
	platform.POST("/accounts/import", h.ImportAccounts)
	platform.GET("/accounts/export", h.ExportAccounts)
	platform.POST("/accounts", h.CreateAccount)
	platform.PUT("/accounts/:id", h.UpdateAccount)
	platform.DELETE("/accounts/:id", h.DeleteAccount)
	platform.DELETE("/accounts", h.DeleteManyAccounts)
	platform.POST("/accounts/:id/activate", h.ActivateAccount)

	platform.GET("/instances", h.ListInstances)
	platform.GET("/instances/export", h.ExportInstances)
	platform.POST("/instances", h.CreateInstance)
	platform.PUT("/instances/:id", h.UpdateInstance)
	platform.DELETE("/instances/:id", h.DeleteInstance)
	platform.POST("/instances/:id/state", h.UpdateInstanceState)
}

func (h *CockpitHandler) Definitions(c *gin.Context) {
	c.JSON(http.StatusOK, models.GetCockpitPlatformDefinitions())
}

func (h *CockpitHandler) ListAllAccounts(c *gin.Context) {
	accounts, err := h.storage.ListAccounts("")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	if h.openai != nil {
		codexAccounts, err := h.openai.List()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
			return
		}
		for _, account := range codexAccounts {
			mapped := mapCodexAccount(account)
			accounts = append(accounts, mapped)
		}
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *CockpitHandler) Overview(c *gin.Context) {
	overview, err := h.buildOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "COCKPIT_OVERVIEW_ERROR"})
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *CockpitHandler) ListAccounts(c *gin.Context) {
	platform, ok := resolveCockpitAccountPlatform(c)
	if !ok {
		return
	}
	accounts, err := h.storage.ListAccounts(platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *CockpitHandler) ImportAccounts(c *gin.Context) {
	platform, ok := resolveCockpitAccountPlatform(c)
	if !ok {
		return
	}
	var accounts []models.PlatformAccount
	if err := c.ShouldBindJSON(&accounts); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	imported := 0
	for _, account := range accounts {
		if account.Email == "" {
			continue
		}
		if account.ID == "" {
			account.ID = uuid.New().String()
		}
		account.Platform = platform
		if account.Status == "" {
			account.Status = "active"
		}
		if account.CreatedAt.IsZero() {
			account.CreatedAt = time.Now()
		}
		account.UpdatedAt = time.Now()
		if err := h.storage.SaveAccount(&account); err == nil {
			imported++
			if account.Active {
				_ = h.storage.SetActiveAccount(platform, account.ID)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "imported": imported})
}

func (h *CockpitHandler) ExportAccounts(c *gin.Context) {
	platform, ok := resolveCockpitAccountPlatform(c)
	if !ok {
		return
	}
	accounts, err := h.storage.ListAccounts(platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *CockpitHandler) CreateAccount(c *gin.Context) {
	platform, ok := resolveCockpitAccountPlatform(c)
	if !ok {
		return
	}
	var account models.PlatformAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if account.ID == "" {
		account.ID = uuid.New().String()
	}
	account.Email = normalizeRequiredText(account.Email)
	if account.Email == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "email is required", Code: "INVALID_REQUEST"})
		return
	}
	account.Platform = platform
	if account.Status == "" {
		account.Status = "active"
	}
	if account.CreatedAt.IsZero() {
		account.CreatedAt = time.Now()
	}
	account.UpdatedAt = time.Now()
	if err := h.storage.SaveAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	if account.Active {
		if err := h.storage.SetActiveAccount(platform, account.ID); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
			return
		}
	}
	c.JSON(http.StatusOK, account)
}

func (h *CockpitHandler) UpdateAccount(c *gin.Context) {
	platform, ok := resolveCockpitAccountPlatform(c)
	if !ok {
		return
	}
	existing, err := h.storage.GetAccount(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	if existing.Platform != platform {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "platform mismatch", Code: "INVALID_REQUEST"})
		return
	}

	var account models.PlatformAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	account.Email = normalizeRequiredText(account.Email)
	if account.Email == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "email is required", Code: "INVALID_REQUEST"})
		return
	}
	account = mergePlatformAccountUpdate(existing, account)
	account.ID = existing.ID
	account.Platform = existing.Platform
	account.CreatedAt = existing.CreatedAt
	account.UpdatedAt = time.Now()
	if account.Status == "" {
		account.Status = existing.Status
		if account.Status == "" {
			account.Status = "active"
		}
	}
	if err := h.storage.SaveAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	if account.Active {
		if err := h.storage.SetActiveAccount(platform, account.ID); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
			return
		}
	}
	c.JSON(http.StatusOK, account)
}

func (h *CockpitHandler) DeleteAccount(c *gin.Context) {
	platform, ok := resolveCockpitAccountPlatform(c)
	if !ok {
		return
	}
	if err := h.storage.DeleteAccount(platform, c.Param("id")); err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CockpitHandler) DeleteManyAccounts(c *gin.Context) {
	platform, ok := resolveCockpitAccountPlatform(c)
	if !ok {
		return
	}
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if err := h.storage.DeleteManyAccounts(platform, req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CockpitHandler) ActivateAccount(c *gin.Context) {
	platform, ok := resolveCockpitAccountPlatform(c)
	if !ok {
		return
	}
	if err := h.storage.SetActiveAccount(platform, c.Param("id")); err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CockpitHandler) ListInstances(c *gin.Context) {
	platform, ok := resolveCockpitAnyPlatform(c)
	if !ok {
		return
	}
	instances, err := h.storage.ListInstances(platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, instances)
}

func (h *CockpitHandler) ExportInstances(c *gin.Context) {
	platform, ok := resolveCockpitAnyPlatform(c)
	if !ok {
		return
	}
	instances, err := h.storage.ListInstances(platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, instances)
}

func (h *CockpitHandler) ListAllInstances(c *gin.Context) {
	instances, err := h.storage.ListInstances("")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, instances)
}

func (h *CockpitHandler) CreateInstance(c *gin.Context) {
	platform, ok := resolveCockpitAnyPlatform(c)
	if !ok {
		return
	}
	var instance models.PlatformInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if instance.ID == "" {
		instance.ID = uuid.New().String()
	}
	instance.Name = normalizeRequiredText(instance.Name)
	if instance.Name == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "name is required", Code: "INVALID_REQUEST"})
		return
	}
	instance.Platform = platform
	if instance.State == "" {
		instance.State = "stopped"
	}
	instance.UpdatedAt = time.Now()
	if instance.CreatedAt.IsZero() {
		instance.CreatedAt = time.Now()
	}
	if err := h.storage.SaveInstance(&instance); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, instance)
}

func (h *CockpitHandler) UpdateInstance(c *gin.Context) {
	platform, ok := resolveCockpitAnyPlatform(c)
	if !ok {
		return
	}
	existing, err := h.storage.GetInstance(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	if existing.Platform != platform {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "platform mismatch", Code: "INVALID_REQUEST"})
		return
	}
	var instance models.PlatformInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	instance.Name = normalizeRequiredText(instance.Name)
	if instance.Name == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "name is required", Code: "INVALID_REQUEST"})
		return
	}
	instance = mergePlatformInstanceUpdate(existing, instance)
	instance.ID = existing.ID
	instance.Platform = existing.Platform
	instance.CreatedAt = existing.CreatedAt
	instance.UpdatedAt = time.Now()
	if instance.State == "" {
		instance.State = existing.State
		if instance.State == "" {
			instance.State = "stopped"
		}
	}
	if err := h.storage.SaveInstance(&instance); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, instance)
}

func (h *CockpitHandler) DeleteInstance(c *gin.Context) {
	platform, ok := resolveCockpitAnyPlatform(c)
	if !ok {
		return
	}
	if err := h.storage.DeleteInstance(platform, c.Param("id")); err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CockpitHandler) UpdateInstanceState(c *gin.Context) {
	platform, ok := resolveCockpitAnyPlatform(c)
	if !ok {
		return
	}
	var req struct {
		State string `json:"state" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	switch req.State {
	case "running", "stopped", "paused":
	default:
		c.JSON(http.StatusBadRequest, models.APIError{Error: "invalid state", Code: "INVALID_REQUEST"})
		return
	}
	if err := h.storage.SetInstanceState(platform, c.Param("id"), req.State); err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "state": req.State})
}

func (h *CockpitHandler) ListWakeupTasks(c *gin.Context) {
	platform := c.Query("platform")
	if platform != "" && !models.IsSupportedCockpitPlatform(platform) {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "unsupported platform", Code: "INVALID_PLATFORM"})
		return
	}
	tasks, err := h.storage.ListWakeupTasks(platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *CockpitHandler) CreateWakeupTask(c *gin.Context) {
	var task models.WakeupTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if !models.IsSupportedCockpitPlatform(task.Platform) {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "unsupported platform", Code: "INVALID_PLATFORM"})
		return
	}
	task.Name = normalizeRequiredText(task.Name)
	task.ScheduleValue = normalizeRequiredText(task.ScheduleValue)
	if task.Name == "" || task.ScheduleValue == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "name and schedule_value are required", Code: "INVALID_REQUEST"})
		return
	}
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	if task.ScheduleType == "" {
		task.ScheduleType = "daily"
	}
	task.UpdatedAt = time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if err := h.storage.SaveWakeupTask(&task); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *CockpitHandler) UpdateWakeupTask(c *gin.Context) {
	existing, err := h.storage.GetWakeupTask(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	var task models.WakeupTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if !models.IsSupportedCockpitPlatform(task.Platform) {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "unsupported platform", Code: "INVALID_PLATFORM"})
		return
	}
	task.Name = normalizeRequiredText(task.Name)
	task.ScheduleValue = normalizeRequiredText(task.ScheduleValue)
	if task.Name == "" || task.ScheduleValue == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "name and schedule_value are required", Code: "INVALID_REQUEST"})
		return
	}
	task = mergeWakeupTaskUpdate(existing, task)
	task.ID = existing.ID
	task.CreatedAt = existing.CreatedAt
	task.UpdatedAt = time.Now()
	if task.ScheduleType == "" {
		task.ScheduleType = existing.ScheduleType
		if task.ScheduleType == "" {
			task.ScheduleType = "daily"
		}
	}
	if err := h.storage.SaveWakeupTask(&task); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *CockpitHandler) DeleteWakeupTask(c *gin.Context) {
	if err := h.storage.DeleteWakeupTask(c.Param("id")); err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CockpitHandler) ToggleWakeupTask(c *gin.Context) {
	enabled, err := h.storage.ToggleWakeupTask(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "enabled": enabled})
}

func (h *CockpitHandler) GetGeneralSettings(c *gin.Context) {
	settings, err := loadCockpitGeneralSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "SETTINGS_ERROR"})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *CockpitHandler) UpdateGeneralSettings(c *gin.Context) {
	var settings models.CockpitGeneralSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	settings = models.NormalizeCockpitGeneralSettings(settings)
	payload, err := json.Marshal(settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "SETTINGS_ERROR"})
		return
	}
	if err := storage.SaveSetting(cockpitGeneralSettingsKey, string(payload)); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "SETTINGS_ERROR"})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *CockpitHandler) buildOverview() (models.CockpitOverviewResponse, error) {
	definitions := models.GetCockpitPlatformDefinitions()
	response := models.CockpitOverviewResponse{
		Platforms:   make([]models.PlatformOverview, 0, len(definitions)),
		GeneratedAt: time.Now(),
	}

	for _, def := range definitions {
		item := models.PlatformOverview{Definition: def}

		switch def.ID {
		case "codex":
			if h.openai != nil {
				accounts, err := h.openai.List()
				if err != nil {
					return response, err
				}
				item.Accounts = int64(len(accounts))
				active, err := h.openai.GetCodexActive()
				if err == nil && active != nil {
					item.ActiveAccounts = 1
					item.ActiveAccountEmail = &active.Email
				}
			}
		default:
			accounts, err := h.storage.CountAccounts(def.ID)
			if err != nil {
				return response, err
			}
			item.Accounts = accounts

			activeAccounts, err := h.storage.CountActiveAccounts(def.ID)
			if err != nil {
				return response, err
			}
			item.ActiveAccounts = activeAccounts

			active, err := h.storage.GetActiveAccount(def.ID)
			if err == nil && active != nil {
				item.ActiveAccountEmail = &active.Email
			}
		}

		instances, err := h.storage.CountInstances(def.ID, false)
		if err != nil {
			return response, err
		}
		item.Instances = instances

		runningInstances, err := h.storage.CountInstances(def.ID, true)
		if err != nil {
			return response, err
		}
		item.RunningInstances = runningInstances

		wakeupTasks, err := h.storage.CountWakeupTasks(def.ID, false)
		if err != nil {
			return response, err
		}
		item.WakeupTasks = wakeupTasks

		enabledWakeupTasks, err := h.storage.CountWakeupTasks(def.ID, true)
		if err != nil {
			return response, err
		}
		item.EnabledWakeupTasks = enabledWakeupTasks

		response.Platforms = append(response.Platforms, item)
		response.Summary.TotalPlatforms++
		response.Summary.TotalAccounts += item.Accounts
		response.Summary.ActiveAccounts += item.ActiveAccounts
		response.Summary.TotalInstances += item.Instances
		response.Summary.RunningInstances += item.RunningInstances
		response.Summary.TotalWakeupTasks += item.WakeupTasks
		response.Summary.EnabledWakeupTasks += item.EnabledWakeupTasks
		if item.Accounts > 0 || item.Instances > 0 || item.WakeupTasks > 0 {
			response.Summary.EnabledPlatforms++
		}
	}

	response.Proxy = h.buildProxyOverview()
	return response, nil
}

func (h *CockpitHandler) buildProxyOverview() models.CodexProxyOverview {
	overview := models.CodexProxyOverview{
		Enabled:  true,
		Strategy: "round_robin",
	}

	if v, ok := storage.GetSetting("proxy_pool_enabled"); ok && v == "false" {
		overview.Enabled = false
	}
	if v, ok := storage.GetSetting("proxy_strategy"); ok && v != "" {
		overview.Strategy = v
	}
	if h.codexStore == nil {
		return overview
	}

	accounts, err := h.codexStore.LoadAllAccounts()
	if err != nil {
		return overview
	}
	overview.Accounts = len(accounts)
	for _, account := range accounts {
		if account.Enabled {
			overview.EnabledAccounts++
		}
		overview.TotalRequests += account.RequestCount
	}
	return overview
}

func resolveCockpitAnyPlatform(c *gin.Context) (string, bool) {
	platform := c.Param("platform")
	if !models.IsSupportedCockpitPlatform(platform) {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "unsupported platform", Code: "INVALID_PLATFORM"})
		return "", false
	}
	return platform, true
}

func resolveCockpitAccountPlatform(c *gin.Context) (string, bool) {
	platform, ok := resolveCockpitAnyPlatform(c)
	if !ok {
		return "", false
	}
	if platform == "codex" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "codex uses the dedicated OpenAI / Codex management flow", Code: "LEGACY_PLATFORM"})
		return "", false
	}
	return platform, true
}

func loadCockpitGeneralSettings() (models.CockpitGeneralSettings, error) {
	defaults := models.DefaultCockpitGeneralSettings()
	raw, ok := storage.GetSetting(cockpitGeneralSettingsKey)
	if !ok || raw == "" {
		return defaults, nil
	}
	var saved models.CockpitGeneralSettings
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		return defaults, err
	}
	return models.NormalizeCockpitGeneralSettings(saved), nil
}

func mapCodexAccount(account models.OpenAIAccount) models.PlatformAccount {
	mapped := models.PlatformAccount{
		ID:           account.ID,
		Platform:     "codex",
		Email:        account.Email,
		DisplayName:  nil,
		AccessToken:  account.AccessToken,
		RefreshToken: account.RefreshToken,
		Plan:         account.Plan,
		Status:       account.Status,
		Active:       account.IsCodexActive,
		TagName:      account.TagName,
		TagColor:     account.TagColor,
		CreatedAt:    account.CreatedAt,
		UpdatedAt:    account.UpdatedAt,
	}
	if account.QuotaUsed != nil {
		used := float64(*account.QuotaUsed)
		mapped.QuotaUsed = &used
	}
	if account.QuotaTotal != nil {
		limit := float64(*account.QuotaTotal)
		mapped.QuotaLimit = &limit
		unit := "requests"
		mapped.QuotaUnit = &unit
	}
	return mapped
}

func normalizeRequiredText(value string) string {
	return strings.TrimSpace(value)
}

func mergePlatformAccountUpdate(existing *models.PlatformAccount, incoming models.PlatformAccount) models.PlatformAccount {
	incoming.CookieToken = existing.CookieToken
	incoming.MetadataJSON = existing.MetadataJSON
	incoming.QuotaResetAt = existing.QuotaResetAt
	return incoming
}

func mergePlatformInstanceUpdate(existing *models.PlatformInstance, incoming models.PlatformInstance) models.PlatformInstance {
	incoming.LastStartedAt = existing.LastStartedAt
	incoming.LastStoppedAt = existing.LastStoppedAt
	return incoming
}

func mergeWakeupTaskUpdate(existing *models.WakeupTask, incoming models.WakeupTask) models.WakeupTask {
	incoming.LastRunAt = existing.LastRunAt
	incoming.NextRunAt = existing.NextRunAt
	incoming.LastStatus = existing.LastStatus
	incoming.LastMessage = existing.LastMessage
	return incoming
}
