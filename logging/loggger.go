package logging

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	oteltrc "go.opentelemetry.io/otel/trace"
)

// SetupLogging sets up logging for the application this method should be used when bootstrapping the application
func SetupLogging() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.With().Caller().Logger()
}

// TraceLogger returns a logger with trace information this method should be used for logging in the application
func TraceLogger(ctx context.Context) *zerolog.Logger {
	logger := log.Logger.With()

	if span := oteltrc.SpanFromContext(ctx); span != nil {
		spanContext := span.SpanContext()
		if spanContext.HasSpanID() {
			logger = logger.Str("logging.googleapis.com/spanId", spanContext.SpanID().String())
		}
		if spanContext.HasTraceID() {
			traceID := fmt.Sprintf("project/trace/%s", span.SpanContext().TraceID().String())
			logger = logger.
				Str("logging.googleapis.com/trace", traceID).
				Bool("logging.googleapis.com/trace_sampled", spanContext.IsSampled())
		}
	}
	parentLogger := logger.Logger()
	return &parentLogger
}
