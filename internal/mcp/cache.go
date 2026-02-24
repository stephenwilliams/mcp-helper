package mcp

import (
	"sync"
	"time"
)

const DefaultCacheTTL = 1 * time.Hour

// Cache stores discovered tools with TTL
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	tools     []Tool
	timestamp time.Time
}

// NewCache creates a new cache with the specified TTL
func NewCache(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	return &Cache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves cached tools for a server, returns nil if expired or not found
// Key format: "scope:serverName" (e.g., "user:github")
func (c *Cache) Get(scope, serverName string) []Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := scope + ":" + serverName
	entry, ok := c.entries[key]
	if !ok {
		return nil
	}

	// Check if expired
	if time.Since(entry.timestamp) > c.ttl {
		return nil
	}

	// Return a copy to prevent external modification
	tools := make([]Tool, len(entry.tools))
	copy(tools, entry.tools)
	return tools
}

// Set stores tools for a server
// Key format: "scope:serverName" (e.g., "user:github")
func (c *Cache) Set(scope, serverName string, tools []Tool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := scope + ":" + serverName
	// Store a copy to prevent external modification
	toolsCopy := make([]Tool, len(tools))
	copy(toolsCopy, tools)

	c.entries[key] = &cacheEntry{
		tools:     toolsCopy,
		timestamp: time.Now(),
	}
}

// Clear removes all cached entries
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

// ClearServer removes cache for all scopes of a specific server
func (c *Cache) ClearServer(serverName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove all entries matching the server name across all scopes
	for key := range c.entries {
		// Key format is "scope:serverName", check if it ends with the server name
		if len(key) > len(serverName) && key[len(key)-len(serverName):] == serverName {
			// Verify it's actually the server name (not just a suffix match)
			if key[len(key)-len(serverName)-1] == ':' {
				delete(c.entries, key)
			}
		}
	}
}
