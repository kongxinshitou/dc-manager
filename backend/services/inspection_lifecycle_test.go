package services

import (
	"errors"
	"testing"
	"time"

	"dcmanager/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type recordingInspectionWebhook struct {
	events []models.InspectionEvent
	err    error
}

func (r *recordingInspectionWebhook) SendInspectionEvent(event models.InspectionEvent, inspection models.Inspection) error {
	r.events = append(r.events, event)
	return r.err
}

func setupInspectionLifecycleDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Role{}, &models.User{}, &models.Device{}, &models.Inspection{}, &models.InspectionEvent{}, &models.SystemConfig{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func createLifecycleUser(t *testing.T, db *gorm.DB, username, displayName, status string) models.User {
	t.Helper()
	user := models.User{Username: username, DisplayName: displayName, PasswordHash: "x", RoleID: 1, Status: status}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func TestTransitionInspectionAssignsActiveUserAndWritesEvent(t *testing.T) {
	db := setupInspectionLifecycleDB(t)
	assignee := createLifecycleUser(t, db, "ops1", "Ops One", "active")
	operator := createLifecycleUser(t, db, "lead", "Lead", "active")
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	inspection := models.Inspection{
		FoundAt:   now.Add(-time.Hour),
		Inspector: "巡检员",
		Issue:     "温度异常",
		Severity:  "一般",
		Status:    models.InspectionStatusOpen,
	}
	if err := db.Create(&inspection).Error; err != nil {
		t.Fatalf("create inspection: %v", err)
	}

	service := InspectionLifecycleService{DB: db, Now: func() time.Time { return now }}
	updated, err := service.Transition(inspection.ID, InspectionTransitionRequest{
		Action:     InspectionActionAssign,
		AssigneeID: &assignee.ID,
		Remark:     "交给值班人员",
	}, operator.ID)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}

	if updated.AssigneeID == nil || *updated.AssigneeID != assignee.ID {
		t.Fatalf("assignee id = %v, want %d", updated.AssigneeID, assignee.ID)
	}
	if updated.AssigneeName != "Ops One" {
		t.Fatalf("assignee name = %q, want Ops One", updated.AssigneeName)
	}

	var events []models.InspectionEvent
	if err := db.Order("id").Find(&events).Error; err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if events[0].EventType != models.InspectionEventAssigned {
		t.Fatalf("event type = %q, want %q", events[0].EventType, models.InspectionEventAssigned)
	}
	if events[0].OperatorID != operator.ID || events[0].AssigneeID == nil || *events[0].AssigneeID != assignee.ID {
		t.Fatalf("event operator/assignee not recorded: %+v", events[0])
	}
}

func TestTransitionInspectionRejectsDisabledAssignee(t *testing.T) {
	db := setupInspectionLifecycleDB(t)
	disabled := createLifecycleUser(t, db, "disabled", "Disabled", "disabled")
	inspection := models.Inspection{
		FoundAt:   time.Now(),
		Inspector: "巡检员",
		Issue:     "电源告警",
		Severity:  "严重",
		Status:    models.InspectionStatusOpen,
	}
	if err := db.Create(&inspection).Error; err != nil {
		t.Fatalf("create inspection: %v", err)
	}

	service := InspectionLifecycleService{DB: db}
	_, err := service.Transition(inspection.ID, InspectionTransitionRequest{
		Action:     InspectionActionAssign,
		AssigneeID: &disabled.ID,
	}, 0)
	if err == nil {
		t.Fatal("transition succeeded, want disabled user error")
	}
}

func TestResolveTransitionSendsWebhookButDoesNotFailWhenWebhookFails(t *testing.T) {
	db := setupInspectionLifecycleDB(t)
	now := time.Date(2026, 5, 6, 11, 0, 0, 0, time.UTC)
	inspection := models.Inspection{
		FoundAt: now.Add(-2 * time.Hour), Inspector: "巡检员", Issue: "链路恢复",
		Severity: "严重", Status: models.InspectionStatusProcessing,
	}
	if err := db.Create(&inspection).Error; err != nil {
		t.Fatalf("create inspection: %v", err)
	}
	webhook := &recordingInspectionWebhook{err: errors.New("webhook down")}

	service := InspectionLifecycleService{DB: db, Now: func() time.Time { return now }, WebhookSender: webhook}
	updated, err := service.Transition(inspection.ID, InspectionTransitionRequest{
		Action: InspectionActionResolve,
		Remark: "已处理",
	}, 0)
	if err != nil {
		t.Fatalf("transition failed because webhook failed: %v", err)
	}
	if updated.Status != models.InspectionStatusResolved {
		t.Fatalf("status = %q, want %q", updated.Status, models.InspectionStatusResolved)
	}
	if updated.ResolvedAt == nil || !updated.ResolvedAt.Equal(now) {
		t.Fatalf("resolved_at = %v, want %v", updated.ResolvedAt, now)
	}
	if len(webhook.events) != 1 {
		t.Fatalf("webhook events = %d, want 1", len(webhook.events))
	}
	var event models.InspectionEvent
	if err := db.First(&event).Error; err != nil {
		t.Fatalf("load event: %v", err)
	}
	if event.WebhookStatus != models.InspectionWebhookFailed {
		t.Fatalf("webhook status = %q, want failed", event.WebhookStatus)
	}
}

func TestScanOverdueInspectionsEscalatesOnlyUnrespondedRecords(t *testing.T) {
	db := setupInspectionLifecycleDB(t)
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	records := []models.Inspection{
		{FoundAt: now.Add(-3 * time.Hour), Inspector: "a", Issue: "严重超时", Severity: "严重", Status: models.InspectionStatusOpen},
		{FoundAt: now.Add(-30 * time.Hour), Inspector: "b", Issue: "轻微但已响应", Severity: "轻微", Status: models.InspectionStatusProcessing, LastRespondedAt: ptrTime(now.Add(-time.Hour))},
		{FoundAt: now.Add(-30 * time.Hour), Inspector: "c", Issue: "已解决", Severity: "严重", Status: models.InspectionStatusResolved},
	}
	for i := range records {
		if err := db.Create(&records[i]).Error; err != nil {
			t.Fatalf("create inspection %d: %v", i, err)
		}
	}
	webhook := &recordingInspectionWebhook{}

	service := InspectionLifecycleService{DB: db, Now: func() time.Time { return now }, WebhookSender: webhook}
	count, err := service.ScanOverdue(InspectionEscalationConfig{
		Enabled: true,
		SeverityHours: map[string]int{
			"严重": 2,
			"一般": 8,
			"轻微": 24,
		},
	})
	if err != nil {
		t.Fatalf("scan overdue: %v", err)
	}
	if count != 1 {
		t.Fatalf("escalated count = %d, want 1", count)
	}

	var escalated models.Inspection
	if err := db.First(&escalated, records[0].ID).Error; err != nil {
		t.Fatalf("load escalated: %v", err)
	}
	if escalated.EscalationLevel != 1 || escalated.LastEscalatedAt == nil {
		t.Fatalf("escalation fields = level %d at %v, want level 1 with timestamp", escalated.EscalationLevel, escalated.LastEscalatedAt)
	}
	if len(webhook.events) != 1 || webhook.events[0].EventType != models.InspectionEventEscalated {
		t.Fatalf("webhook events = %+v, want one escalation", webhook.events)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
