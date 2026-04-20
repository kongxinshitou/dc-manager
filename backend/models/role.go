package models

// 预定义权限码常量
const (
	PermDeviceRead       = "device:read"
	PermDeviceWrite      = "device:write"
	PermDeviceDelete     = "device:delete"
	PermDeviceImport     = "device:import"
	PermInspectionRead   = "inspection:read"
	PermInspectionWrite  = "inspection:write"
	PermInspectionDelete = "inspection:delete"
	PermInspectionImport = "inspection:import"
	PermDashboard        = "dashboard:view"
	PermUserManage       = "user:manage"
	PermRoleManage       = "role:manage"
	PermImageUpload      = "image:upload"
	PermImageDelete      = "image:delete"
	// 新增权限
	PermApprovalSubmit  = "approval:submit"
	PermApprovalApprove = "approval:approve"
	PermApprovalExecute = "approval:execute"
	PermApprovalView    = "approval:view"
	PermDatacenterManage = "datacenter:manage"
	PermDatacenterView   = "datacenter:view"
	PermConfigManage     = "config:manage"
)

// AllPermissions 返回所有权限码列表，用于 admin 角色和前端展示
var AllPermissions = []string{
	PermDeviceRead,
	PermDeviceWrite,
	PermDeviceDelete,
	PermDeviceImport,
	PermInspectionRead,
	PermInspectionWrite,
	PermInspectionDelete,
	PermInspectionImport,
	PermDashboard,
	PermUserManage,
	PermRoleManage,
	PermImageUpload,
	PermImageDelete,
	PermApprovalSubmit,
	PermApprovalApprove,
	PermApprovalExecute,
	PermApprovalView,
	PermDatacenterManage,
	PermDatacenterView,
	PermConfigManage,
}

// PermissionGroups 权限按功能分组，用于前端展示
var PermissionGroups = []struct {
	Label       string
	Permissions []string
}{
	{"设备管理", []string{PermDeviceRead, PermDeviceWrite, PermDeviceDelete, PermDeviceImport}},
	{"巡检管理", []string{PermInspectionRead, PermInspectionWrite, PermInspectionDelete, PermInspectionImport, PermImageUpload, PermImageDelete}},
	{"大屏", []string{PermDashboard}},
	{"系统管理", []string{PermUserManage, PermRoleManage}},
	{"审批管理", []string{PermApprovalSubmit, PermApprovalApprove, PermApprovalExecute, PermApprovalView}},
	{"机房管理", []string{PermDatacenterManage, PermDatacenterView}},
	{"系统配置", []string{PermConfigManage}},
}

// PermissionLabels 权限码的中文标签
var PermissionLabels = map[string]string{
	PermDeviceRead:       "查看设备",
	PermDeviceWrite:      "创建/编辑设备",
	PermDeviceDelete:     "删除设备",
	PermDeviceImport:     "导入设备",
	PermInspectionRead:   "查看巡检",
	PermInspectionWrite:  "创建/编辑巡检",
	PermInspectionDelete: "删除巡检",
	PermInspectionImport: "导入巡检",
	PermDashboard:        "查看大屏",
	PermUserManage:       "用户管理",
	PermRoleManage:       "角色管理",
	PermImageUpload:      "上传图片",
	PermImageDelete:      "删除图片",
	PermApprovalSubmit:   "提交审批",
	PermApprovalApprove:  "审批操作",
	PermApprovalExecute:  "执行审批",
	PermApprovalView:     "查看审批",
	PermDatacenterManage: "机房管理",
	PermDatacenterView:   "查看机房",
	PermConfigManage:     "系统配置",
}

type Role struct {
	ID          uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"uniqueIndex;size:50;not null"`
	DisplayName string `json:"display_name" gorm:"size:100"`
	Permissions string `json:"permissions" gorm:"type:text"`  // JSON 数组 '["device:read","device:write"]'
	IsSystem    bool   `json:"is_system" gorm:"default:false"`
}
