package telegramexporter

import (
    "context"
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "text/template"
	"strings"
    "go.uber.org/zap"
    "go.opentelemetry.io/collector/pdata/plog"
    "go.opentelemetry.io/collector/pdata/pmetric"
    "go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type telegramExporter struct {
    // Existing fields
    logger      *zap.Logger
    config      *Config
    client      *http.Client
    msgTemplate *template.Template
    // New field
    batchProc   *batchProcessor
}

func newTelegramExporter(logger *zap.Logger, config *Config) (*telegramExporter, error) {
    tmpl, err := template.New("telegram").Parse(config.MessageTemplate)
    if err != nil {
        return nil, fmt.Errorf("failed to parse message template: %w", err)
    }

    exp := &telegramExporter{
        logger:      logger,
        config:      config,
        client:      &http.Client{},
        msgTemplate: tmpl,
    }

    exp.batchProc = newBatchProcessor(config, exp.sendBatch)
    return exp, nil
}

func (e *telegramExporter) sendBatch(messages []*messageItem) error {
    if len(messages) == 0 {
        return nil
    }

    // All messages in a batch have the same threadID
    threadID := messages[0].threadID
    
    // Join messages with newlines
    var combinedMsg strings.Builder
    for i, msg := range messages {
        if i > 0 {
            combinedMsg.WriteString("\n")
        }
        combinedMsg.WriteString(msg.content)
    }

    return e.sendMessage(combinedMsg.String(), threadID)
}


// sendMessage sends the formatted message to Telegram
func (e *telegramExporter) sendMessage(msg string, messageThreadID int) error {
    url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", e.config.BotToken)
    payload := map[string]interface{}{
        "chat_id":          e.config.ChatID,
        "text":             msg,
        "message_thread_id": messageThreadID,
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal telegram payload: %w", err)
    }

    resp, err := e.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("failed to send telegram message: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("telegram API returned non-200 status code: %d", resp.StatusCode)
    }

    return nil
}

// pushTraces pushes trace data to Telegram
func (e *telegramExporter) pushTraces(_ context.Context, td ptrace.Traces) error {
    msg := fmt.Sprintf("Traces\nResource spans: %d\nSpans: %d",
        td.ResourceSpans().Len(),
        td.SpanCount())

    return e.sendMessage(msg, -1)  // Default to GENERAL for traces
}

// pushMetrics pushes metric data to Telegram
func (e *telegramExporter) pushMetrics(_ context.Context, md pmetric.Metrics) error {
    msg := fmt.Sprintf("Metrics\nResource metrics: %d\nMetrics: %d\nData points: %d",
        md.ResourceMetrics().Len(),
        md.MetricCount(),
        md.DataPointCount())

    return e.sendMessage(msg, -1)  // Default to GENERAL for metrics
}

// pushLogs processes logs and sends messages to Telegram
func (e *telegramExporter) pushLogs(_ context.Context, ld plog.Logs) error {
    resourceLogs := ld.ResourceLogs()
    for i := 0; i < resourceLogs.Len(); i++ {
        resourceLog := resourceLogs.At(i)
        resource := resourceLog.Resource()

        // Extract service.name and host.name from resource attributes
        serviceName := ""
        if serviceNameVal, exists := resource.Attributes().Get("service.name"); exists {
            serviceName = serviceNameVal.AsString()
        }

        hostName := ""
        if hostNameVal, exists := resource.Attributes().Get("host.name"); exists {
            hostName = hostNameVal.AsString()
        }

        logRecords := resourceLog.ScopeLogs().At(0).LogRecords()
        for j := 0; j < logRecords.Len(); j++ {
            logRecord := logRecords.At(j)
            severity := logRecord.SeverityText()

            // Initialize the attributes map
            attributes := make(map[string]interface{})
            logRecord.Attributes().Range(func(k string, v pcommon.Value) bool {
                attributes[k] = v.AsString()
                return true
            })

            // Get the message_thread_id based on "channel" attribute or severity
            messageThreadID := e.getMessageThreadID(severity, attributes)

            // Skip sending if messageThreadID is nil
            if messageThreadID == nil {
                continue
            }

            // Proceed to construct the message and send it
            body := logRecord.Body().AsString()

            // Get the timestamp from the log record and format it to local time with timezone offset
            localTime := logRecord.Timestamp().AsTime().Local().Format("2006-01-02 15:04:05 -0700")

            // Construct the message
            msg := fmt.Sprintf("%s | %s | %s | %s\n%s", localTime, hostName, serviceName, severity, body)

            // Send the message to Telegram
            // if err := e.sendMessage(msg, *messageThreadID); err != nil {
            //     return fmt.Errorf("failed to send log to Telegram: %w", err)
            // }
			    // When ready to send a message, use the batch processor
			if err := e.batchProc.add(msg, *messageThreadID); err != nil {
				return fmt.Errorf("failed to process log message: %w", err)
			}
        }
    }

    return nil
}




func (e *telegramExporter) getMessageThreadID(severity string, logAttributes map[string]interface{}) *int {
    // Helper function to check if severity is in the error category
    isErrorSeverity := func(sev string) bool {
        errorSevs := []string{"WARNING", "ERROR", "CRITICAL"}
        upperSev := strings.ToUpper(sev)
        for _, es := range errorSevs {
            if upperSev == es {
                return true
            }
        }
        return false
    }

    // Helper function to check if severity is in the general category
    isGeneralSeverity := func(sev string) bool {
        generalSevs := []string{"DEBUG", "INFO", "SUCCESS"}
        upperSev := strings.ToUpper(sev)
        for _, gs := range generalSevs {
            if upperSev == gs {
                return true
            }
        }
        return false
    }

    // Get ERROR and GENERAL channel configs
    var errorChannel, generalChannel *ChannelConfig
    for idx := range e.config.Channels {
        channel := &e.config.Channels[idx]
        if channel.Name == "ERROR" {
            errorChannel = channel
        } else if channel.Name == "GENERAL" {
            generalChannel = channel
        }
    }

    // Check if channel attribute exists
    if channelAttr, exists := logAttributes["channel"]; exists {
        channelName, ok := channelAttr.(string)
        if !ok {
            return nil // Invalid channel attribute type
        }

        // First priority rule: If channel attribute exists and severity is in error category,
        // send to ERROR channel regardless of channel match
        if isErrorSeverity(severity) && errorChannel != nil {
            return &errorChannel.MessageThreadID
        }

        // Second priority rule: If channel attribute matches configured channel,
        // send to that channel
        for idx := range e.config.Channels {
            channel := &e.config.Channels[idx]
            if channel.Name == channelName {
                return &channel.MessageThreadID
            }
        }

        // Third priority rule: If channel doesn't match any configured channel
        // and severity is in general category, send to GENERAL channel
        if isGeneralSeverity(severity) && generalChannel != nil {
            return &generalChannel.MessageThreadID
        }

        return nil // Skip if none of the conditions match
    }

    // If no channel attribute, use default severity-based routing
    if isErrorSeverity(severity) && errorChannel != nil {
        return &errorChannel.MessageThreadID
    } else if isGeneralSeverity(severity) && generalChannel != nil {
        return &generalChannel.MessageThreadID
    }

    return nil
}



// Helper function to find a channel configuration by name
func (e *telegramExporter) findChannelConfigByName(name string) *ChannelConfig {
    for idx := range e.config.Channels {
        channel := &e.config.Channels[idx]
        if channel.Name == name {
            return channel
        }
    }
    return nil
}

