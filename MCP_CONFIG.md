# MCP 配置说明

## MCP Endpoint

```
http://localhost:8080/mcp
```

协议：JSON-RPC 2.0 over HTTP POST

## 支持的工具

| 工具名 | 说明 |
|--------|------|
| `query_devices` | 查询设备台账，支持机房/机柜/品牌/IP/负责人等过滤 |
| `add_device` | 新增设备台账记录 |
| `delete_device` | 删除设备台账记录（需提供id） |
| `query_inspections` | 查询巡检记录，支持多字段过滤 |
| `add_inspection` | 新增巡检记录 |
| `delete_inspection` | 删除巡检记录（需提供id） |
| `get_issue_status` | 获取问题状态统计（各机房/等级/状态分布） |

## Claude Desktop 配置示例

在 `claude_desktop_config.json` 中添加：

```json
{
  "mcpServers": {
    "dc-manager": {
      "command": "curl",
      "args": ["-s", "-X", "POST", "http://localhost:8080/mcp"]
    }
  }
}
```

或使用支持 HTTP MCP 的客户端直接连接 `http://localhost:8080/mcp`。

## 调用示例

### 查询某机房的设备
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "query_devices",
    "arguments": {
      "datacenter": "总部大楼",
      "page_size": 10
    }
  }
}
```

### 新增巡检记录
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "add_inspection",
    "arguments": {
      "datacenter": "IDC1-1",
      "cabinet": "A-01",
      "inspector": "张三",
      "issue": "服务器风扇异响",
      "severity": "一般",
      "status": "待处理"
    }
  }
}
```

### 查询问题状态统计
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_issue_status",
    "arguments": {}
  }
}
```
