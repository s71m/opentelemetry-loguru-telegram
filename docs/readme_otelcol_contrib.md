#### 1) install go https://go.dev/doc/install   
```bash
wget https://go.dev/dl/go1.23.2.linux-amd64.tar.gz   
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz  
export PATH=$PATH:/usr/local/go/bin
```

#### 2) install OpenTelemetry Collector builder https://opentelemetry.io/docs/collector/custom-collector/
```bash 
curl --proto '=https' --tlsv1.2 -fL -o ocb \
https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/cmd%2Fbuilder%2Fv0.111.0/ocb_0.111.0_linux_amd64
chmod +x ocb
```
#### 3) create <work_dir> name it otelcol-contrib, copy here builder-config.yaml
```bash 
mkdir otelcol-contrib && cd otelcol-contrib
cp <your_folder>/builder-config.yaml otelcol-contrib/
```
#### 4) create folder, copy project files: go.mod and other *.go files
```bash 
mkdir -p otelcol-contrib/exporter/telegramexporter
cp <your_folder>/exporter/telegramexporter/* <work_dir>/otelcol-contrib/exporter/telegramexporter
```

#### 5) build custom otelcol
```bash 
./ocb --config builder-config.yaml 
```
#### 6) launch custom otelcol 
```bash 
go run ./otelcol-dev --config /etc/otelcol-contrib/config.yaml
```
#### 7) for debug mode and viewing all data that is being processed by otelcol-dev, uncomment section:
```yaml
  debug:  # Use the debug exporter for detailed logging
    verbosity: detailed

service:
  pipelines:
    metrics:

    logs/otlp:
      receivers: [otlp]
      processors: [transform, batch, resourcedetection]
      exporters: [clickhouse, telegram, debug] # add debug to exporters for desired pipelines

    logs/syslog:
      receivers: [syslog]
      processors: [transform, batch, resourcedetection, resource/syslog]
      exporters: [clickhouse, telegram, debug] # add debug to exporters for desired pipelines
```
#### 8) also you can check journal and send test message via syslog
```bash 
journalctl -u otelcol-dev.service -n 100
logger -p local0.err "Error message 1"
logger -p local0.crit "Critical message 1"
logger -p local0.info "Info message 1"
logger -p user.err "This is a test error message 2"
logger -p alert "This is an alert message"
```
#### 7) now we can create debian package, copy build-otelcol-dev.sh to <work_dir>
```bash 
cd <work_dir>
bash ./build-otelcol-dev.sh
```
#### 8) Copy to another machine with Ubuntu
```bash 
sudo dpkg -i otelcol-dev_1.0.0.deb
sudo apt-get install -f  # If there are any missing dependencies
```
#### 9) Change credentials. I can't figured it out, why EXPORT CLICKHOUSE_USER is not working, so have to pass environments directly to systemd service
```bash 
nano /etc/otelcol-contrib/secrets.env
```
#### 10) There can be rsyslog or syslog-ng
##### For rsyslog add this line to the end 
```bash
sudo nano /etc/rsyslog.conf
```
```
*.* @127.0.0.1:54525;RSYSLOG_SyslogProtocol23Format
```
##### For syslog-ng add this
```bash
sudo nano /etc/syslog-ng/syslog-ng.conf
```
destination d_otel {
    network(
        "127.0.0.1"
        port(54525)
        transport("tcp")
        flags(syslog-protocol)
        disk-buffer(
            mem-buf-size(1M)
            disk-buf-size(1048576)
            reliable(yes)
        )
        log-fifo-size(100)
        time-reopen(60)
    );
};


log {
    source(s_src);
    destination(d_otel);
};
```
before this line
```
@include "/etc/syslog-ng/conf.d/*.conf"
```

### Sending to telegram

1. First Priority Rule:
If "channel" logattribute exists AND severity is in ["WARNING", "ERROR", "CRITICAL"], message goes to ERROR channel regardless of channel match

2. Second Priority Rule:
If "channel" logattributeattribute exists AND matches a configured channel, message goes to that specific channel

3. Third Priority Rule:
If "channel" logattribute exists but doesn't match any configured channel AND severity is in ["DEBUG", "INFO", "SUCCESS"], message goes to GENERAL channel