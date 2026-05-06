package services

import (
	"bytes"
	"dcmanager/models"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	InspectionActionAssign          = "assign"
	InspectionActionStartProcessing = "start_processing"
	InspectionActionResolve         = "resolve"
	InspectionActionReopen          = "reopen"

	ConfigInspectionWebhook    = "inspection_webhook_config"
	ConfigInspectionEscalation = "inspection_escalation_config"
)

type InspectionTransitionRequest struct {
	Action     string `json:"action" binding:"required"`
	AssigneeID *uint  `json:"assignee_id"`
	Remark     string `json:"remark"`
}

type InspectionWebhookSender interface {
	SendInspectionEvent(event models.InspectionEvent, inspection models.Inspection) error
}

type InspectionLifecycleService struct {
	DB            *gorm.DB
	Now           func() time.Time
	WebhookSender InspectionWebhookSender
}

type InspectionWebhookConfig struct {
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhook_url"`
}

type InspectionEscalationConfig struct {
	Enabled             bool           `json:"enabled"`
	ScanIntervalMinutes int            `json:"scan_interval_minutes"`
	SeverityHours       map[string]int `json:"severity_hours"`
}

func DefaultInspectionEscalationConfig() InspectionEscalationConfig {
	return InspectionEscalationConfig{
		Enabled:             true,
		ScanIntervalMinutes: 5,
		SeverityHours: map[string]int{
			"严重": 2,
			"一般": 8,
			"轻微": 24,
		},
	}
}

func LoadInspectionEscalationConfig(db *gorm.DB) InspectionEscalationConfig {
	cfg := DefaultInspectionEscalationConfig()
	var item models.SystemConfig
	if err := db.Where("`key` = ?", ConfigInspectionEscalation).First(&item).Error; err != nil {
		return cfg
	}
	_ = json.Unmarshal([]byte(item.Value), &cfg)
	if cfg.ScanIntervalMinutes <= 0 {
		cfg.ScanIntervalMinutes = 5
	}
	if cfg.SeverityHours == nil {
		cfg.SeverityHours = DefaultInspectionEscalationConfig().SeverityHours
	}
	return cfg
}

func NewConfiguredInspectionWebhookSender(db *gorm.DB) InspectionWebhookSender {
	var item models.SystemConfig
	if err := db.Where("`key` = ?", ConfigInspectionWebhook).First(&item).Error; err != nil {
		return nil
	}
	var cfg InspectionWebhookConfig
	if err := json.Unmarshal([]byte(item.Value), &cfg); err != nil || !cfg.Enabled || strings.TrimSpace(cfg.WebhookURL) == "" {
		return nil
	}
	return WeComInspectionWebhookSender{
		URL:    strings.TrimSpace(cfg.WebhookURL),
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s InspectionLifecycleService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s InspectionLifecycleService) Transition(id uint, req InspectionTransitionRequest, operatorID uint) (*models.Inspection, error) {
	if s.DB == nil {
		return nil, errors.New("database is required")
	}
	var event models.InspectionEvent
	var updated models.Inspection
	now := s.now()

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		var inspection models.Inspection
		if err := tx.First(&inspection, id).Error; err != nil {
			return err
		}
		fromStatus := inspection.Status
		eventType := ""

		switch req.Action {
		case InspectionActionAssign:
			if req.AssigneeID == nil {
				return errors.New("assignee_id is required")
			}
			user, err := ResolveInspectionAssignee(tx, *req.AssigneeID)
			if err != nil {
				return err
			}
			inspection.AssigneeID = &user.ID
			inspection.AssigneeName = user.DisplayName
			if inspection.AssigneeName == "" {
				inspection.AssigneeName = user.Username
			}
			eventType = models.InspectionEventAssigned

		case InspectionActionStartProcessing:
			if inspection.Status == models.InspectionStatusResolved {
				return errors.New("已解决巡检不能开始处理")
			}
			inspection.Status = models.InspectionStatusProcessing
			inspection.LastRespondedAt = &now
			eventType = models.InspectionEventStarted

		case InspectionActionResolve:
			inspection.Status = models.InspectionStatusResolved
			inspection.ResolvedAt = &now
			inspection.LastRespondedAt = &now
			eventType = models.InspectionEventResolved

		case InspectionActionReopen:
			if inspection.Status != models.InspectionStatusResolved {
				return errors.New("只有已解决巡检可以重开")
			}
			inspection.Status = models.InspectionStatusOpen
			inspection.ResolvedAt = nil
			eventType = models.InspectionEventReopened

		default:
			return fmt.Errorf("unsupported inspection action: %s", req.Action)
		}

		if err := tx.Save(&inspection).Error; err != nil {
			return err
		}
		event = buildInspectionEvent(inspection, eventType, fromStatus, inspection.Status, operatorID, req.Remark, now)
		if shouldSendInspectionWebhook(event.EventType, inspection) && s.WebhookSender != nil {
			event.WebhookStatus = models.InspectionWebhookSent
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		updated = inspection
		return nil
	}); err != nil {
		return nil, err
	}

	s.dispatchWebhook(&event, updated)
	return &updated, nil
}

func (s InspectionLifecycleService) RecordCreated(inspection models.Inspection, operatorID uint) models.InspectionEvent {
	event := buildInspectionEvent(inspection, models.InspectionEventCreated, "", inspection.Status, operatorID, "", s.now())
	if shouldSendInspectionWebhook(event.EventType, inspection) && s.WebhookSender != nil {
		event.WebhookStatus = models.InspectionWebhookSent
	}
	if err := s.DB.Create(&event).Error; err != nil {
		log.Printf("failed to record inspection created event: %v", err)
		return event
	}
	s.dispatchWebhook(&event, inspection)
	return event
}

func (s InspectionLifecycleService) ScanOverdue(cfg InspectionEscalationConfig) (int, error) {
	if s.DB == nil {
		return 0, errors.New("database is required")
	}
	if !cfg.Enabled {
		return 0, nil
	}
	if cfg.SeverityHours == nil {
		cfg.SeverityHours = DefaultInspectionEscalationConfig().SeverityHours
	}
	now := s.now()
	var inspections []models.Inspection
	if err := s.DB.Where("status != ?", models.InspectionStatusResolved).Find(&inspections).Error; err != nil {
		return 0, err
	}

	escalated := 0
	for _, inspection := range inspections {
		hours := cfg.SeverityHours[inspection.Severity]
		if hours <= 0 {
			continue
		}
		baseline := inspection.FoundAt
		if inspection.LastRespondedAt != nil {
			continue
		}
		if inspection.LastEscalatedAt != nil {
			baseline = *inspection.LastEscalatedAt
		}
		if now.Sub(baseline) < time.Duration(hours)*time.Hour {
			continue
		}

		inspection.EscalationLevel++
		inspection.LastEscalatedAt = &now
		event := buildInspectionEvent(inspection, models.InspectionEventEscalated, inspection.Status, inspection.Status, 0,
			fmt.Sprintf("未响应超过%d小时，自动升级", hours), now)
		if s.WebhookSender != nil {
			event.WebhookStatus = models.InspectionWebhookSent
		}
		if err := s.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&inspection).Error; err != nil {
				return err
			}
			return tx.Create(&event).Error
		}); err != nil {
			return escalated, err
		}
		s.dispatchWebhook(&event, inspection)
		escalated++
	}
	return escalated, nil
}

func ResolveInspectionAssignee(db *gorm.DB, id uint) (models.User, error) {
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		return user, errors.New("责任人不存在")
	}
	if user.Status != "active" {
		return user, errors.New("责任人账号已禁用")
	}
	return user, nil
}

func FindInspectionAssigneeByName(db *gorm.DB, name string) (*models.User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	var user models.User
	if err := db.Where("status = ? AND (username = ? OR display_name = ?)", "active", name, name).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func ApplyInspectionAssignee(db *gorm.DB, inspection *models.Inspection) error {
	if inspection.AssigneeID == nil {
		inspection.AssigneeName = ""
		return nil
	}
	user, err := ResolveInspectionAssignee(db, *inspection.AssigneeID)
	if err != nil {
		return err
	}
	inspection.AssigneeName = user.DisplayName
	if inspection.AssigneeName == "" {
		inspection.AssigneeName = user.Username
	}
	return nil
}

func buildInspectionEvent(inspection models.Inspection, eventType, fromStatus, toStatus string, operatorID uint, remark string, at time.Time) models.InspectionEvent {
	status := models.InspectionWebhookSkipped
	return models.InspectionEvent{
		InspectionID:    inspection.ID,
		EventType:       eventType,
		FromStatus:      fromStatus,
		ToStatus:        toStatus,
		OperatorID:      operatorID,
		AssigneeID:      inspection.AssigneeID,
		AssigneeName:    inspection.AssigneeName,
		EscalationLevel: inspection.EscalationLevel,
		Remark:          remark,
		WebhookStatus:   status,
		CreatedAt:       at,
	}
}

func shouldSendInspectionWebhook(eventType string, inspection models.Inspection) bool {
	switch eventType {
	case models.InspectionEventCreated:
		return inspection.Severity == "严重"
	case models.InspectionEventEscalated, models.InspectionEventResolved:
		return true
	default:
		return false
	}
}

func (s InspectionLifecycleService) dispatchWebhook(event *models.InspectionEvent, inspection models.Inspection) {
	if event.ID == 0 || event.WebhookStatus == models.InspectionWebhookSkipped || s.WebhookSender == nil {
		return
	}
	if err := s.WebhookSender.SendInspectionEvent(*event, inspection); err != nil {
		event.WebhookStatus = models.InspectionWebhookFailed
		event.WebhookError = err.Error()
		_ = s.DB.Model(&models.InspectionEvent{}).Where("id = ?", event.ID).
			Updates(map[string]any{"webhook_status": event.WebhookStatus, "webhook_error": event.WebhookError}).Error
		return
	}
	_ = s.DB.Model(&models.InspectionEvent{}).Where("id = ?", event.ID).
		Update("webhook_status", models.InspectionWebhookSent).Error
}

type WeComInspectionWebhookSender struct {
	URL    string
	Client *http.Client
}

func (s WeComInspectionWebhookSender) SendInspectionEvent(event models.InspectionEvent, inspection models.Inspection) error {
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	content := formatInspectionMarkdown(event, inspection)
	body, _ := json.Marshal(map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	})
	resp, err := client.Post(s.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("企业微信 webhook 返回 HTTP %d", resp.StatusCode)
	}
	return nil
}

func formatInspectionMarkdown(event models.InspectionEvent, inspection models.Inspection) string {
	title := "巡检事件"
	switch event.EventType {
	case models.InspectionEventCreated:
		title = "严重巡检问题"
	case models.InspectionEventEscalated:
		title = "巡检超时升级"
	case models.InspectionEventResolved:
		title = "巡检问题已解决"
	}
	location := strings.Trim(strings.Join([]string{inspection.Datacenter, inspection.Cabinet, inspection.UPosition}, " / "), " /")
	if location == "" {
		location = "-"
	}
	return fmt.Sprintf("**%s**\n>记录ID：%d\n>状态：%s\n>等级：%s\n>责任人：%s\n>位置：%s\n>问题：%s\n>备注：%s",
		title, inspection.ID, inspection.Status, inspection.Severity, emptyDash(inspection.AssigneeName), location, emptyDash(inspection.Issue), emptyDash(event.Remark))
}

func emptyDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}

func StartInspectionEscalationWorker(db *gorm.DB) {
	cfg := LoadInspectionEscalationConfig(db)
	if !cfg.Enabled {
		return
	}
	interval := time.Duration(cfg.ScanIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			cfg := LoadInspectionEscalationConfig(db)
			svc := InspectionLifecycleService{
				DB:            db,
				WebhookSender: NewConfiguredInspectionWebhookSender(db),
			}
			if count, err := svc.ScanOverdue(cfg); err != nil {
				log.Printf("inspection escalation scan failed: %v", err)
			} else if count > 0 {
				log.Printf("inspection escalation scan escalated %d records", count)
			}
		}
	}()
}
