module custom-collector/exporter/telegramexporter

go 1.22.0

require (
    go.opentelemetry.io/collector/component v0.111.0
    go.opentelemetry.io/collector/consumer v0.111.0
    go.opentelemetry.io/collector/exporter v0.111.0
    go.opentelemetry.io/collector/pdata v1.17.0
)

require (
    github.com/gogo/protobuf v1.3.2
    github.com/golang/protobuf v1.5.3
    github.com/json-iterator/go v1.1.12
    go.uber.org/atomic v1.11.0
    go.uber.org/multierr v1.11.0
    google.golang.org/protobuf v1.32.0
)