import sys
import functools
from contextlib import contextmanager
from loguru import logger
from opentelemetry import trace
from opentelemetry.trace import format_trace_id, format_span_id
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter

class OpenTelemetryUtil:
    def __init__(
        self,
        service_name: str,
        otlp_endpoint: str = "localhost:4317",
        app_name: str = "app",
        log_level: str = "INFO"
    ):
        """Initialize OpenTelemetry with tracing and logging.

        Args:
            service_name: Name of the service for tracing
            otlp_endpoint: Endpoint for OTLP exporter
            app_name: Name of the application for logs
            log_level: Minimum log level to capture
        """
        self.service_name = service_name
        self.otlp_endpoint = otlp_endpoint
        self.app_name = app_name
        self.log_level = log_level
        self._setup_tracing()
        self._setup_logging()

    def _setup_tracing(self) -> None:
        """Configure OpenTelemetry tracing with OTLP exporter."""
        resource = Resource.create({"service.name": self.service_name})
        tracer_provider = TracerProvider(resource=resource)
        otlp_exporter = OTLPSpanExporter(
            endpoint=self.otlp_endpoint,
            insecure=True
        )
        span_processor = BatchSpanProcessor(otlp_exporter)
        tracer_provider.add_span_processor(span_processor)
        trace.set_tracer_provider(tracer_provider)
        self.tracer = trace.get_tracer(__name__)

    def _setup_logging(self) -> None:
        """Configure Loguru logging with OTLP export."""
        def trace_context_processor(record):
            """Add trace context to log record."""
            span = trace.get_current_span()
            ctx = span.get_span_context()
            if ctx.is_valid:
                record["extra"].update({
                    "trace_id": format_trace_id(ctx.trace_id),
                    "span_id": format_span_id(ctx.span_id),
                })
            else:
                record["extra"].update({
                    "trace_id": "0" * 32,
                    "span_id": "0" * 16,
                })
            record["extra"]["appname"] = self.app_name

        # Configure log format
        log_format = (
            "<green>{time:YYYY-MM-DD HH:mm:ss.SSS}</green> | "
            "<level>{level: <8}</level> | "
            "<cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> | "
            "{extra[appname]} | "
            "<magenta>trace_id={extra[trace_id]}</magenta> "
            "<magenta>span_id={extra[span_id]}</magenta> | "
            "{message}"
        )

        # Reset and configure logger
        logger.remove()
        logger.configure(patcher=trace_context_processor)

        # Add console handler
        logger.add(
            sys.stdout,
            format=log_format,
            level=self.log_level,
            enqueue=True,
            colorize=True
        )

        # Add OTLP handler
        from utils.loguru_otlp_handler import OTLPHandler
        otlp_handler = OTLPHandler(
            service_name=self.service_name,
            exporter=OTLPLogExporter(
                endpoint=self.otlp_endpoint,
                insecure=True
            )
        )
        logger.add(
            otlp_handler.sink,
            level=self.log_level
        )


    def trace_span(self, name: str = None):
        """Decorator to trace a function.

        Args:
            name: Optional name for the span. Defaults to function name.
        """
        def decorator(func):
            @functools.wraps(func)
            def wrapper(*args, **kwargs):
                span_name = name or func.__name__
                with self.tracer.start_as_current_span(span_name) as span:
                    return func(*args, **kwargs)
            return wrapper
        return decorator

    @contextmanager
    def trace_context(self, name: str):
        """Context manager for creating a trace span.

        Args:
            name: Name for the span
        """
        with self.tracer.start_as_current_span(name) as span:
            yield span

def create_tracer(
    service_name: str,
    otlp_endpoint: str = "localhost:4317",
    app_name: str = None,
    log_level: str = "INFO"
) -> OpenTelemetryUtil:
    """Create an OpenTelemetryUtil instance with the given configuration.

    Args:
        service_name: Name of the service for tracing
        otlp_endpoint: Endpoint for OTLP exporter
        app_name: Name of the application for logs (defaults to service_name)
        log_level: Minimum log level to capture

    Returns:
        Configured OpenTelemetryUtil instance
    """
    if app_name is None:
        app_name = service_name

    return OpenTelemetryUtil(
        service_name=service_name,
        otlp_endpoint=otlp_endpoint,
        app_name=app_name,
        log_level=log_level
    )