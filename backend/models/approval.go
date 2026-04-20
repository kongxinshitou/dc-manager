package models

import "time"

type Approval struct {
	ID            uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	ApprovalNo    string     `json:"approval_no" gorm:"uniqueIndex;size:50"`
	DeviceID      uint       `json:"device_id" gorm:"index;not null"`
	OperationType string     `json:"operation_type" gorm:"size:50"`
	RequestData   string     `json:"request_data" gorm:"type:text"`
	ApplicantID   uint       `json:"applicant_id"`
	ApplicantName string     `json:"applicant_name" gorm:"size:100"`
	ApproverID    *uint      `json:"approver_id"`
	ApproverName  string     `json:"approver_name" gorm:"size:100"`
	Status        string     `json:"status" gorm:"size:20;default:'pending'"`
	ApproveRemark string     `json:"approve_remark"`
	ApprovedAt    *time.Time `json:"approved_at"`
	ExecutedAt    *time.Time `json:"executed_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ApprovalQuery struct {
	Status        string `form:"status"`
	OperationType string `form:"operation_type"`
	ApplicantID   uint   `form:"applicant_id"`
	Tab           string `form:"tab"` // pending / my_requests / all
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
}
