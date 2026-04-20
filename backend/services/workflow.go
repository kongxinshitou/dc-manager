package services

import (
	"dcmanager/models"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Device status constants
const (
	StatusInStock  = "in_stock"
	StatusOutStock = "out_stock"
)

// Device sub-status constants
const (
	SubNewPurchase = "new_purchase"
	SubRecycled    = "recycled"
	SubRacked      = "racked"
	SubDispatched  = "dispatched"
	SubScrapped    = "scrapped"
	SubUnracked    = "unracked"
)

// Operation type constants
const (
	OpInStockNew     = "in_stock_new"
	OpInStockRecycle = "in_stock_recycle"
	OpRack           = "rack"
	OpDispatch       = "dispatch"
	OpScrap          = "scrap"
	OpUnrack         = "unrack"
)

// transition defines a valid state transition
type transition struct {
	FromDeviceStatus string
	FromSubStatus    string
	Operation        string
	ToDeviceStatus   string
	ToSubStatus      string
}

var transitions = []transition{
	// Initial creation
	{"", "", OpInStockNew, StatusInStock, SubNewPurchase},
	{"", "", OpInStockRecycle, StatusInStock, SubRecycled},

	// From in_stock → out_stock operations
	{StatusInStock, SubNewPurchase, OpRack, StatusOutStock, SubRacked},
	{StatusInStock, SubRecycled, OpRack, StatusOutStock, SubRacked},
	{StatusInStock, SubNewPurchase, OpDispatch, StatusOutStock, SubDispatched},
	{StatusInStock, SubRecycled, OpDispatch, StatusOutStock, SubDispatched},
	{StatusInStock, SubNewPurchase, OpScrap, StatusOutStock, SubScrapped},
	{StatusInStock, SubRecycled, OpScrap, StatusOutStock, SubScrapped},

	// Unrack: out_stock/racked → out_stock/unracked
	{StatusOutStock, SubRacked, OpUnrack, StatusOutStock, SubUnracked},

	// Confirm recycle from unracked: out_stock/unracked → in_stock/recycled
	{StatusOutStock, SubUnracked, OpInStockRecycle, StatusInStock, SubRecycled},
}

// ValidateTransition checks if a state transition is valid and returns the target state.
func ValidateTransition(deviceStatus, subStatus, operation string) (newDeviceStatus, newSubStatus string, err error) {
	for _, t := range transitions {
		if t.FromDeviceStatus == deviceStatus && t.FromSubStatus == subStatus && t.Operation == operation {
			return t.ToDeviceStatus, t.ToSubStatus, nil
		}
	}
	return "", "", fmt.Errorf("invalid transition: %s/%s → %s", deviceStatus, subStatus, operation)
}

// ExecuteTransition performs a device state transition within a transaction.
// details is a JSON map with operation-specific fields.
func ExecuteTransition(db *gorm.DB, device *models.Device, operation string, details json.RawMessage, operatorID uint, approvalID *uint) error {
	newDeviceStatus, newSubStatus, err := ValidateTransition(device.DeviceStatus, device.SubStatus, operation)
	if err != nil {
		return err
	}

	// Parse details into a map for field updates
	var detailMap map[string]any
	if len(details) > 0 {
		if err := json.Unmarshal(details, &detailMap); err != nil {
			return fmt.Errorf("invalid details JSON: %w", err)
		}
	}

	fromStatus := device.DeviceStatus + "/" + device.SubStatus

	return db.Transaction(func(tx *gorm.DB) error {
		// Update device status
		device.DeviceStatus = newDeviceStatus
		device.SubStatus = newSubStatus

		// Apply operation-specific field updates
		if err := applyOperationFields(device, operation, detailMap); err != nil {
			return err
		}

		if err := tx.Save(device).Error; err != nil {
			return fmt.Errorf("failed to save device: %w", err)
		}

		// Create operation record
		op := models.DeviceOperation{
			DeviceID:      device.ID,
			OperationType: operation,
			FromStatus:    fromStatus,
			ToStatus:      newDeviceStatus + "/" + newSubStatus,
			OperatorID:    operatorID,
			ApprovalID:    approvalID,
			Details:       string(details),
		}
		if err := tx.Create(&op).Error; err != nil {
			return fmt.Errorf("failed to create operation record: %w", err)
		}

		return nil
	})
}

// applyOperationFields updates device fields based on operation type.
func applyOperationFields(device *models.Device, operation string, details map[string]any) error {
	switch operation {
	case OpInStockNew:
		setDetailString(device, details, "storage_location", func(v string) { device.StorageLocation = v })
		setDetailString(device, details, "custodian", func(v string) { device.Custodian = v })

	case OpRack:
		// Clear previous operation-specific fields
		device.StorageLocation = ""
		device.ScrapRemark = ""
		device.DispatchAddress = ""
		device.DispatchCustodian = ""
		// Set rack-specific fields
		setDetailString(device, details, "datacenter", func(v string) { device.Datacenter = v })
		setDetailString(device, details, "cabinet", func(v string) { device.Cabinet = v })
		setDetailInt(device, details, "start_u", func(v int) { device.StartU = &v })
		setDetailInt(device, details, "u_count", func(v int) { device.UCount = &v })
		if device.StartU != nil && device.UCount != nil {
			endU := *device.StartU + *device.UCount - 1
			device.EndU = &endU
			device.UPosition = fmt.Sprintf("%d-%dU", *device.StartU, endU)
		}
		setDetailString(device, details, "applicant", func(v string) { device.Applicant = v })
		setDetailString(device, details, "project_name", func(v string) { device.ProjectName = v })
		setDetailString(device, details, "business_unit", func(v string) { device.BusinessUnit = v })
		setDetailString(device, details, "department", func(v string) { device.Department = v })
		setDetailString(device, details, "business_address", func(v string) { device.BusinessAddress = v })
		setDetailString(device, details, "vip_address", func(v string) { device.VipAddress = v })
		setDetailString(device, details, "mgmt_ip", func(v string) { device.MgmtIP = v })
		setDetailString(device, details, "os", func(v string) { device.OS = v })
		setDetailString(device, details, "custodian", func(v string) { device.Custodian = v })
		// Set FK fields
		setDetailUint(device, details, "datacenter_id", func(v uint) { device.DatacenterID = &v })
		setDetailUint(device, details, "cabinet_id", func(v uint) { device.CabinetID = &v })

	case OpDispatch:
		device.StorageLocation = ""
		device.ScrapRemark = ""
		setDetailString(device, details, "dispatch_address", func(v string) { device.DispatchAddress = v })
		setDetailString(device, details, "dispatch_custodian", func(v string) { device.DispatchCustodian = v })
		setDetailString(device, details, "applicant", func(v string) { device.Applicant = v })
		setDetailString(device, details, "project_name", func(v string) { device.ProjectName = v })
		setDetailString(device, details, "business_unit", func(v string) { device.BusinessUnit = v })
		setDetailString(device, details, "department", func(v string) { device.Department = v })

	case OpScrap:
		device.StorageLocation = ""
		device.DispatchAddress = ""
		device.DispatchCustodian = ""
		setDetailString(device, details, "scrap_remark", func(v string) { device.ScrapRemark = v })

	case OpUnrack:
		// Keep rack fields for reference, set storage location for physical move
		setDetailString(device, details, "storage_location", func(v string) { device.StorageLocation = v })
		setDetailString(device, details, "custodian", func(v string) { device.Custodian = v })

	case OpInStockRecycle:
		// Clear rack fields when recycling from unracked state
		device.Datacenter = ""
		device.Cabinet = ""
		device.UPosition = ""
		device.StartU = nil
		device.EndU = nil
		device.UCount = nil
		device.DatacenterID = nil
		device.CabinetID = nil
		device.VipAddress = ""
		device.BusinessAddress = ""
		setDetailString(device, details, "storage_location", func(v string) { device.StorageLocation = v })
		setDetailString(device, details, "custodian", func(v string) { device.Custodian = v })
	}

	return nil
}

// CheckUPositionConflict checks if the given U position range conflicts with existing devices.
func CheckUPositionConflict(db *gorm.DB, datacenter, cabinet string, startU, endU int, excludeDeviceID uint) error {
	var count int64
	db.Model(&models.Device{}).
		Where("datacenter = ? AND cabinet = ? AND start_u IS NOT NULL AND end_u IS NOT NULL AND id != ?",
			datacenter, cabinet, excludeDeviceID).
		Where("start_u <= ? AND end_u >= ?", endU, startU).
		Count(&count)
	if count > 0 {
		return errors.New("U位冲突：该位置已有设备占用")
	}
	return nil
}

// helper functions for setting detail fields
func setDetailString(_ *models.Device, details map[string]any, key string, setter func(string)) {
	if v, ok := details[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			setter(s)
		}
	}
}

func setDetailInt(_ *models.Device, details map[string]any, key string, setter func(int)) {
	if v, ok := details[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			setter(int(n))
		case int:
			setter(n)
		}
	}
}

func setDetailUint(_ *models.Device, details map[string]any, key string, setter func(uint)) {
	if v, ok := details[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			setter(uint(n))
		case int:
			setter(uint(n))
		}
	}
}

// GenerateApprovalNo generates an approval number like APR-20260412-001
func GenerateApprovalNo(db *gorm.DB) string {
	today := time.Now().Format("20060102")
	prefix := "APR-" + today + "-"

	var count int64
	db.Model(&models.Approval{}).Where("approval_no LIKE ?", prefix+"%").Count(&count)

	return fmt.Sprintf("%s%03d", prefix, count+1)
}
