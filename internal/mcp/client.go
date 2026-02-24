package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const (
	DefaultTimeout  = 10 * time.Second
	ProtocolVersion = "2024-11-05"
	GracePeriod     = 2 * time.Second
)

// Client provides methods to interact with MCP servers
type Client struct {
	timeout time.Duration
	cache   *Cache
}

// NewClient creates a new MCP client
func NewClient(timeout time.Duration, cache *Cache) *Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if cache == nil {
		cache = NewCache(DefaultCacheTTL)
	}
	return &Client{
		timeout: timeout,
		cache:   cache,
	}
}

// ListToolsStdio connects to a stdio MCP server and lists tools
// Performs full MCP handshake: initialize -> initialized notification -> tools/list
func (c *Client) ListToolsStdio(ctx context.Context, command string, args []string, env map[string]string) ([]Tool, error) {
	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Spawn server process
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = buildEnvSlice(env)

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Discard stderr to prevent blocking (could log it instead)
	go io.Copy(io.Discard, stderr)

	// Ensure cleanup
	defer func() {
		c.gracefulShutdown(cmd)
	}()

	// Create JSON encoder/decoder
	encoder := json.NewEncoder(stdin)
	decoder := json.NewDecoder(stdout)

	// Step 1: Send initialize request
	if err := c.initializeServer(encoder, decoder); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	// Step 2: Send initialized notification
	if err := c.sendInitializedNotification(encoder); err != nil {
		return nil, fmt.Errorf("initialized notification failed: %w", err)
	}

	// Step 3: List tools with pagination
	tools, err := c.listTools(encoder, decoder)
	if err != nil {
		return nil, fmt.Errorf("tools/list failed: %w", err)
	}

	return tools, nil
}

// initializeServer sends the initialize request and waits for response
func (c *Client) initializeServer(encoder *json.Encoder, decoder *json.Decoder) error {
	reqID := 1
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      &reqID,
		Method:  "initialize",
		Params: InitializeParams{
			ProtocolVersion: ProtocolVersion,
			Capabilities:    make(map[string]interface{}),
			ClientInfo: ClientInfo{
				Name:    "mcp-helper",
				Version: "1.0.0",
			},
		},
	}

	if err := encoder.Encode(req); err != nil {
		return fmt.Errorf("failed to send initialize: %w", err)
	}

	// Read response
	var resp JSONRPCResponse
	if err := decoder.Decode(&resp); err != nil {
		return fmt.Errorf("failed to read initialize response: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	// Validate response
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	return nil
}

// sendInitializedNotification sends the initialized notification (no response expected)
func (c *Client) sendInitializedNotification(encoder *json.Encoder) error {
	notification := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
		// No ID for notifications
	}

	if err := encoder.Encode(notification); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// listTools calls tools/list with pagination support
func (c *Client) listTools(encoder *json.Encoder, decoder *json.Decoder) ([]Tool, error) {
	var allTools []Tool
	var cursor *string
	reqID := 2 // Start at 2 (1 was used for initialize)

	for {
		params := make(map[string]interface{})
		if cursor != nil {
			params["cursor"] = *cursor
		}

		req := JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      &reqID,
			Method:  "tools/list",
			Params:  params,
		}

		if err := encoder.Encode(req); err != nil {
			return nil, fmt.Errorf("failed to send tools/list: %w", err)
		}

		// Read response
		var resp JSONRPCResponse
		if err := decoder.Decode(&resp); err != nil {
			return nil, fmt.Errorf("failed to read tools/list response: %w", err)
		}

		if resp.Error != nil {
			return nil, fmt.Errorf("tools/list error: %s (code %d)", resp.Error.Message, resp.Error.Code)
		}

		// Parse result
		var result ListToolsResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, fmt.Errorf("failed to parse tools/list result: %w", err)
		}

		allTools = append(allTools, result.Tools...)

		// Check for pagination
		if result.NextCursor == nil {
			break
		}
		cursor = result.NextCursor
		reqID++
	}

	return allTools, nil
}

// gracefulShutdown attempts graceful shutdown with SIGTERM, then SIGKILL after grace period
func (c *Client) gracefulShutdown(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	// Send SIGTERM for graceful shutdown
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited
		return
	}

	// Wait for process to exit or timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited gracefully
		return
	case <-time.After(GracePeriod):
		// Force kill after grace period
		cmd.Process.Kill()
		<-done // Wait for kill to complete
	}
}

// ListToolsHTTP connects to an HTTP MCP server and lists tools
// Performs full MCP handshake: initialize -> initialized notification -> tools/list
func (c *Client) ListToolsHTTP(ctx context.Context, url string, headers map[string]string) ([]Tool, error) {
	if url == "" {
		return nil, fmt.Errorf("HTTP server URL is empty")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: c.timeout,
	}

	// Helper to make HTTP requests with session support
	var sessionID string
	doRequest := func(req JSONRPCRequest) (*http.Response, error) {
		reqBody, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")
		// Add session ID if we have one
		if sessionID != "" {
			httpReq.Header.Set("Mcp-Session-Id", sessionID)
		}
		// Add custom headers (e.g., authorization)
		for k, v := range headers {
			httpReq.Header.Set(k, v)
		}

		return client.Do(httpReq)
	}

	// Step 1: Initialize
	reqID := 1
	initReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      &reqID,
		Method:  "initialize",
		Params: InitializeParams{
			ProtocolVersion: ProtocolVersion,
			Capabilities:    make(map[string]interface{}),
			ClientInfo: ClientInfo{
				Name:    "mcp-helper",
				Version: "1.0.0",
			},
		},
	}

	resp, err := doRequest(initReq)
	if err != nil {
		return nil, fmt.Errorf("initialize request failed: %w", err)
	}

	// Capture session ID from response header
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		sessionID = sid
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read initialize response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("initialize HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var initResp JSONRPCResponse
	if err := json.Unmarshal(body, &initResp); err != nil {
		return nil, fmt.Errorf("failed to decode initialize response: %w", err)
	}

	if initResp.Error != nil {
		return nil, fmt.Errorf("initialize error: %s (code %d)", initResp.Error.Message, initResp.Error.Code)
	}

	// Step 2: Send initialized notification
	initializedReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
		// No ID for notifications
	}

	resp, err = doRequest(initializedReq)
	if err != nil {
		return nil, fmt.Errorf("initialized notification failed: %w", err)
	}
	resp.Body.Close()

	// Step 3: List tools
	reqID = 2
	toolsReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      &reqID,
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	resp, err = doRequest(toolsReq)
	if err != nil {
		return nil, fmt.Errorf("tools/list request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read tools/list response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tools/list HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode tools/list response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s (code %d)", rpcResp.Error.Message, rpcResp.Error.Code)
	}

	var result ListToolsResult
	if err := json.Unmarshal(rpcResp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools result: %w", err)
	}

	return result.Tools, nil
}

// DiscoverTools discovers tools from multiple servers concurrently
// Uses cache when available, returns results for each server
func (c *Client) DiscoverTools(ctx context.Context, servers []ServerConfig, useCache bool) []ServerInfo {
	results := make([]ServerInfo, len(servers))
	var wg sync.WaitGroup

	for i, server := range servers {
		wg.Add(1)
		go func(idx int, srv ServerConfig) {
			defer wg.Done()

			info := ServerInfo{
				Name:      srv.Name,
				Scope:     srv.Scope,
				Transport: srv.Transport,
			}

			// Check cache first if enabled
			if useCache {
				if cached := c.cache.Get(srv.Scope, srv.Name); cached != nil {
					info.Tools = cached
					results[idx] = info
					return
				}
			}

			// Discover tools based on transport type
			var tools []Tool
			var err error

			switch srv.Transport {
			case "http":
				tools, err = c.ListToolsHTTP(ctx, srv.URL, srv.Headers)
			case "stdio":
				tools, err = c.ListToolsStdio(ctx, srv.Command, srv.Args, srv.Env)
			default:
				err = fmt.Errorf("unsupported transport: %s", srv.Transport)
			}

			if err != nil {
				info.Error = err
			} else {
				info.Tools = tools
				// Cache successful results
				c.cache.Set(srv.Scope, srv.Name, tools)
			}

			results[idx] = info
		}(i, server)
	}

	wg.Wait()
	return results
}

// buildEnvSlice builds environment variable slice from map, merging with current env
func buildEnvSlice(envMap map[string]string) []string {
	// Start with current environment
	env := os.Environ()

	// Add/override with provided env vars
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// Helper to read JSON-RPC messages with timeout
type messageReader struct {
	decoder *json.Decoder
	mu      sync.Mutex
}

func newMessageReader(r io.Reader) *messageReader {
	return &messageReader{
		decoder: json.NewDecoder(r),
	}
}

func (mr *messageReader) readMessage(ctx context.Context) (*JSONRPCResponse, error) {
	type result struct {
		resp *JSONRPCResponse
		err  error
	}

	ch := make(chan result, 1)
	go func() {
		mr.mu.Lock()
		defer mr.mu.Unlock()

		var resp JSONRPCResponse
		err := mr.decoder.Decode(&resp)
		ch <- result{&resp, err}
	}()

	select {
	case res := <-ch:
		return res.resp, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
