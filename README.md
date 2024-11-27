
For enabling telegram exporter need to build custom otelcol-contrib, how described here:

After that add this section to etc/otelcol-contrib/config.yaml 

```yaml
exporters:
    telegram:  
        enabled: true  
        bot_token: "${TELEGRAM_BOT_TOKEN}"  
        chat_id: "${TELEGRAM_CHAT_ID}"  
        max_message_length: 4096  
        batch_enabled: true    # Enable/disable batching functionality  
        batch_timeout: 5s       # Time to wait before sending a batch  
        batch_size: 10           # Number of messages to accumulate before sending  
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
```
And example of tracing and logging in demo files:
 [demo_tracer.py](demo_tracer.py) [demo_logger.py](demo_logger.py)