package models

import "time"

type Inspection struct {
	ID         uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	DeviceID   *uint      `json:"device_id"`   // 关联设备（可为空）
	Device     *Device    `json:"device" gorm:"foreignKey:DeviceID"`
	Datacenter string     `json:"datacenter"`  // 机房
	Cabinet    string     `json:"cabinet"`     // 机柜
	UPosition  string     `json:"u_position"`  // U位（冗余保留）
	StartU     *int       `json:"start_u"`     // 起始U
	EndU       *int       `json:"end_u"`       // 结束U
	FoundAt    time.Time  `json:"found_at"`    // 发现问题时间
	Inspector  string     `json:"inspector"`   // 巡检人
	Issue      string     `json:"issue"`       // 问题描述
	Severity   string     `json:"severity"`    // 等级：严重/一般/轻微
	Status     string     `json:"status"`      // 状态：待处理/处理中/已解决
	ResolvedAt *time.Time `json:"resolved_at"` // 解决时间
	Remark     string     `json:"remark"`      // 备注
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type InspectionQuery struct {
	Datacenter string `form:"datacenter"`
	Cabinet    string `form:"cabinet"`
	Inspector  string `form:"inspector"`
	Severity   string `form:"severity"`
	Status     string `form:"status"`
	StartTime  string `form:"start_time"`
	EndTime    string `form:"end_time"`
	Keyword    string `form:"keyword"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	OrderBy    string `form:"order_by"`
	Sort       string `form:"sort"`
}
