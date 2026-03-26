package config

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/log/global"
	meter "go.opentelemetry.io/otel/metric"
	otelLog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type ApplicationMetrics struct {
	httpRequestsReceivedTotal  meter.Int64Counter
	httpRequestDurationSeconds meter.Float64Histogram
}

const (
	ServiceName      = "spenn"
	ServiceNamespace = "spenn"
)

func ConfigureOpenTelemetry(
	ctx context.Context,
	log *logrus.Logger,
) (*ApplicationMetrics, func(context.Context) error, error) {
	res, err := resource.New(
		ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String(ServiceName)),
		resource.WithAttributes(semconv.ServiceNamespaceKey.String(ServiceNamespace)),
		resource.WithSchemaURL(semconv.SchemaURL),
	)
	if err != nil {
		return nil, nil, err
	}

	loggerProvider, err := configureLogs(ctx, res, log)
	if err != nil {
		return nil, nil, err
	}

	applicationMetrics, err := configureMetrics(res)
	if err != nil {
		_ = loggerProvider.Shutdown(ctx)

		return nil, nil, err
	}

	return applicationMetrics, loggerProvider.Shutdown, nil
}

func configureLogs(
	ctx context.Context,
	resource *resource.Resource,
	log *logrus.Logger,
) (*otelLog.LoggerProvider, error) {
	logExporter, err := otlploggrpc.New(
		ctx,
	)
	if err != nil {
		return nil, err
	}

	processor := otelLog.NewBatchProcessor(logExporter)
	loggerProvider := otelLog.NewLoggerProvider(
		otelLog.WithResource(resource),
		otelLog.WithProcessor(processor),
	)

	global.SetLoggerProvider(loggerProvider)

	hook := otellogrus.NewHook(
		ServiceNamespace+"/"+ServiceName,
		otellogrus.WithLoggerProvider(loggerProvider),
	)

	log.AddHook(hook)

	return loggerProvider, nil
}

func configureMetrics(resource *resource.Resource) (*ApplicationMetrics, error) {
	metricExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metricExporter),
		metric.WithResource(resource),
	)
	otel.SetMeterProvider(meterProvider)

	metrics := meterProvider.Meter(ServiceName)

	httpRequestsReceivedTotal, err := metrics.Int64Counter(
		"http_requests_received_total",
		meter.WithDescription("Total number of HTTP requests received"),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create counter: %w", err)
	}

	httpRequestDurationSeconds, err := metrics.Float64Histogram(
		"http_request_duration_seconds",
		meter.WithDescription("The duration of HTTP requests processed by Gin, in seconds."),
		meter.WithExplicitBucketBoundaries(
			0.01,
			0.02,
			0.05,
			0.1,
			0.2,
			0.5,
			1,
			2,
			5,
			10,
			20,
			60,
			120,
			300,
			600,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create histogram: %w", err)
	}

	return &ApplicationMetrics{
		httpRequestsReceivedTotal:  httpRequestsReceivedTotal,
		httpRequestDurationSeconds: httpRequestDurationSeconds,
	}, nil
}

func MetricsMiddleware(conf *Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		t := time.Now()

		ctx.Next()

		latency := time.Since(t)
		statusCode := ctx.Writer.Status()
		method := ctx.Request.Method
		endpoint := ctx.Request.URL.Path

		meterAttributes := []attribute.KeyValue{
			attribute.Key("code").Int(statusCode),
			attribute.Key("method").String(method),
			attribute.Key("endpoint").String(endpoint),
		}

		if endpoint == "/metrics" {
			return
		}

		conf.ApplicationMetrics.httpRequestDurationSeconds.Record(
			ctx.Request.Context(),
			latency.Seconds(),
			meter.WithAttributes(meterAttributes...),
		)

		conf.ApplicationMetrics.httpRequestsReceivedTotal.Add(
			ctx.Request.Context(),
			1,
			meter.WithAttributes(meterAttributes...),
		)
	}
}

func LogrusMiddleware(log *logrus.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		ctx.Next()

		latency := time.Since(start)

		// Skip logging for /metrics endpoint to reduce noise in logs
		if ctx.Request.URL.Path != "/metrics" {
			log.WithFields(logrus.Fields{
				"status":    ctx.Writer.Status(),
				"method":    ctx.Request.Method,
				"path":      ctx.Request.URL.Path,
				"ip":        ctx.ClientIP(),
				"latency":   latency,
				"userAgent": ctx.Request.UserAgent(),
			}).Info("request completed")
		}
	}
}
