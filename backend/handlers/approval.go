package handlers

import (
	"dcmanager/auth"
	"dcmanager/database"
	"dcmanager/models"
	"dcmanager/services"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// SubmitApproval creates a new approval request
func SubmitApproval(c *gin.Context) {
	var body struct {
		DeviceID      uint            `json:"device_id" binding:"required"`
		OperationType string          `json:"operation_type" binding:"required"`
		RequestData   json.RawMessage `json:"request_data"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify device exists
	var device models.Device
	if err := database.DB.First(&device, body.DeviceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}

	// Validate transition
	if _, _, err := services.ValidateTransition(device.DeviceStatus, device.SubStatus, body.OperationType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	username := getUsername(c)

	approvalNo := services.GenerateApprovalNo(database.DB)

	approval := models.Approval{
		ApprovalNo:    approvalNo,
		DeviceID:      body.DeviceID,
		OperationType: body.OperationType,
		RequestData:   string(body.RequestData),
		ApplicantID:   userID,
		ApplicantName: username,
		Status:        "pending",
	}

	if err := database.DB.Create(&approval).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, approval)
}

// GetApprovals lists approvals with filtering
func GetApprovals(c *gin.Context) {
	var query models.ApprovalQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	db := database.DB.Model(&models.Approval{})

	userID := getUserID(c)
	isAdminOrApprover := hasPermission(c, "approval:approve")

	switch query.Tab {
	case "pending":
		db = db.Where("status = ?", "pending")
	case "my_requests":
		db = db.Where("applicant_id = ?", userID)
	case "all":
		// no filter
	default:
		// Default: show pending + my requests
		if !isAdminOrApprover {
			db = db.Where("applicant_id = ? OR (status = ? AND 1 = ?)", userID, "pending", boolToInt(isAdminOrApprover))
		}
	}

	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.OperationType != "" {
		db = db.Where("operation_type = ?", query.OperationType)
	}

	var total int64
	db.Count(&total)

	var approvals []models.Approval
	db.Order("created_at desc").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&approvals)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  query.Page,
		"data":  approvals,
	})
}

// GetApproval returns approval detail
func GetApproval(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var approval models.Approval
	if err := database.DB.First(&approval, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "审批单不存在"})
		return
	}

	// Get associated device
	var device models.Device
	database.DB.First(&device, approval.DeviceID)

	c.JSON(http.StatusOK, gin.H{
		"approval": approval,
		"device":   device,
	})
}

// ApproveApproval approves an approval
func ApproveApproval(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var approval models.Approval
	if err := database.DB.First(&approval, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "审批单不存在"})
		return
	}

	if approval.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能审批待审批状态的申请"})
		return
	}

	var body struct {
		Remark string `json:"remark"`
	}
	c.ShouldBindJSON(&body)

	userID := getUserID(c)
	username := getUsername(c)
	now := time.Now()

	approval.ApproverID = &userID
	approval.ApproverName = username
	approval.Status = "approved"
	approval.ApproveRemark = body.Remark
	approval.ApprovedAt = &now

	database.DB.Save(&approval)
	c.JSON(http.StatusOK, approval)
}

// RejectApproval rejects an approval
func RejectApproval(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var approval models.Approval
	if err := database.DB.First(&approval, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "审批单不存在"})
		return
	}

	if approval.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能驳回待审批状态的申请"})
		return
	}

	var body struct {
		Remark string `json:"remark"`
	}
	c.ShouldBindJSON(&body)

	userID := getUserID(c)
	username := getUsername(c)
	now := time.Now()

	approval.ApproverID = &userID
	approval.ApproverName = username
	approval.Status = "rejected"
	approval.ApproveRemark = body.Remark
	approval.ApprovedAt = &now

	database.DB.Save(&approval)
	c.JSON(http.StatusOK, approval)
}

// ExecuteApproval executes an approved operation
func ExecuteApproval(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var approval models.Approval
	if err := database.DB.First(&approval, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "审批单不存在"})
		return
	}

	if approval.Status != "approved" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能执行已批准的操作"})
		return
	}

	// Get device
	var device models.Device
	if err := database.DB.First(&device, approval.DeviceID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "关联设备不存在"})
		return
	}

	// Execute the transition
	userID := getUserID(c)
	now := time.Now()

	if err := services.ExecuteTransition(database.DB, &device, approval.OperationType, json.RawMessage(approval.RequestData), userID, &approval.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("执行失败: %s", err.Error())})
		return
	}

	approval.Status = "executed"
	approval.ExecutedAt = &now
	database.DB.Save(&approval)

	c.JSON(http.StatusOK, gin.H{
		"approval": approval,
		"device":   device,
	})
}

// CancelApproval cancels a pending approval (by applicant)
func CancelApproval(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var approval models.Approval
	if err := database.DB.First(&approval, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "审批单不存在"})
		return
	}

	if approval.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能取消待审批状态的申请"})
		return
	}

	userID := getUserID(c)
	if approval.ApplicantID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能取消自己提交的申请"})
		return
	}

	approval.Status = "cancelled"
	database.DB.Save(&approval)
	c.JSON(http.StatusOK, approval)
}

// helper: get username from context
func getUsername(c *gin.Context) string {
	if claims, exists := c.Get("currentUser"); exists {
		if cl, ok := claims.(*auth.Claims); ok {
			return cl.Username
		}
	}
	return ""
}

// helper: check if user has permission
func hasPermission(c *gin.Context, perm string) bool {
	if claims, exists := c.Get("currentUser"); exists {
		if cl, ok := claims.(*auth.Claims); ok {
			if cl.RoleName == "admin" {
				return true
			}
			for _, p := range cl.Permissions {
				if p == perm {
					return true
				}
			}
		}
	}
	return false
}

// helper: bool to int for SQL
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
