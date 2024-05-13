/*
https://github.com/equinix-labs/otel-init-go
Copyright [yyyy] [name of copyright owner]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package otel

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// SimpleCarrier is an abstraction for handling traceparent propagation
// that needs a type that implements the propagation.TextMapCarrier().
// This is the simplest possible implementation that is a little fragile
// but since we're not doing anything else with it, it's fine for this.
type SimpleCarrier map[string]string

// Get implements the otel interface for propagation.
func (otp SimpleCarrier) Get(key string) string {
	return otp[key]
}

// Set implements the otel interface for propagation.
func (otp SimpleCarrier) Set(key, value string) {
	otp[key] = value
}

// Keys implements the otel interface for propagation.
func (otp SimpleCarrier) Keys() []string {
	out := []string{}
	for k := range otp {
		out = append(out, k)
	}
	return out
}

// Clear implements the otel interface for propagation.
func (otp SimpleCarrier) Clear() {
	for k := range otp {
		delete(otp, k)
	}
}

// TraceparentStringFromContext gets the current trace from the context and
// returns a W3C traceparent string. Depends on global OTel TextMapPropagator.
func TraceparentStringFromContext(ctx context.Context) string {
	carrier := SimpleCarrier{}
	prop := otel.GetTextMapPropagator()
	prop.Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

// ContextWithEnvTraceparent is a helper that looks for the the TRACEPARENT
// environment variable and if it's set, it grabs the traceparent and
// adds it to the context it returns. When there is no envvar or it's
// empty, the original context is returned unmodified.
// Depends on global OTel TextMapPropagator.
func ContextWithEnvTraceparent(ctx context.Context) context.Context {
	traceparent := os.Getenv("TRACEPARENT")
	if traceparent != "" {
		return ContextWithTraceparentString(ctx, traceparent)
	}
	return ctx
}

// ContextWithTraceparentString takes a W3C traceparent string, uses the otel
// carrier code to get it into a context it returns ready to go.
// Depends on global OTel TextMapPropagator.
func ContextWithTraceparentString(ctx context.Context, traceparent string) context.Context {
	carrier := SimpleCarrier{}
	carrier.Set("traceparent", traceparent)
	prop := otel.GetTextMapPropagator()
	return prop.Extract(ctx, carrier)
}

// Config holds the typed values of configuration read from the environment.
// It is public mainly to make testing easier and most users should never
// use it directly.
type Config struct {
	Servicename string `json:"service_name"`
	Endpoint    string `json:"endpoint"`
	Insecure    bool   `json:"insecure"`
	Logger      logr.Logger
}

// Init sets up the OpenTelemetry plumbing so it's ready to use.
// It requires a context.Context and returns context and a func() that encapuslates clean shutdown.
func Init(ctx context.Context, c Config) (context.Context, context.CancelFunc, error) {
	if c.Endpoint != "" {
		return c.initTracing(ctx)
	}

	// no configuration, nothing to do, the calling code is inert
	// config is available in the returned context (for test/debug)
	return ctx, func() {}, nil
}

func (c Config) initTracing(ctx context.Context) (context.Context, context.CancelFunc, error) {
	// set the service name that will show up in tracing UIs
	resAttrs := resource.WithAttributes(semconv.ServiceNameKey.String(c.Servicename))
	res, err := resource.New(ctx, resAttrs)
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to create OpenTelemetry service name resource: %w", err)
	}

	retryPolicy := `{
		"methodConfig": [{
			"retryPolicy": {
				"MaxAttempts": 1000,
				"InitialBackoff": ".01s",
				"MaxBackoff": ".01s",
				"BackoffMultiplier": 1.0,
				"RetryableStatusCodes": [ "UNAVAILABLE" ]
			}
		}]
	}`

	grpcOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(c.Endpoint),
		otlptracegrpc.WithDialOption(grpc.WithDefaultServiceConfig(retryPolicy)),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: time.Second * 5,
			MaxInterval:     time.Second * 30,
			MaxElapsedTime:  time.Minute * 5,
		}),
	}
	if c.Insecure {
		grpcOpts = append(grpcOpts, otlptracegrpc.WithInsecure())
	} else {
		creds := credentials.NewClientTLSFromCert(nil, "")
		grpcOpts = append(grpcOpts, otlptracegrpc.WithTLSCredentials(creds))
	}
	// TODO: add TLS client cert auth

	exporter, err := otlptracegrpc.New(context.Background(), grpcOpts...)
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to configure OTLP exporter: %w", err)
	}

	// TODO: more configuration opportunities here
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(prop)

	// inject the tracer into the otel globals, start background goroutines
	otel.SetTracerProvider(tracerProvider)

	// logger
	otel.SetLogger(c.Logger)

	// set a custom error handler so that we can use our own logger
	otel.SetErrorHandler(c)

	// the public function will wrap this in its own shutdown function
	return ctx, func() {
		ctx1, done := context.WithTimeout(context.Background(), 5*time.Second)
		err = tracerProvider.Shutdown(ctx1)
		if err != nil {
			c.Logger.Info("shutdown of OpenTelemetry tracerProvider failed: %s", err)
		}
		done()

		ctx2, done := context.WithTimeout(context.Background(), 5*time.Second)
		err = exporter.Shutdown(ctx2)
		if err != nil {
			c.Logger.Info("shutdown of OpenTelemetry OTLP exporter failed: %s", err)
		}
		done()
	}, nil
}

func (c Config) Handle(err error) {
	if err != nil {
		c.Logger.Info("OpenTelemetry error", "err", err)
	}
}
