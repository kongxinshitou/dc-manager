package models

import "time"

type InspectionImage struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	InspectionID uint      `json:"inspection_id" gorm:"not null;index"`
	Inspection   *Inspection `json:"-" gorm:"foreignKey:InspectionID"`
	FilePath     string    `json:"file_path" gorm:"not null"`
	FileName     string    `json:"file_name" gorm:"size:255;not null"`
	FileSize     int64     `json:"file_size"`
	ContentType  string    `json:"content_type" gorm:"size:100"`
	UploadedAt   time.Time `json:"uploaded_at"`
}
