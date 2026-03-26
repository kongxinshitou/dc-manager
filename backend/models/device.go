package models

import "time"

type Device struct {
	ID              uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	Source          string     `json:"source"`           // 来源区域
	AssetNumber     string     `json:"asset_number"`     // 资产编号
	Status          string     `json:"status"`           // 状态
	Datacenter      string     `json:"datacenter"`       // 机房
	Cabinet         string     `json:"cabinet"`          // 机柜号
	UPosition       string     `json:"u_position"`       // U位置
	StartU          *int       `json:"start_u"`          // 起始U（如04-05U → 4）
	EndU            *int       `json:"end_u"`            // 结束U（如04-05U → 5）
	Brand           string     `json:"brand"`            // 设备品牌
	Model           string     `json:"model"`            // 设备型号
	DeviceType      string     `json:"device_type"`      // 设备类型
	SerialNumber    string     `json:"serial_number"`    // 序列号
	OS              string     `json:"os"`               // 操作系统
	IPAddress       string     `json:"ip_address"`       // IP地址
	SystemAccount   string     `json:"system_account"`   // 系统账号密码
	MgmtIP          string     `json:"mgmt_ip"`          // 远程管理IP
	MgmtAccount     string     `json:"mgmt_account"`     // 管理口账号
	ManufactureDate *time.Time `json:"manufacture_date"` // 设备出厂时间
	WarrantyStart   *time.Time `json:"warranty_start"`   // 维保起始时间
	WarrantyEnd     *time.Time `json:"warranty_end"`     // 维保结束时间
	Purpose         string     `json:"purpose"`          // 设备用途
	Owner           string     `json:"owner"`            // 责任人
	Remark          string     `json:"remark"`           // 备注
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type DeviceQuery struct {
	Source     string `form:"source"`
	Status     string `form:"status"`
	Datacenter string `form:"datacenter"`
	Cabinet    string `form:"cabinet"`
	Brand      string `form:"brand"`
	Model      string `form:"model"`
	DeviceType string `form:"device_type"`
	IPAddress  string `form:"ip_address"`
	Owner      string `form:"owner"`
	Keyword    string `form:"keyword"` // 全局搜索
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	OrderBy    string `form:"order_by"`
	Sort       string `form:"sort"`
}
