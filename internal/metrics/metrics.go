// Package metrics wires an OpenTelemetry [metric.MeterProvider] onto
// the process.
//
// The template ships with metrics disabled. [Init] returns a no-op
// shutdown func and never touches the OTel global, so the runtime
// footprint is zero until you opt in. otelhttp (wired in
// cmd/server/server.go) still records into whatever provider the OTel
// global exposes, which defaults to a no-op — so instrumentation sites
// stay in place and cost nothing until a real provider is installed
// here.
//
// Turning metrics on is a deliberate per-project choice. The shape of
// the change is intentionally localised:
//
//  1. Pick an exporter package — [prometheus] (pull, via a scrape
//     endpoint), [otlpmetrichttp]/[otlpmetricgrpc] (push, to a
//     collector), [stdoutmetric] (debug), etc. — and `go get` it.
//  2. In the `enabled` branch of [Init], construct the exporter, wrap
//     it in a [sdkmetric.Reader], build a [sdkmetric.MeterProvider],
//     install it via otel.SetMeterProvider, and return the provider's
//     Shutdown method as the ShutdownFunc.
//  3. If the exporter is pull-based (Prometheus), start a second
//     http.Server for the scrape endpoint — see cmd/server/main.go for
//     where that second listener would slot into the existing
//     errgroup.
//
// A typical "enabled" body using the Prometheus exporter looks like:
//
//	exp, err := prometheus.New()
//	if err != nil {
//	    return nil, fmt.Errorf("prometheus exporter: %w", err)
//	}
//	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exp))
//	otel.SetMeterProvider(provider)
//	return provider.Shutdown, nil
//
// [prometheus]: https://pkg.go.dev/go.opentelemetry.io/otel/exporters/prometheus
// [otlpmetrichttp]: https://pkg.go.dev/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp
// [otlpmetricgrpc]: https://pkg.go.dev/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc
// [stdoutmetric]: https://pkg.go.dev/go.opentelemetry.io/otel/exporters/stdout/stdoutmetric
package metrics

import "context"

// ShutdownFunc flushes and tears down a MeterProvider. It is safe to
// call on the disabled path — that returns a no-op. Callers should
// invoke it during graceful shutdown, typically inside the same
// errgroup goroutine that handles http.Server.Shutdown, with a bounded
// context.
type ShutdownFunc func(context.Context) error

// noopShutdown is returned whenever metrics are disabled. It satisfies
// the ShutdownFunc contract so callers can treat the shutdown hook
// uniformly whether the exporter is active or not.
func noopShutdown(context.Context) error { return nil }

// Init installs a MeterProvider on the OTel global when enabled is
// true, and returns a shutdown func that flushes and tears it down.
// When enabled is false, Init is a no-op and returns a no-op shutdown
// so callers can invoke the returned hook unconditionally.
//
// The template ships with the `enabled` branch empty — see the package
// doc comment for the shape of the change when you wire a real
// exporter.
func Init(enabled bool) (ShutdownFunc, error) {
	if !enabled {
		return noopShutdown, nil
	}

	// TODO: construct an exporter, wrap it in a sdkmetric.Reader, build
	// a sdkmetric.MeterProvider, install it via otel.SetMeterProvider,
	// and return provider.Shutdown. See the package doc comment for an
	// example.
	return noopShutdown, nil
}
