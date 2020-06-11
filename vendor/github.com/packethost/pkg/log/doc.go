// Copyright 2020 - 2020, Packethost, Inc and contributors
// SPDX-License-Identifier: Apache-2.0

// Package log sets up a shared logger that can be used by all packages run under one binary.
//
// This package wraps zap very lightly so zap best practices apply here too, namely use `With` for KV pairs to add context to a line.
// The lack of a wide gamut of logging levels is by design.
// The intended use case for each of the levels are:
//   Error:
//     Logs a message as an error, may also have external side effects such as posting to rollbar, sentry or alerting directly.
//   Info:
//     Used for production.
//     Context should all be in K=V pairs so they can be useful to ops and future-you-at-3am.
//   Debug:
//     Meant for developer use *during development*.
package log
