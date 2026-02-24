package mcp

import "encoding/json"

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int        `json:"id,omitempty"` // Pointer to distinguish between 0 and null (notifications)
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// ListToolsResult is the response from tools/list
type ListToolsResult struct {
	Tools      []Tool  `json:"tools"`
	NextCursor *string `json:"nextCursor,omitempty"` // For pagination
}

// InitializeParams for the initialize request
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
}

// ClientInfo contains client metadata
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult is the response from initialize
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      ServerInfoMetadata     `json:"serverInfo"`
}

// ServerInfoMetadata contains server metadata from initialize response
type ServerInfoMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerInfo contains discovered server metadata
type ServerInfo struct {
	Name      string
	Scope     string // "user", "project", "local"
	Transport string // "stdio" or "http"
	Tools     []Tool
	Error     error // Non-nil if connection failed
}

// ServerConfig holds the config needed to connect to a server
type ServerConfig struct {
	Name      string
	Scope     string
	Transport string // "stdio" or "http"
	// Stdio fields
	Command string
	Args    []string
	Env     map[string]string
	// HTTP fields
	URL     string
	Headers map[string]string
}
