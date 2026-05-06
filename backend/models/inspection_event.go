package models

import "time"

const (
	InspectionEventCreated   = "created"
	InspectionEventAssigned  = "assigned"
	InspectionEventStarted   = "started"
	InspectionEventResolved  = "resolved"
	InspectionEventReopened  = "reopened"
	InspectionEventEscalated = "escalated"
	InspectionEventUpdated   = "updated"
	InspectionEventDeleted   = "deleted"
	InspectionWebhookSkipped = "skipped"
	InspectionWebhookSent    = "sent"
	InspectionWebhookFailed  = "failed"
)

type InspectionEvent struct {
	ID              uint        `json:"id" gorm:"primaryKey;autoIncrement"`
	InspectionID    uint        `json:"inspection_id" gorm:"index;not null"`
	Inspection      *Inspection `json:"-" gorm:"foreignKey:InspectionID"`
	EventType       string      `json:"event_type" gorm:"size:50;index;not null"`
	FromStatus      string      `json:"from_status" gorm:"size:50"`
	ToStatus        string      `json:"to_status" gorm:"size:50"`
	OperatorID      uint        `json:"operator_id" gorm:"index"`
	AssigneeID      *uint       `json:"assignee_id" gorm:"index"`
	AssigneeName    string      `json:"assignee_name" gorm:"size:100"`
	EscalationLevel int         `json:"escalation_level"`
	Remark          string      `json:"remark"`
	WebhookStatus   string      `json:"webhook_status" gorm:"size:20;default:'skipped'"`
	WebhookError    string      `json:"webhook_error" gorm:"type:text"`
	CreatedAt       time.Time   `json:"created_at"`
}
