// factory.go
package telegramexporter

import (
    "context"

    "go.opentelemetry.io/collector/component"
    "go.opentelemetry.io/collector/consumer"
    "go.opentelemetry.io/collector/exporter"
    "go.opentelemetry.io/collector/exporter/exporterhelper"
)

var componentType = component.MustNewType("telegram")

// NewFactory creates a factory for Telegram exporter
func NewFactory() exporter.Factory {
    return exporter.NewFactory(
        componentType,
        createDefaultConfig,
        exporter.WithTraces(createTraces, component.StabilityLevelBeta),
        exporter.WithMetrics(createMetrics, component.StabilityLevelBeta),
        exporter.WithLogs(createLogs, component.StabilityLevelBeta),
    )
}

func createDefaultConfig() component.Config {
    return &Config{
        Enabled:          false,  // Changed to false by default
        MaxMessageLength: 4096,
        MessageTemplate: "{{.ResourceAttributes}}\n{{.Name}}: {{.Value}}",
    }
}

func createTraces(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Traces, error) {
    cfg := config.(*Config)
    exp, err := newTelegramExporter(set.TelemetrySettings.Logger, cfg)
    if err != nil {
        return nil, err
    }

    return exporterhelper.NewTracesExporter(
        ctx,
        set,
        config,
        exp.pushTraces,
        exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
        exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
    )
}

func createMetrics(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Metrics, error) {
    cfg := config.(*Config)
    exp, err := newTelegramExporter(set.TelemetrySettings.Logger, cfg)
    if err != nil {
        return nil, err
    }

    return exporterhelper.NewMetricsExporter(
        ctx,
        set,
        config,
        exp.pushMetrics,
        exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
        exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
    )
}

func createLogs(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Logs, error) {
    cfg := config.(*Config)
    exp, err := newTelegramExporter(set.TelemetrySettings.Logger, cfg)
    if err != nil {
        return nil, err
    }

    return exporterhelper.NewLogsExporter(
        ctx,
        set,
        config,
        exp.pushLogs,
        exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
        exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
    )
}