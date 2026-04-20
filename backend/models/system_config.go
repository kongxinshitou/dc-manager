package models

type SystemConfig struct {
	Key   string `json:"key" gorm:"primaryKey;size:100"`
	Value string `json:"value" gorm:"type:text"`
}
