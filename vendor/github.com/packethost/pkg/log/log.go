// Copyright 2019 - 2020, Packethost, Inc and contributors
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"os"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/packethost/pkg/log/internal/rollbar"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
)

var (
	// zap.LevelFlag adds an option to the default set of command line flags as part of the flag pacakge.
	logLevel = zap.LevelFlag("log-level", zap.InfoLevel, "Log level. one of ERROR, INFO, or DEBUG")
)

// Logger is a wrapper around zap.SugaredLogger
type Logger struct {
	service string
	s       *zap.SugaredLogger
	cleanup func()
}

func setupConfig(service string) zap.Config {
	var config zap.Config
	if os.Getenv("DEBUG") != "" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	// We expect that errors will already log the stacktrace from pkg/errors functionality as errorVerbose context
	// key
	config.DisableStacktrace = true

	if os.Getenv("LOG_DISCARD_LOGS") != "" {
		config.OutputPaths = nil
		config.ErrorOutputPaths = nil
	}

	config.Level = zap.NewAtomicLevelAt(*logLevel)
	return config
}

func buildConfig(c zap.Config) (*zap.Logger, error) {
	l, err := c.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build logger config")
	}
	return l, nil
}

func configureLogger(l *zap.Logger, service string) (Logger, error) {
	l = l.With(zap.String("service", service))

	rollbarClean := rollbar.Setup(l.Sugar().With("pkg", "log"), service)
	cleanup := func() {
		rollbarClean()
		_ = l.Sync()
	}

	return Logger{service: service, s: l.Sugar(), cleanup: cleanup}.AddCallerSkip(1), nil
}

// Init initializes the logging system and sets the "service" key to the provided argument.
// This func should only be called once and after flag.Parse() has been called otherwise leveled logging will not be configured correctly.
func Init(service string) (Logger, error) {
	config := setupConfig(service)
	l, err := buildConfig(config)
	if err != nil {
		return Logger{}, err
	}

	return configureLogger(l, service)

}

// Test returns a logger that does not log to rollbar and can be used with testing.TB to only log on test failure or run with -v
func Test(t zaptest.TestingT, service string) Logger {
	l := zaptest.NewLogger(t)
	return Logger{service: service, s: l.Sugar(), cleanup: func() { _ = l.Sync() }}.AddCallerSkip(1).Package(t.Name())
}

// Close finishes and flushes up any in-flight logs
func (l Logger) Close() {
	if l.cleanup == nil {
		return
	}
	l.cleanup()
}

// Error is used to log an error, the error will be forwared to rollbar and/or other external services.
// All the values of arg are stringified and concatenated without any strings.
// If no args are provided err.Error() is used as the log message.
func (l Logger) Error(err error, args ...interface{}) {
	rollbar.Notify(err, args)
	if len(args) == 0 {
		args = append(args, err)
	}
	l.s.With("error", err).Error(args...)
}

// Fatal calls Error followed by a panic(err)
func (l Logger) Fatal(err error, args ...interface{}) {
	l.AddCallerSkip(1).Error(err, args...)
	panic(err)
}

// Info is used to log message in production, only simple strings should be given in the args.
// Context should be added as K=V pairs using the `With` method.
// All the values of arg are stringified and concatenated without any strings.
func (l Logger) Info(args ...interface{}) {
	l.s.Info(args...)
}

// Debug is used to log messages in development, not even for lab.
// No one cares what you pass to Debug.
// All the values of arg are stringified and concatenated without any strings.
func (l Logger) Debug(args ...interface{}) {
	l.s.Debug(args...)
}

// With is used to add context to the logger, a new logger copy with the new K=V pairs as context is returned.
func (l Logger) With(args ...interface{}) Logger {
	return Logger{service: l.service, s: l.s.With(args...), cleanup: l.cleanup}
}

// AddCallerSkip increases the number of callers skipped by caller annotation.
// When building wrappers around the Logger, supplying this option prevents Logger from always reporting the wrapper code as the caller.
func (l Logger) AddCallerSkip(skip int) Logger {
	s := l.s.Desugar().WithOptions(zap.AddCallerSkip(skip)).Sugar()
	return Logger{service: l.service, s: s, cleanup: l.cleanup}
}

// Package returns a copy of the logger with the "pkg" set to the argument.
// It should be called before the original Logger has had any keys set to values, otherwise confusion may ensue.
func (l Logger) Package(pkg string) Logger {
	return Logger{service: l.service, s: l.s.With("pkg", pkg), cleanup: l.cleanup}
}

// GRPCLoggers returns server side logging middleware for gRPC servers
func (l Logger) GRPCLoggers() (grpc.StreamServerInterceptor, grpc.UnaryServerInterceptor) {
	logger := l.s.Desugar()
	return grpc_zap.StreamServerInterceptor(logger), grpc_zap.UnaryServerInterceptor(logger)
}
