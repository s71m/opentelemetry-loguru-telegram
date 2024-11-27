from loguru import logger

from utils.loguru_tracer_config import create_tracer

# Initialize tracer
tracer = create_tracer(
    service_name="demo-service",
    otlp_endpoint="localhost:4317"
)


class Analyzer:
    @tracer.trace_span()
    def method1(self):
        logger.info("Processing in method1")
        self.method2()

    @tracer.trace_span()
    def method2(self):
        with tracer.trace_context("data_transform"):
            logger.info("Transforming data")
            self.method3()

    @tracer.trace_span()
    def method3(self):
        logger.info("Processing in method3")
        raise ValueError("Sample error in method3")

    @tracer.trace_span()
    def perform_analysis(self, raise_error=False):
        logger.info("Starting analysis")
        try:
            with tracer.trace_context("analysis_context"):
                self.method1()
        except Exception as e:
            logger.exception(f"Analysis failed: {e}")
            raise


def main():
    logger.info("Starting application")

    with tracer.trace_context("application"):
        analyzer = Analyzer()

        try:
            # First run - successful
            with tracer.trace_context("analysis_run_1"):
                analyzer.perform_analysis()
        except Exception:
            pass

        try:
            # Second run - with exception
            with tracer.trace_context("analysis_run_2"):
                analyzer.perform_analysis(raise_error=True)
        except Exception:
            logger.error("Analysis run 2 failed")


if __name__ == "__main__":
    main()