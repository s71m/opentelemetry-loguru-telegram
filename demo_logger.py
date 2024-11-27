from loguru import logger
from utils.loguru_config import create_logger

# Usage example:
if __name__ == "__main__":
    create_logger(
        service_name="demo-service",
        app_name="demo-app",
        log_level="DEBUG",
    )

    logger.bind(channel="INFO").debug("Debug message to channel INFO")
    logger.bind(channel="INFO").info("Info message to channel INFO")

    # Now you can use logger anywhere in your application
    logger.warning("Warning message")
    logger.error("Error message")
    logger.critical("Critical message")
    #
    try:
        1 / 0
    except Exception as e:
        logger.exception("An error occurred")

