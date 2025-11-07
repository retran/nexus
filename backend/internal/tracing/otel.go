// Copyright 2025 Andrew Vasilyevpackage tracing

// SPDX-License-Identifier: Apache-2.0

// Package tracing provides OpenTelemetry tracing setup for Nexus services.
package tracing

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// InitTracerProvider initializes the OpenTelemetry tracer provider
// and returns a shutdown function that should be called with defer.
//
// Environment variables:
//   - OTEL_EXPORTER_OTLP_ENDPOINT: OTLP collector endpoint (e.g., "alloy:4317")
//   - SERVICE_NAME: Name of the service (required)
//
// Example:
//
//	shutdown, err := tracing.InitTracerProvider(ctx)
//	if err != nil {
//	    log.Fatalf("Failed to initialize tracer: %v", err)
//	}
//	defer shutdown(ctx)
func InitTracerProvider(ctx context.Context) (func(context.Context) error, error) {
	// Get OTLP endpoint from environment
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		log.Println("OTEL_EXPORTER_OTLP_ENDPOINT not set, tracing disabled")
		return func(context.Context) error { return nil }, nil
	}

	// Get service name from environment
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "nexus-service"
	}

	// Create OTLP trace exporter
	traceExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(os.Getenv("SERVICE_VERSION")),
			semconv.DeploymentEnvironmentKey.String("dev"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sample all traces in dev
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for trace context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Printf("OpenTelemetry tracer initialized for service '%s' (endpoint: %s)", serviceName, endpoint)

	// Return shutdown function
	return tp.Shutdown, nil
}
