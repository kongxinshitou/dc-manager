package models

import "time"

type Device struct {
	ID              uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	Source          string     `json:"source"`           // 来源区域
	AssetNumber     string     `json:"asset_number"`     // 资产编号
	Status          string     `json:"status"`           // 状态(旧字段，保留兼容)
	Datacenter      string     `json:"datacenter"`       // 机房(旧字段，保留兼容)
	Cabinet         string     `json:"cabinet"`          // 机柜号(旧字段，保留兼容)
	UPosition       string     `json:"u_position"`       // U位置
	StartU          *int       `json:"start_u"`          // 起始U
	EndU            *int       `json:"end_u"`            // 结束U
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

	// === 新增字段 ===
	Vendor          string     `json:"vendor"`           // 厂商
	ArrivalDate     *time.Time `json:"arrival_date"`     // 到货日期
	WarrantyYears   int        `json:"warranty_years" gorm:"default:0"` // 原厂维保年限
	ContractNo      string     `json:"contract_no"`      // 合同号
	FinanceNo       string     `json:"finance_no"`       // 财务编号
	DeviceStatus    string     `json:"device_status" gorm:"default:'in_stock'"` // 主状态: in_stock / out_stock
	SubStatus       string     `json:"sub_status" gorm:"default:'new_purchase'"` // 子状态: new_purchase/recycled/racked/dispatched/scrapped
	StorageLocation string     `json:"storage_location"` // 存放位置
	Custodian       string     `json:"custodian"`        // 保管员
	ScrapRemark     string     `json:"scrap_remark"`     // 报废备注
	DispatchAddress string     `json:"dispatch_address"` // 外发地址
	DispatchCustodian string   `json:"dispatch_custodian"` // 外发保管人
	Applicant       string     `json:"applicant"`        // 申请人
	ProjectName     string     `json:"project_name"`     // 项目名称
	BusinessUnit    string     `json:"business_unit"`    // 所属业务
	Department      string     `json:"department"`       // 所属部门
	UCount          *int       `json:"u_count"`          // 占用几U
	BusinessAddress string     `json:"business_address"` // 业务地址
	VipAddress      string     `json:"vip_address"`      // VIP地址
	DatacenterID    *uint      `json:"datacenter_id" gorm:"index"` // 机房ID(FK)
	CabinetID       *uint      `json:"cabinet_id" gorm:"index"`    // 机柜ID(FK)

	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type DeviceQuery struct {
	Source       string `form:"source"`
	Status       string `form:"status"`
	DeviceStatus string `form:"device_status"`
	SubStatus    string `form:"sub_status"`
	Datacenter   string `form:"datacenter"`
	Cabinet      string `form:"cabinet"`
	Brand        string `form:"brand"`
	Model        string `form:"model"`
	DeviceType   string `form:"device_type"`
	IPAddress    string `form:"ip_address"`
	MgmtIP       string `form:"mgmt_ip"`
	Owner        string `form:"owner"`
	Vendor       string `form:"vendor"`
	ContractNo   string `form:"contract_no"`
	FinanceNo    string `form:"finance_no"`
	Custodian    string `form:"custodian"`
	Keyword      string `form:"keyword"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
	OrderBy      string `form:"order_by"`
	Sort         string `form:"sort"`
}
