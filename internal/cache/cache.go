package cache

import (
	"sync"

	toolspec "github.com/hydrocode-de/tool-spec-go"
)

type Cache struct {
	mu          sync.RWMutex
	images      map[string]toolspec.SpecFile
	tools       map[string]toolspec.ToolSpec
	Initialised bool
}

func (c *Cache) GetToolSpec(key string) (*toolspec.ToolSpec, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	spec, ok := c.tools[key]
	return &spec, ok
}

func (c *Cache) SetToolSpec(key string, spec *toolspec.ToolSpec) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tools[key] = *spec
}

func (c *Cache) ListToolSpecs() []toolspec.ToolSpec {
	c.mu.RLock()
	defer c.mu.RUnlock()

	specs := make([]toolspec.ToolSpec, 0)
	for _, spec := range c.tools {
		specs = append(specs, spec)
	}
	return specs
}

func (c *Cache) GetImageSpec(key string) (*toolspec.SpecFile, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	spec, ok := c.images[key]
	return &spec, ok
}

func (c *Cache) SetImageSpec(key string, spec toolspec.SpecFile) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.images[key] = spec
}

func (c *Cache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tools = make(map[string]toolspec.ToolSpec)
	c.images = make(map[string]toolspec.SpecFile)
	c.Initialised = false
}

func (c *Cache) IsInitialised() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.Initialised
}

func (c *Cache) SetInitialised(initialised bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Initialised = initialised
}
