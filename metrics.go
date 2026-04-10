package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type instruments struct {
	requestDuration  metric.Float64Histogram
	requestsInFlight metric.Int64UpDownCounter
}

// initMeter sets up the OTel MeterProvider with an OTLP HTTP exporter that
// pushes metrics to the endpoint configured via OTEL_EXPORTER_OTLP_ENDPOINT.
// Returns the instruments struct and the provider for graceful shutdown.
func initMeter(ctx context.Context) (*instruments, *sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("create OTLP metric exporter: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)
	meter := provider.Meter("rollouts-demo")

	shutdownOnErr := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(shutdownCtx); err != nil {
			log.Printf("metrics: provider cleanup during init error: %v", err)
		}
	}

	requestDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		shutdownOnErr()
		return nil, nil, fmt.Errorf("create histogram: %w", err)
	}

	requestsInFlight, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of HTTP requests currently being served"),
	)
	if err != nil {
		shutdownOnErr()
		return nil, nil, fmt.Errorf("create updowncounter: %w", err)
	}

	inst := &instruments{
		requestDuration:  requestDuration,
		requestsInFlight: requestsInFlight,
	}

	return inst, provider, nil
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.written {
		return
	}
	r.statusCode = code
	r.written = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.written {
		r.statusCode = http.StatusOK
		r.written = true
	}
	return r.ResponseWriter.Write(b)
}

// Flush proxies to the underlying ResponseWriter if it implements http.Flusher.
// Required because http.FileServer uses Flusher for chunked/range responses.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func metricsMiddleware(inst *instruments, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		path := normalizePath(r.URL.Path)

		inFlightAttrs := metric.WithAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.String("http.route", path),
		)

		inst.requestsInFlight.Add(ctx, 1, inFlightAttrs)
		// Use context.Background() so the decrement is always recorded even
		// when the request context has already been cancelled (client disconnect).
		defer inst.requestsInFlight.Add(context.Background(), -1, inFlightAttrs)

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rec, r)

		attrs := metric.WithAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.String("http.route", path),
			attribute.Int("http.response.status_code", rec.statusCode),
		)
		inst.requestDuration.Record(ctx, float64(time.Since(start).Milliseconds()), attrs)
	})
}

// normalizePath collapses URL paths into a small set of metric labels to
// prevent high-cardinality explosion. All paths except /color are grouped
// under "static" — including 404s from non-existent paths.
// NOTE: update this allowlist when adding new API routes to the mux.
func normalizePath(p string) string {
	if p == "/color" {
		return "/color"
	}
	return "static"
}
