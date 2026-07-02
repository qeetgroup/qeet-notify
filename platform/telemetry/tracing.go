package telemetry

// Tracing provides distributed trace propagation for qeet-notify.
//
// Current state: no-op stub. To enable full OpenTelemetry tracing add
// go.opentelemetry.io/otel and go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
// to go.mod, then replace this file with a real OTLP exporter wired to
// config.QeetLogsOTLPEndpoint. The OTLPEndpoint config key already exists.
//
// Pattern when enabled:
//
//	tp, err := tracing.NewProvider(cfg.QeetLogsOTLPEndpoint, "qeet-notify", version.String())
//	otel.SetTracerProvider(tp)
//	defer tp.Shutdown(ctx)

// TracingConfig holds the optional OTLP endpoint for exporting spans.
type TracingConfig struct {
	OTLPEndpoint string // e.g. "localhost:4317" or "" to disable
	ServiceName  string
	Version      string
}

// NewProvider is a no-op placeholder. Replace with otel SDK init when adding tracing.
func NewProvider(cfg TracingConfig) (shutdown func(), err error) {
	// No-op: tracing disabled until go.opentelemetry.io/otel is added to go.mod.
	return func() {}, nil
}
