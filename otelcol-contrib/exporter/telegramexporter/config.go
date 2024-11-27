// config.go - only new additions
package telegramexporter

import (
    "fmt"
    "time"
    "go.opentelemetry.io/collector/component"
)

// ChannelConfig represents the configuration for a specific logging channel
type ChannelConfig struct {
    Name            string   `mapstructure:"name"`
    MessageThreadID int      `mapstructure:"message_thread_id"`
    Severities      []string `mapstructure:"severities"` // List of severities to route to this channel
}

type Config struct {
    // Existing fields remain unchanged
    Enabled          bool            `mapstructure:"enabled"`
    BotToken         string          `mapstructure:"bot_token"`
    ChatID           string          `mapstructure:"chat_id"`
    MessageTemplate  string          `mapstructure:"message_template,omitempty"`
    MaxMessageLength int             `mapstructure:"max_message_length,omitempty"`
    Channels         []ChannelConfig `mapstructure:"channels"`

    // New batch settings
    BatchEnabled bool          `mapstructure:"batch_enabled"`
    BatchTimeout time.Duration `mapstructure:"batch_timeout"`
    BatchSize    int           `mapstructure:"batch_size"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
    if !cfg.Enabled {
        return nil
    }

    if cfg.BotToken == "" {
        return fmt.Errorf("bot_token cannot be empty when exporter is enabled")
    }
    if cfg.ChatID == "" {
        return fmt.Errorf("chat_id cannot be empty when exporter is enabled")
    }
    if cfg.MaxMessageLength <= 0 {
        cfg.MaxMessageLength = 4096
    }
    if cfg.MessageTemplate == "" {
        cfg.MessageTemplate = "{{.ResourceAttributes}}\n{{.Name}}: {{.Value}}"
    }

    // Validate batch settings
    if cfg.BatchEnabled {
        if cfg.BatchSize <= 0 {
            cfg.BatchSize = 10
        }
        if cfg.BatchTimeout <= 0 {
            cfg.BatchTimeout = 3 * time.Second
        }
    }

    return nil
}