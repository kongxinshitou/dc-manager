package mcp

import (
	"dcmanager/database"
	"dcmanager/models"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// MCP protocol types
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolCallResult struct {
	Content []TextContent `json:"content"`
}

func getTools() []Tool {
	return []Tool{
		{
			Name:        "query_devices",
			Description: "查询数据中心设备台账，支持按机房、机柜、品牌、型号、IP、负责人等字段过滤",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"datacenter":  map[string]interface{}{"type": "string", "description": "机房名称"},
					"cabinet":     map[string]interface{}{"type": "string", "description": "机柜号"},
					"brand":       map[string]interface{}{"type": "string", "description": "设备品牌"},
					"model":       map[string]interface{}{"type": "string", "description": "设备型号"},
					"device_type": map[string]interface{}{"type": "string", "description": "设备类型"},
					"ip_address":  map[string]interface{}{"type": "string", "description": "IP地址"},
					"owner":       map[string]interface{}{"type": "string", "description": "责任人"},
					"keyword":     map[string]interface{}{"type": "string", "description": "全局关键字搜索"},
					"page":        map[string]interface{}{"type": "integer", "description": "页码，默认1"},
					"page_size":   map[string]interface{}{"type": "integer", "description": "每页数量，默认20"},
				},
			},
		},
		{
			Name:        "add_device",
			Description: "新增数据中心设备台账记录",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"source":       map[string]interface{}{"type": "string", "description": "来源区域"},
					"status":       map[string]interface{}{"type": "string", "description": "状态，如Online/Offline"},
					"datacenter":   map[string]interface{}{"type": "string", "description": "机房名称"},
					"cabinet":      map[string]interface{}{"type": "string", "description": "机柜号"},
					"u_position":   map[string]interface{}{"type": "string", "description": "U位置"},
					"brand":        map[string]interface{}{"type": "string", "description": "设备品牌"},
					"model":        map[string]interface{}{"type": "string", "description": "设备型号"},
					"device_type":  map[string]interface{}{"type": "string", "description": "设备类型"},
					"serial_number": map[string]interface{}{"type": "string", "description": "序列号"},
					"os":           map[string]interface{}{"type": "string", "description": "操作系统"},
					"ip_address":   map[string]interface{}{"type": "string", "description": "IP地址"},
					"mgmt_ip":      map[string]interface{}{"type": "string", "description": "远程管理IP"},
					"purpose":      map[string]interface{}{"type": "string", "description": "设备用途"},
					"owner":        map[string]interface{}{"type": "string", "description": "责任人"},
					"remark":       map[string]interface{}{"type": "string", "description": "备注"},
				},
				"required": []string{"datacenter", "brand", "model", "device_type"},
			},
		},
		{
			Name:        "delete_device",
			Description: "删除数据中心设备台账记录",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "integer", "description": "设备ID"},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "query_inspections",
			Description: "查询巡检记录，支持按机房、机柜、巡检人、等级、状态、时间范围过滤",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"datacenter": map[string]interface{}{"type": "string", "description": "机房名称"},
					"cabinet":    map[string]interface{}{"type": "string", "description": "机柜号"},
					"inspector":  map[string]interface{}{"type": "string", "description": "巡检人"},
					"severity":   map[string]interface{}{"type": "string", "description": "问题等级：严重/一般/轻微"},
					"status":     map[string]interface{}{"type": "string", "description": "状态：待处理/处理中/已解决"},
					"start_time": map[string]interface{}{"type": "string", "description": "开始时间 YYYY-MM-DD"},
					"end_time":   map[string]interface{}{"type": "string", "description": "结束时间 YYYY-MM-DD"},
					"keyword":    map[string]interface{}{"type": "string", "description": "关键字搜索"},
					"page":       map[string]interface{}{"type": "integer", "description": "页码"},
					"page_size":  map[string]interface{}{"type": "integer", "description": "每页数量"},
				},
			},
		},
		{
			Name:        "add_inspection",
			Description: "新增巡检记录",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"device_id":  map[string]interface{}{"type": "integer", "description": "关联设备ID（可选）"},
					"datacenter": map[string]interface{}{"type": "string", "description": "机房"},
					"cabinet":    map[string]interface{}{"type": "string", "description": "机柜"},
					"u_position": map[string]interface{}{"type": "string", "description": "U位"},
					"found_at":   map[string]interface{}{"type": "string", "description": "发现时间 RFC3339格式，默认当前时间"},
					"inspector":  map[string]interface{}{"type": "string", "description": "巡检人"},
					"issue":      map[string]interface{}{"type": "string", "description": "问题描述"},
					"severity":   map[string]interface{}{"type": "string", "description": "等级：严重/一般/轻微"},
					"status":     map[string]interface{}{"type": "string", "description": "状态：待处理/处理中/已解决"},
					"remark":     map[string]interface{}{"type": "string", "description": "备注"},
				},
				"required": []string{"datacenter", "inspector", "issue", "severity", "status"},
			},
		},
		{
			Name:        "delete_inspection",
			Description: "删除巡检记录",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "integer", "description": "巡检记录ID"},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "get_issue_status",
			Description: "获取数据中心巡检问题状态统计，包括各机房问题数量、严重等级分布、状态分布",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

func handleQueryDevices(args json.RawMessage) (string, error) {
	var params struct {
		Datacenter string `json:"datacenter"`
		Cabinet    string `json:"cabinet"`
		Brand      string `json:"brand"`
		Model      string `json:"model"`
		DeviceType string `json:"device_type"`
		IPAddress  string `json:"ip_address"`
		Owner      string `json:"owner"`
		Keyword    string `json:"keyword"`
		Page       int    `json:"page"`
		PageSize   int    `json:"page_size"`
	}
	json.Unmarshal(args, &params)
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	db := database.DB.Model(&models.Device{})
	if params.Datacenter != "" {
		db = db.Where("datacenter LIKE ?", "%"+params.Datacenter+"%")
	}
	if params.Cabinet != "" {
		db = db.Where("cabinet LIKE ?", "%"+params.Cabinet+"%")
	}
	if params.Brand != "" {
		db = db.Where("brand LIKE ?", "%"+params.Brand+"%")
	}
	if params.Model != "" {
		db = db.Where("model LIKE ?", "%"+params.Model+"%")
	}
	if params.DeviceType != "" {
		db = db.Where("device_type LIKE ?", "%"+params.DeviceType+"%")
	}
	if params.IPAddress != "" {
		db = db.Where("ip_address LIKE ?", "%"+params.IPAddress+"%")
	}
	if params.Owner != "" {
		db = db.Where("owner LIKE ?", "%"+params.Owner+"%")
	}
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		db = db.Where("datacenter LIKE ? OR cabinet LIKE ? OR brand LIKE ? OR model LIKE ? OR serial_number LIKE ? OR ip_address LIKE ? OR purpose LIKE ? OR owner LIKE ?",
			kw, kw, kw, kw, kw, kw, kw, kw)
	}

	var total int64
	db.Count(&total)
	var devices []models.Device
	db.Offset((params.Page - 1) * params.PageSize).Limit(params.PageSize).Find(&devices)

	result := map[string]interface{}{"total": total, "page": params.Page, "data": devices}
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func handleAddDevice(args json.RawMessage) (string, error) {
	var device models.Device
	if err := json.Unmarshal(args, &device); err != nil {
		return "", err
	}
	device.ID = 0
	if err := database.DB.Create(&device).Error; err != nil {
		return "", err
	}
	b, _ := json.MarshalIndent(device, "", "  ")
	return string(b), nil
}

func handleDeleteDevice(args json.RawMessage) (string, error) {
	var params struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}
	if err := database.DB.Delete(&models.Device{}, params.ID).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"message": "设备 %d 已删除"}`, params.ID), nil
}

func handleQueryInspections(args json.RawMessage) (string, error) {
	var params struct {
		Datacenter string `json:"datacenter"`
		Cabinet    string `json:"cabinet"`
		Inspector  string `json:"inspector"`
		Severity   string `json:"severity"`
		Status     string `json:"status"`
		StartTime  string `json:"start_time"`
		EndTime    string `json:"end_time"`
		Keyword    string `json:"keyword"`
		Page       int    `json:"page"`
		PageSize   int    `json:"page_size"`
	}
	json.Unmarshal(args, &params)
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	db := database.DB.Model(&models.Inspection{}).Preload("Device")
	if params.Datacenter != "" {
		db = db.Where("datacenter LIKE ?", "%"+params.Datacenter+"%")
	}
	if params.Cabinet != "" {
		db = db.Where("cabinet LIKE ?", "%"+params.Cabinet+"%")
	}
	if params.Inspector != "" {
		db = db.Where("inspector LIKE ?", "%"+params.Inspector+"%")
	}
	if params.Severity != "" {
		db = db.Where("severity = ?", params.Severity)
	}
	if params.Status != "" {
		db = db.Where("status = ?", params.Status)
	}
	if params.StartTime != "" {
		t, err := time.Parse("2006-01-02", params.StartTime)
		if err == nil {
			db = db.Where("found_at >= ?", t)
		}
	}
	if params.EndTime != "" {
		t, err := time.Parse("2006-01-02", params.EndTime)
		if err == nil {
			db = db.Where("found_at <= ?", t.Add(24*time.Hour))
		}
	}
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		db = db.Where("datacenter LIKE ? OR cabinet LIKE ? OR inspector LIKE ? OR issue LIKE ?", kw, kw, kw, kw)
	}

	var total int64
	db.Count(&total)
	var inspections []models.Inspection
	db.Order("found_at DESC").Offset((params.Page - 1) * params.PageSize).Limit(params.PageSize).Find(&inspections)

	result := map[string]interface{}{"total": total, "page": params.Page, "data": inspections}
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func handleAddInspection(args json.RawMessage) (string, error) {
	var inspection models.Inspection
	if err := json.Unmarshal(args, &inspection); err != nil {
		return "", err
	}
	inspection.ID = 0
	if inspection.FoundAt.IsZero() {
		inspection.FoundAt = time.Now()
	}
	if err := database.DB.Create(&inspection).Error; err != nil {
		return "", err
	}
	database.DB.Preload("Device").First(&inspection, inspection.ID)
	b, _ := json.MarshalIndent(inspection, "", "  ")
	return string(b), nil
}

func handleDeleteInspection(args json.RawMessage) (string, error) {
	var params struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}
	if err := database.DB.Delete(&models.Inspection{}, params.ID).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"message": "巡检记录 %d 已删除"}`, params.ID), nil
}

func handleGetIssueStatus(args json.RawMessage) (string, error) {
	type RoomStat struct {
		Datacenter string `json:"datacenter"`
		Count      int64  `json:"count"`
	}
	type StatusStat struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	type SeverityStat struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}

	var roomStats []RoomStat
	database.DB.Model(&models.Inspection{}).Select("datacenter, count(*) as count").
		Where("status != ?", "已解决").Group("datacenter").Scan(&roomStats)

	var statusStats []StatusStat
	database.DB.Model(&models.Inspection{}).Select("status, count(*) as count").
		Group("status").Scan(&statusStats)

	var severityStats []SeverityStat
	database.DB.Model(&models.Inspection{}).Select("severity, count(*) as count").
		Where("status != ?", "已解决").Group("severity").Scan(&severityStats)

	var totalUnresolved int64
	database.DB.Model(&models.Inspection{}).Where("status != ?", "已解决").Count(&totalUnresolved)

	result := map[string]interface{}{
		"total_unresolved": totalUnresolved,
		"by_room":          roomStats,
		"by_status":        statusStats,
		"by_severity":      severityStats,
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func HandleMCP(c *gin.Context) {
	var req MCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := MCPResponse{JSONRPC: "2.0", ID: req.ID}

	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":      map[string]interface{}{"name": "dc-manager-mcp", "version": "1.0.0"},
		}
	case "tools/list":
		resp.Result = ToolsListResult{Tools: getTools()}
	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &MCPError{Code: -32602, Message: "Invalid params"}
			c.JSON(http.StatusOK, resp)
			return
		}

		var resultText string
		var err error

		switch params.Name {
		case "query_devices":
			resultText, err = handleQueryDevices(params.Arguments)
		case "add_device":
			resultText, err = handleAddDevice(params.Arguments)
		case "delete_device":
			resultText, err = handleDeleteDevice(params.Arguments)
		case "query_inspections":
			resultText, err = handleQueryInspections(params.Arguments)
		case "add_inspection":
			resultText, err = handleAddInspection(params.Arguments)
		case "delete_inspection":
			resultText, err = handleDeleteInspection(params.Arguments)
		case "get_issue_status":
			resultText, err = handleGetIssueStatus(params.Arguments)
		default:
			resp.Error = &MCPError{Code: -32601, Message: "Tool not found: " + params.Name}
			c.JSON(http.StatusOK, resp)
			return
		}

		if err != nil {
			resp.Error = &MCPError{Code: -32000, Message: err.Error()}
		} else {
			resp.Result = ToolCallResult{Content: []TextContent{{Type: "text", Text: resultText}}}
		}
	default:
		resp.Error = &MCPError{Code: -32601, Message: "Method not found"}
	}

	c.JSON(http.StatusOK, resp)
}
