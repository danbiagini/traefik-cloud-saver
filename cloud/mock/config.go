package mock

// Config implements cloud.ProviderConfig
type Config struct {
    InitialScale map[string]int32 `json:"initialScale,omitempty"` // Allow pre-configuring service scales
    FailAfter    int             `json:"failAfter,omitempty"`    // Fail operations after N calls
}

func (c *Config) Validate() error {
    return nil // Mock config is always valid
}

func (c *Config) GetType() string {
    return "mock"
} 