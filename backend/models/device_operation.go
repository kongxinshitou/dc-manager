package models

import "time"

type DeviceOperation struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	DeviceID      uint      `json:"device_id" gorm:"index;not null"`
	OperationType string    `json:"operation_type" gorm:"size:50;not null"`
	FromStatus    string    `json:"from_status"`
	ToStatus      string    `json:"to_status"`
	OperatorID    uint      `json:"operator_id"`
	ApprovalID    *uint     `json:"approval_id"`
	Details       string    `json:"details" gorm:"type:text"`
	Remark        string    `json:"remark"`
	CreatedAt     time.Time `json:"created_at"`
}
