package mcp

import (
	"dcmanager/config"
	"dcmanager/database"
	"dcmanager/models"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	uRangeRe     = regexp.MustCompile(`^(\d+)\s*[-~]\s*(\d+)\s*[Uu]$`)
	uSingleRe    = regexp.MustCompile(`^(\d+)\s*[Uu]$`)
	datacenterRe = regexp.MustCompile(`(?i)^(IDC)?\s*(\d+)\s*[-]\s*(\d+)$`)
	cabinetRe    = regexp.MustCompile(`^([A-Za-z]+)[\s\-]*(\d+)$`)
)

// normalizeDatacenter 标准化机房名称: "1-2" → "IDC1-2", "idc1-2" → "IDC1-2"
func normalizeDatacenter(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	m := datacenterRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	return fmt.Sprintf("IDC%s-%s", m[2], m[3])
}

// normalizeCabinet 标准化机柜号: "A01" → "A-01", "B17" → "B-17", "A-01" → "A-01"
func normalizeCabinet(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	m := cabinetRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	letter := strings.ToUpper(m[1])
	num := m[2]
	if len(num) == 1 {
		num = "0" + num
	}
	return fmt.Sprintf("%s-%s", letter, num)
}

// normalizeUPosition 标准化U位: "5-6U" → "05-06U", "1U" → "01U"
func normalizeUPosition(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if m := uRangeRe.FindStringSubmatch(raw); m != nil {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		return fmt.Sprintf("%02d-%02dU", a, b)
	}
	if m := uSingleRe.FindStringSubmatch(raw); m != nil {
		a, _ := strconv.Atoi(m[1])
		return fmt.Sprintf("%02dU", a)
	}
	return raw
}

// parseUPosition parses "04-05U" → (4,5), "04U" → (4,4), others → (nil,nil)
func parseUPosition(pos string) (startU, endU *int) {
	pos = strings.TrimSpace(pos)
	if pos == "" {
		return nil, nil
	}
	if m := uRangeRe.FindStringSubmatch(pos); m != nil {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		return &a, &b
	}
	if m := uSingleRe.FindStringSubmatch(pos); m != nil {
		a, _ := strconv.Atoi(m[1])
		return &a, &a
	}
	return nil, nil
}

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

// SSE session store
var sessions sync.Map // sessionID -> chan string

// validateMCPKey checks if the request provides a valid MCP API key.
// Returns true if the key is valid or if no key is configured (open mode).
func validateMCPKey(c *gin.Context) bool {
	if config.MCPAPIKey == "" {
		return true // open mode, no key configured
	}

	// Check Authorization: Bearer <key> header
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == config.MCPAPIKey {
			return true
		}
	}

	// Check api_key query parameter (needed for SSE/EventSource which cannot set custom headers)
	if c.Query("api_key") == config.MCPAPIKey {
		return true
	}

	return false
}

func getTools() []Tool {
	return []Tool{
		{
			Name:        "query_devices",
			Description: "查询数据中心设备台账，支持按机房、机柜、品牌、型号、IP、负责人等字段过滤, 调用这个接口时候, 注意属性字段的标准化, 具体可见属性字段的description如: datacenter, cabinet, u_position等字段的格式要求和规范化说明",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"datacenter":  map[string]interface{}{"type": "string", "description": "机房名称, 格式通常为IDC2-1, IDC1-2, IDC1-1, 如果遇到不规则的机房名称, 如1-1则可以改为IDC1-1, 2-1改为IDC2-1, 以此类推"},
					"cabinet":     map[string]interface{}{"type": "string", "description": "机柜号, 格式通常为A-01, A-02, B-01, B-02, C-01, C-02, D-01, D-02, E-01, E-02, F-01, F-02, 如果遇到不规则的机柜, 如A01则可以改为A-01, B1改为B-01, 以此类推"},
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
			Description: "新增数据中心设备台账记录, 调用这个接口时候, 注意属性字段的标准化, 具体可见属性字段的description如: datacenter, cabinet, u_position等字段的格式要求和规范化说明",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"source":        map[string]interface{}{"type": "string", "description": "来源区域"},
					"status":        map[string]interface{}{"type": "string", "description": "状态，如Online/Offline"},
					"datacenter":    map[string]interface{}{"type": "string", "description": "机房名称, 格式通常为IDC2-1, IDC1-2, IDC1-1, 如果遇到不规则的机房名称, 如1-1则可以改为IDC1-1, 2-1改为IDC2-1, 以此类推"},
					"cabinet":       map[string]interface{}{"type": "string", "description": "机柜号, 格式通常为A-01, A-02, B-01, B-02, C-01, C-02, D-01, D-02, E-01, E-02, F-01, F-02, 如果遇到不规则的机柜, 如A01则可以改为A-01, B1改为B-01, 以此类推"},
					"u_position":    map[string]interface{}{"type": "string", "description": "U位置, 格式通常为01U, 03-10U, 如果发现个位数省略前面的0则加上去, 如1U改为01U, 5-6U改为05-06U, 以此类推, 其他情况则为空"},
					"brand":         map[string]interface{}{"type": "string", "description": "设备品牌"},
					"model":         map[string]interface{}{"type": "string", "description": "设备型号"},
					"device_type":   map[string]interface{}{"type": "string", "description": "设备类型"},
					"serial_number": map[string]interface{}{"type": "string", "description": "序列号"},
					"os":            map[string]interface{}{"type": "string", "description": "操作系统"},
					"ip_address":    map[string]interface{}{"type": "string", "description": "IP地址"},
					"mgmt_ip":       map[string]interface{}{"type": "string", "description": "远程管理IP"},
					"purpose":       map[string]interface{}{"type": "string", "description": "设备用途"},
					"owner":         map[string]interface{}{"type": "string", "description": "责任人"},
					"remark":        map[string]interface{}{"type": "string", "description": "备注"},
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
			Name:        "batch_delete_devices",
			Description: "批量删除数据中心设备台账记录，接受设备ID数组，单次最多500条",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ids": map[string]interface{}{
						"type":        "array",
						"description": "设备ID数组",
						"items":       map[string]interface{}{"type": "integer"},
					},
				},
				"required": []string{"ids"},
			},
		},
		{
			Name:        "query_inspections",
			Description: "查询巡检记录，支持按机房、机柜、巡检人、等级、状态、时间范围过滤, 调用这个接口时候, 注意属性字段的标准化, 具体可见属性字段的description如: datacenter, cabinet, u_position等字段的格式要求和规范化说明",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"datacenter": map[string]interface{}{"type": "string", "description": "机房名称, 格式通常为IDC2-1, IDC1-2, IDC1-1, 如果遇到不规则的机房名称, 如1-1则可以改为IDC1-1, 2-1改为IDC2-1, 以此类推"},
					"cabinet":    map[string]interface{}{"type": "string", "description": "机柜号, 格式通常为A-01, A-02, B-01, B-02, C-01, C-02, D-01, D-02, E-01, E-02, F-01, F-02, 如果遇到不规则的机柜, 如A01则可以改为A-01, B1改为B-01, 以此类推"},
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
			Description: "新增巡检记录。当提供机房、机柜、U位信息时，会自动匹配关联对应的设备，无需手动指定device_id, 调用这个接口时候, 注意属性字段的标准化, 具体可见属性字段的description如: datacenter, cabinet, u_position等字段的格式要求和规范化说明",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"device_id":  map[string]interface{}{"type": "integer", "description": "关联设备ID（可选）"},
					"datacenter": map[string]interface{}{"type": "string", "description": "机房名称, 格式通常为IDC2-1, IDC1-2, IDC1-1, 如果遇到不规则的机房名称, 如1-1则可以改为IDC1-1, 2-1改为IDC2-1, 以此类推"},
					"cabinet":    map[string]interface{}{"type": "string", "description": "机柜号, 格式通常为A-01, A-02, B-01, B-02, C-01, C-02, D-01, D-02, E-01, E-02, F-01, F-02, 如果遇到不规则的机柜, 如A01则可以改为A-01, B1改为B-01, 以此类推"},
					"u_position": map[string]interface{}{"type": "string", "description": "U位置, 格式通常为01U, 03-10U, 如果发现个位数省略前面的0则加上去, 如1U改为01U, 5-6U改为05-06U, 以此类推, 其他情况则为空"},
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
			Name:        "batch_delete_inspections",
			Description: "批量删除巡检记录，接受巡检记录ID数组，单次最多500条",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ids": map[string]interface{}{
						"type":        "array",
						"description": "巡检记录ID数组",
						"items":       map[string]interface{}{"type": "integer"},
					},
				},
				"required": []string{"ids"},
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
		{
			Name:        "batch_add_inspections",
			Description: "批量新增巡检记录，接受JSON数组。每条记录自动标准化机房/机柜/U位字段，严重程度和状态会自动归一化，并尝试关联对应设备。单次最多500条。",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"inspections": map[string]interface{}{
						"type":        "array",
						"description": "巡检记录数组",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"datacenter": map[string]interface{}{"type": "string", "description": "机房名称, 格式通常为IDC2-1, IDC1-2, IDC1-1, 如果遇到不规则的机房名称, 如1-1则可以改为IDC1-1"},
								"cabinet":    map[string]interface{}{"type": "string", "description": "机柜号, 格式通常为A-01, A-02, B-01, 如A01则改为A-01"},
								"u_position": map[string]interface{}{"type": "string", "description": "U位置, 格式通常为01U, 03-10U, 如1U改为01U"},
								"found_at":   map[string]interface{}{"type": "string", "description": "发现时间 RFC3339格式，默认当前时间"},
								"inspector":  map[string]interface{}{"type": "string", "description": "巡检人"},
								"issue":      map[string]interface{}{"type": "string", "description": "问题描述"},
								"severity":   map[string]interface{}{"type": "string", "description": "等级：严重/一般/轻微，也支持critical/high/low等英文"},
								"status":     map[string]interface{}{"type": "string", "description": "状态：待处理/处理中/已解决，也支持open/processing/resolved等英文"},
								"remark":     map[string]interface{}{"type": "string", "description": "备注"},
							},
							"required": []string{"inspector", "issue"},
						},
					},
				},
				"required": []string{"inspections"},
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

	// 标准化查询字段
	params.Datacenter = normalizeDatacenter(params.Datacenter)
	params.Cabinet = normalizeCabinet(params.Cabinet)

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
	var params struct {
		Source       string `json:"source"`
		Status       string `json:"status"`
		Datacenter   string `json:"datacenter"`
		Cabinet      string `json:"cabinet"`
		UPosition    string `json:"u_position"`
		Brand        string `json:"brand"`
		Model        string `json:"model"`
		DeviceType   string `json:"device_type"`
		SerialNumber string `json:"serial_number"`
		OS           string `json:"os"`
		IPAddress    string `json:"ip_address"`
		MgmtIP       string `json:"mgmt_ip"`
		Purpose      string `json:"purpose"`
		Owner        string `json:"owner"`
		Remark       string `json:"remark"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	// 标准化字段
	params.Datacenter = normalizeDatacenter(params.Datacenter)
	params.Cabinet = normalizeCabinet(params.Cabinet)
	params.UPosition = normalizeUPosition(params.UPosition)
	startU, endU := parseUPosition(params.UPosition)

	device := models.Device{
		Source:       params.Source,
		Status:       params.Status,
		Datacenter:   params.Datacenter,
		Cabinet:      params.Cabinet,
		UPosition:    params.UPosition,
		StartU:       startU,
		EndU:         endU,
		Brand:        params.Brand,
		Model:        params.Model,
		DeviceType:   params.DeviceType,
		SerialNumber: params.SerialNumber,
		OS:           params.OS,
		IPAddress:    params.IPAddress,
		MgmtIP:       params.MgmtIP,
		Purpose:      params.Purpose,
		Owner:        params.Owner,
		Remark:       params.Remark,
	}
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

func handleBatchDeleteDevices(args json.RawMessage) (string, error) {
	var params struct {
		IDs []uint `json:"ids"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}
	if len(params.IDs) == 0 {
		return "", fmt.Errorf("ids数组不能为空")
	}
	if len(params.IDs) > 500 {
		return "", fmt.Errorf("单次最多删除500条，当前%d条", len(params.IDs))
	}
	result := database.DB.Delete(&models.Device{}, params.IDs)
	if result.Error != nil {
		return "", result.Error
	}
	b, _ := json.Marshal(map[string]interface{}{
		"deleted": result.RowsAffected,
		"message": fmt.Sprintf("成功删除%d条设备记录", result.RowsAffected),
	})
	return string(b), nil
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

	// 标准化查询字段
	params.Datacenter = normalizeDatacenter(params.Datacenter)
	params.Cabinet = normalizeCabinet(params.Cabinet)

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
	// 使用中间结构体解析，避免 time.Time 反序列化失败导致整个请求被拒绝
	var params struct {
		DeviceID   *uint  `json:"device_id"`
		Datacenter string `json:"datacenter"`
		Cabinet    string `json:"cabinet"`
		UPosition  string `json:"u_position"`
		FoundAt    string `json:"found_at"`
		Inspector  string `json:"inspector"`
		Issue      string `json:"issue"`
		Severity   string `json:"severity"`
		Status     string `json:"status"`
		Remark     string `json:"remark"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	// 标准化字段
	params.Datacenter = normalizeDatacenter(params.Datacenter)
	params.Cabinet = normalizeCabinet(params.Cabinet)
	params.UPosition = normalizeUPosition(params.UPosition)

	// 解析时间
	var foundAt time.Time
	if params.FoundAt != "" {
		if t, err := time.Parse(time.RFC3339, params.FoundAt); err == nil {
			foundAt = t
		}
	}
	if foundAt.IsZero() {
		foundAt = time.Now()
	}

	// 解析U位
	startU, endU := parseUPosition(params.UPosition)

	inspection := models.Inspection{
		DeviceID:   params.DeviceID,
		Datacenter: params.Datacenter,
		Cabinet:    params.Cabinet,
		UPosition:  params.UPosition,
		StartU:     startU,
		EndU:       endU,
		FoundAt:    foundAt,
		Inspector:  params.Inspector,
		Issue:      params.Issue,
		Severity:   params.Severity,
		Status:     params.Status,
		Remark:     params.Remark,
	}

	// 如果未指定 device_id，但提供了机房+机柜，则自动匹配关联设备
	var matchedDevice *models.Device
	if inspection.DeviceID == nil && inspection.Datacenter != "" && inspection.Cabinet != "" {
		matchedDevice = autoMatchDevice(inspection.Datacenter, inspection.Cabinet, inspection.UPosition)
		if matchedDevice != nil {
			inspection.DeviceID = &matchedDevice.ID
		}
	}

	if err := database.DB.Create(&inspection).Error; err != nil {
		return "", err
	}
	database.DB.Preload("Device").First(&inspection, inspection.ID)

	// 构建返回结果，包含匹配信息
	result := map[string]interface{}{
		"inspection": inspection,
	}
	if matchedDevice != nil {
		result["auto_matched"] = true
		result["matched_device_summary"] = fmt.Sprintf("自动关联设备: %s %s (ID:%d, %s/%s/%s)",
			matchedDevice.Brand, matchedDevice.Model, matchedDevice.ID,
			matchedDevice.Datacenter, matchedDevice.Cabinet, matchedDevice.UPosition)
	} else if inspection.DeviceID == nil && inspection.Datacenter != "" && inspection.Cabinet != "" && inspection.UPosition != "" {
		result["auto_matched"] = false
		result["match_hint"] = "未能自动匹配到设备，请确认机房、机柜、U位信息是否正确"
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

// autoMatchDevice 根据机房、机柜、U位自动匹配设备
func autoMatchDevice(datacenter, cabinet, uPosition string) *models.Device {
	db := database.DB.Where("datacenter = ? AND cabinet = ?", datacenter, cabinet)

	startU, endU := parseUPosition(uPosition)

	if startU != nil && endU != nil {
		// 有U位信息：查找U位范围有重叠的设备
		// 设备的 [StartU, EndU] 与巡检的 [startU, endU] 有交集
		var devices []models.Device
		db.Where("start_u IS NOT NULL AND end_u IS NOT NULL AND start_u <= ? AND end_u >= ?", *endU, *startU).
			Find(&devices)

		if len(devices) == 1 {
			return &devices[0]
		}
		// 多个设备匹配时，优先选择U位完全匹配的
		if len(devices) > 1 {
			for i := range devices {
				if devices[i].StartU != nil && devices[i].EndU != nil &&
					*devices[i].StartU == *startU && *devices[i].EndU == *endU {
					return &devices[i]
				}
			}
			// 没有完全匹配的，返回第一个（范围最相关的）
			return &devices[0]
		}
	} else {
		// 没有U位信息，只按机房+机柜匹配，仅当该位置只有一台设备时才自动关联
		var devices []models.Device
		db.Find(&devices)
		if len(devices) == 1 {
			return &devices[0]
		}
	}

	return nil
}

func normalizeSeverityMCP(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "严重", "紧急", "critical", "high", "p0", "p1":
		return "严重"
	case "轻微", "低", "low", "minor", "p3", "p4":
		return "轻微"
	default:
		return "一般"
	}
}

func normalizeStatusMCP(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "处理中", "进行中", "in progress", "processing":
		return "处理中"
	case "已解决", "已处理", "完成", "resolved", "closed", "done":
		return "已解决"
	default:
		return "待处理"
	}
}

func handleBatchAddInspections(args json.RawMessage) (string, error) {
	var params struct {
		Inspections []struct {
			Datacenter string `json:"datacenter"`
			Cabinet    string `json:"cabinet"`
			UPosition  string `json:"u_position"`
			FoundAt    string `json:"found_at"`
			Inspector  string `json:"inspector"`
			Issue      string `json:"issue"`
			Severity   string `json:"severity"`
			Status     string `json:"status"`
			Remark     string `json:"remark"`
		} `json:"inspections"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}
	if len(params.Inspections) == 0 {
		return "", fmt.Errorf("inspections数组不能为空")
	}
	if len(params.Inspections) > 500 {
		return "", fmt.Errorf("单次最多导入500条，当前%d条", len(params.Inspections))
	}

	inserted := 0
	failed := 0
	var createdRecords []models.Inspection

	for _, item := range params.Inspections {
		datacenter := normalizeDatacenter(item.Datacenter)
		cabinet := normalizeCabinet(item.Cabinet)
		upos := normalizeUPosition(item.UPosition)
		startU, endU := parseUPosition(upos)

		var foundAt time.Time
		if item.FoundAt != "" {
			if t, err := time.Parse(time.RFC3339, item.FoundAt); err == nil {
				foundAt = t
			}
		}
		if foundAt.IsZero() {
			foundAt = time.Now()
		}

		insp := models.Inspection{
			Datacenter: datacenter,
			Cabinet:    cabinet,
			UPosition:  upos,
			StartU:     startU,
			EndU:       endU,
			FoundAt:    foundAt,
			Inspector:  item.Inspector,
			Issue:      item.Issue,
			Severity:   normalizeSeverityMCP(item.Severity),
			Status:     normalizeStatusMCP(item.Status),
			Remark:     item.Remark,
		}

		// Auto-match device
		if insp.DeviceID == nil && datacenter != "" && cabinet != "" {
			matched := autoMatchDevice(datacenter, cabinet, upos)
			if matched != nil {
				insp.DeviceID = &matched.ID
			}
		}

		if err := database.DB.Create(&insp).Error; err != nil {
			failed++
			continue
		}
		database.DB.Preload("Device").First(&insp, insp.ID)
		createdRecords = append(createdRecords, insp)
		inserted++
	}

	result := map[string]interface{}{
		"inserted": inserted,
		"failed":   failed,
		"records":  createdRecords,
		"message":  fmt.Sprintf("批量导入完成，成功%d条，失败%d条", inserted, failed),
	}
	b, _ := json.MarshalIndent(result, "", "  ")
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

func handleBatchDeleteInspections(args json.RawMessage) (string, error) {
	var params struct {
		IDs []uint `json:"ids"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}
	if len(params.IDs) == 0 {
		return "", fmt.Errorf("ids数组不能为空")
	}
	if len(params.IDs) > 500 {
		return "", fmt.Errorf("单次最多删除500条，当前%d条", len(params.IDs))
	}
	result := database.DB.Delete(&models.Inspection{}, params.IDs)
	if result.Error != nil {
		return "", result.Error
	}
	b, _ := json.Marshal(map[string]interface{}{
		"deleted": result.RowsAffected,
		"message": fmt.Sprintf("成功删除%d条巡检记录", result.RowsAffected),
	})
	return string(b), nil
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

func processRequest(req MCPRequest) MCPResponse {
	resp := MCPResponse{JSONRPC: "2.0", ID: req.ID}

	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":      map[string]interface{}{"name": "dc-manager-mcp", "version": "1.0.0"},
		}
	case "notifications/initialized":
		// 客户端通知，无需响应（ID 为 nil）
		resp.ID = nil
		return resp
	case "tools/list":
		resp.Result = ToolsListResult{Tools: getTools()}
	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &MCPError{Code: -32602, Message: "Invalid params"}
			return resp
		}

		var resultText string
		var err error

		switch params.Name {
		case "query_devices":
			resultText, err = handleQueryDevices(params.Arguments)
			fmt.Println(time.Now().String(), "query_devices")
		case "add_device":
			resultText, err = handleAddDevice(params.Arguments)
			fmt.Println(time.Now().String(), "add_device")
		case "delete_device":
			resultText, err = handleDeleteDevice(params.Arguments)
			fmt.Println(time.Now().String(), "delete_device")
		case "batch_delete_devices":
			resultText, err = handleBatchDeleteDevices(params.Arguments)
			fmt.Println(time.Now().String(), "batch_delete_devices")
		case "query_inspections":
			resultText, err = handleQueryInspections(params.Arguments)
			fmt.Println(time.Now().String(), "query_inspections")
		case "add_inspection":
			resultText, err = handleAddInspection(params.Arguments)
			fmt.Println(time.Now().String(), "add_inspection")
		case "delete_inspection":
			resultText, err = handleDeleteInspection(params.Arguments)
			fmt.Println(time.Now().String(), "delete_inspection")
		case "batch_delete_inspections":
			resultText, err = handleBatchDeleteInspections(params.Arguments)
			fmt.Println(time.Now().String(), "batch_delete_inspections")
		case "batch_add_inspections":
			resultText, err = handleBatchAddInspections(params.Arguments)
			fmt.Println(time.Now().String(), "batch_add_inspections")
		case "get_issue_status":
			resultText, err = handleGetIssueStatus(params.Arguments)
			fmt.Println(time.Now().String(), "get_issue_status")
		default:
			resp.Error = &MCPError{Code: -32601, Message: "Tool not found: " + params.Name}
			return resp
		}

		if err != nil {
			resp.Error = &MCPError{Code: -32000, Message: err.Error()}
		} else {
			resp.Result = ToolCallResult{Content: []TextContent{{Type: "text", Text: resultText}}}
		}
	default:
		resp.Error = &MCPError{Code: -32601, Message: "Method not found"}
	}

	return resp
}

// HandleMCPSSE - GET /mcp/sse，建立 SSE 长连接
func HandleMCPSSE(c *gin.Context) {
	if !validateMCPKey(c) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "MCP API key required"})
		return
	}

	sessionID := uuid.New().String()
	ch := make(chan string, 16)
	sessions.Store(sessionID, ch)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no")

	// 发送 endpoint 事件，告诉客户端往哪里 POST 消息
	fmt.Fprintf(c.Writer, "event: endpoint\ndata: /mcp/messages?sessionId=%s\n\n", sessionID)
	c.Writer.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer sessions.Delete(sessionID)

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", msg)
			c.Writer.Flush()
		case <-ticker.C:
			fmt.Fprintf(c.Writer, ": ping\n\n")
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

// HandleMCPMessages - POST /mcp/messages，接收客户端消息并通过 SSE 推送响应
func HandleMCPMessages(c *gin.Context) {
	if !validateMCPKey(c) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "MCP API key required"})
		return
	}

	sessionID := c.Query("sessionId")
	val, ok := sessions.Load(sessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	ch := val.(chan string)

	var req MCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := processRequest(req)

	// notifications 无需响应
	if resp.ID == nil && req.ID == nil {
		c.Status(http.StatusAccepted)
		return
	}

	b, _ := json.Marshal(resp)
	select {
	case ch <- string(b):
	default:
	}
	c.Status(http.StatusAccepted)
}

// HandleMCP - POST /mcp，兼容直接 HTTP 调用（Claude Code 等）
func HandleMCP(c *gin.Context) {
	if !validateMCPKey(c) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "MCP API key required"})
		return
	}

	var req MCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp := processRequest(req)
	c.JSON(http.StatusOK, resp)
}
