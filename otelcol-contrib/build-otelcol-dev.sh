#!/bin/bash

# Set version
VERSION="1.0.0"
PACKAGE_NAME="otelcol-dev"

# Create directory structure
mkdir -p ${PACKAGE_NAME}_${VERSION}/DEBIAN
mkdir -p ${PACKAGE_NAME}_${VERSION}/usr/local/bin
mkdir -p ${PACKAGE_NAME}_${VERSION}/etc/otelcol-contrib
mkdir -p ${PACKAGE_NAME}_${VERSION}/lib/systemd/system

# Copy the binary
cp ./otelcol-dev/otelcol-dev ${PACKAGE_NAME}_${VERSION}/usr/local/bin/

chmod 755 ${PACKAGE_NAME}_${VERSION}/usr/local/bin/otelcol-dev

# Create control file
cat > ${PACKAGE_NAME}_${VERSION}/DEBIAN/control << EOL
Package: otelcol-dev
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: amd64
Maintainer: stim <your.email@example.com>
Description: Custom OpenTelemetry Collector
 A custom build of the OpenTelemetry Collector
 with additional exporters and processors.
EOL

# Create postinst script
cat > ${PACKAGE_NAME}_${VERSION}/DEBIAN/postinst << EOL
#!/bin/bash

# Create system user and group for OpenTelemetry Collector
useradd --system \
    --no-create-home \
    --shell /sbin/nologin \
    otelcol-contrib || true  # || true prevents failure if user already exists

# Add the user to the systemd-journal group
usermod -a -G systemd-journal otelcol-contrib

# Set proper permissions for the config directory
chown -R otelcol-contrib:otelcol-contrib /etc/otelcol-contrib
chmod 750 /etc/otelcol-contrib

# Set proper permissions for the secrets file
chmod 640 /etc/otelcol-contrib/secrets.env
chown otelcol-contrib:otelcol-contrib /etc/otelcol-contrib/secrets.env

# Set proper permissions for the binary
chmod 755 /usr/local/bin/otelcol-dev
chown otelcol-contrib:otelcol-contrib /usr/local/bin/otelcol-dev

# Reload systemd and start service
systemctl daemon-reload
systemctl enable otelcol-dev
systemctl start otelcol-dev
EOL

# Make postinst executable
chmod 755 ${PACKAGE_NAME}_${VERSION}/DEBIAN/postinst

# Create secrets.env for otelcol
cat > ${PACKAGE_NAME}_${VERSION}/etc/otelcol-contrib/secrets.env << EOL
CLICKHOUSE_USERNAME=default
CLICKHOUSE_PASSWORD=password

TELEGRAM_BOT_TOKEN=token
TELEGRAM_CHAT_ID=chat_id
EOL

# Create systemd service file
cat > ${PACKAGE_NAME}_${VERSION}/lib/systemd/system/otelcol-dev.service << EOL
[Unit]
Description=OpenTelemetry Collector Contrib
After=network-online.target clickhouse-server.service
Wants=network-online.target clickhouse-server.service
PartOf=clickhouse-server.service

[Service]
EnvironmentFile=/etc/otelcol-contrib/secrets.env
ExecStart=/usr/local/bin/otelcol-dev --config /etc/otelcol-contrib/config.yaml $OTELCOL_OPTIONS
KillMode=mixed
Type=simple
TimeoutSec=30
Restart=on-failure
RestartSec=5
User=otelcol-contrib
Group=otelcol-contrib
SyslogIdentifier=otelcol-contrib

SupplementaryGroups=systemd-journal

[Install]
WantedBy=multi-user.target
EOL

# Create default config file
cat > ${PACKAGE_NAME}_${VERSION}/etc/otelcol-contrib/config.yaml << EOL
receivers:
  hostmetrics:
    collection_interval: 60s
    scrapers:
      cpu: {}
      memory: {}
      disk: {}
      filesystem: {}
      network: {}
      load: {}

  syslog:
    udp:
      listen_address: 0.0.0.0:54525
    protocol: rfc5424
    operators:
      - type: move
        from: attributes.message
        to: body

  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317 #change to localhost

processors:

  transform:
    log_statements:
      - context: log

        statements:
          # --- DEBUG ---
          - set(severity_text, "DEBUG") where severity_text == "trace"
          - set(severity_text, "DEBUG") where severity_text == "debug"
          - set(severity_text, "DEBUG") where severity_text == "DEBUG"

          # --- INFO ---
          - set(severity_text, "INFO") where severity_text == "info"
          - set(severity_text, "INFO") where severity_text == "INFO"

          # --- WARNING ---
          - set(severity_text, "WARNING") where severity_text == "warn"
          - set(severity_text, "WARNING") where severity_text == "WARN"

          # --- ERROR ---
          - set(severity_text, "ERROR") where severity_text == "err"
          - set(severity_text, "ERROR") where severity_text == "error"

          # --- CRITICAL ---
          - set(severity_text, "CRITICAL") where severity_text == "crit"
          - set(severity_text, "CRITICAL") where severity_text == "fatal"
          - set(severity_text, "CRITICAL") where severity_text == "FATAL"

          # --- Default Mapping (CRITICAL), in case new severity_text create new statements ---
          - set(severity_text, "CRITICAL") where severity_text == ""

  batch:
    timeout: 10s
    send_batch_size: 10000

  resourcedetection:
    detectors: [system]
    system:
      resource_attributes:
        host.name:
          enabled: true
        os.type:
          enabled: true
        os.description:
          enabled: false

  resource/hostmetrics:
    attributes:
      - key: service.name
        value: "hostmetrics"
        action: upsert

  resource/syslog:
    attributes:
      - key: service.name
        value: "syslog"
        action: upsert

exporters:
  clickhouse:
    endpoint: tcp://localhost:9000
    database: otel
    username: "\${CLICKHOUSE_USERNAME}"
    password: "\${CLICKHOUSE_PASSWORD}"
    ttl: 72h
    compress: lz4
    timeout: 10s
    create_schema: true
    logs_table_name: otel_logs
    traces_table_name: otel_traces
    sending_queue:
      enabled: true
      num_consumers: 2
      queue_size: 1000
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s

  telegram:
    enabled: true
    bot_token: "\${TELEGRAM_BOT_TOKEN}"
    chat_id: "\${TELEGRAM_CHAT_ID}"
    message_template: "{{.ResourceAttributes}}\n{{.Name}}: {{.Value}}"
    max_message_length: 4096
    batch_enabled: true    # Enable/disable batching functionality
    batch_timeout: 3s       # Time to wait before sending a batch
    batch_size: 2           # Number of messages to accumulate before sending
    # channels section to support message routing by severity or channel
    channels:
      - name: "ERROR"
        message_thread_id: 4
        severities: ["WARNING", "ERROR", "CRITICAL"]
      - name: "INFO"
        message_thread_id: -1
        severities: ["DEBUG", "INFO", "SUCCESS"]
      - name: "TRADE"
        message_thread_id: 2

  #debug:  # Use the debug exporter for detailed logging
  #  verbosity: detailed

service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [batch, resourcedetection, resource/hostmetrics]
      exporters: [clickhouse]

    # Separate log pipelines for each receiver
    logs/otlp:
      receivers: [otlp]
      processors: [transform, batch, resourcedetection]
      exporters: [clickhouse, telegram]

    logs/syslog:
      receivers: [syslog]
      processors: [transform, batch, resourcedetection, resource/syslog]
      exporters: [clickhouse, telegram]

    traces:
      receivers: [otlp]
      processors: [batch, resourcedetection]
      exporters: [clickhouse]
EOL

# Set appropriate permissions
chmod 644 ${PACKAGE_NAME}_${VERSION}/lib/systemd/system/otelcol-dev.service
chmod 644 ${PACKAGE_NAME}_${VERSION}/etc/otelcol-contrib/config.yaml

# Build the package
dpkg-deb --build ${PACKAGE_NAME}_${VERSION}