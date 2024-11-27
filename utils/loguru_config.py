import sys
from loguru import logger
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
from typing import Optional


def create_logger(
        service_name: str,
        otlp_endpoint: str = "localhost:4317",
        app_name: str = "app",
        log_level: str = "INFO"
) -> None:
    """Configure Loguru logger with console and OTLP output.

    Args:
        service_name: Name of the service
        otlp_endpoint: Endpoint for OTLP exporter
        app_name: Name of the application for logs
        log_level: Minimum log level to capture
    """
    # Configure log format
    log_format = (
        "<green>{time:YYYY-MM-DD HH:mm:ss.SSS}</green> | "
        "<level>{level: <8}</level> | "
        "<cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> | "
        "{extra[appname]} | "
        "{message}"
    )

    # Reset and configure logger
    logger.remove()

    # Add console handler
    logger.add(
        sys.stdout,
        format=log_format,
        level=log_level,
        enqueue=True,
        colorize=True
    )

    # Add OTLP handler
    from utils.loguru_otlp_handler import OTLPHandler
    otlp_handler = OTLPHandler(
        service_name=service_name,
        exporter=OTLPLogExporter(
            endpoint=otlp_endpoint,
            insecure=True
        )
    )
    logger.add(
        otlp_handler.sink,
        level=log_level
    )

    # Add default context
    logger.configure(
        extra={"appname": app_name}
    )

