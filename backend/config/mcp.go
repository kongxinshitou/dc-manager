package config

import "os"

// MCPAPIKey controls access to MCP endpoints.
// If empty, MCP endpoints are open (backward compatible).
// If set, requests must include the key via Authorization header or api_key query param.
var MCPAPIKey = os.Getenv("MCP_API_KEY")
