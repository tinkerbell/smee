package main

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestExtractTraceparentFromFilename(t *testing.T) {
	tests := map[string]struct {
		fileIn  string
		fileOut string
		err     error
		spanId  string
		traceId string
	}{
		"do nothing when no tp": {fileIn: "undionly.ipxe", fileOut: "undionly.ipxe", err: nil},
		"ignore bad filename": {
			fileIn:  "undionly.ipxe-00-0000-0000-00",
			fileOut: "undionly.ipxe-00-0000-0000-00",
			err:     nil,
		},
		"ignore corrupt tp": {
			fileIn:  "undionly.ipxe-00-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-abcdefghijklmnop-01",
			fileOut: "undionly.ipxe-00-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-abcdefghijklmnop-01",
			err:     nil,
		},
		"extract tp": {
			fileIn:  "undionly.ipxe-00-23b1e307bb35484f535a1f772c06910e-d887dc3912240434-01",
			fileOut: "undionly.ipxe",
			err:     nil,
			spanId:  "d887dc3912240434",
			traceId: "23b1e307bb35484f535a1f772c06910e",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			ctx, outfile, err := extractTraceparentFromFilename(ctx, tc.fileIn)
			if err != tc.err {
				t.Errorf("filename %q should have resulted in error %q but got %q", tc.fileIn, tc.err, err)
			}
			if outfile != tc.fileOut {
				t.Errorf("filename %q should have resulted in %q but got %q", tc.fileIn, tc.fileOut, outfile)
			}

			if tc.spanId != "" {
				sc := trace.SpanContextFromContext(ctx)
				got := sc.SpanID().String()
				if tc.spanId != got {
					t.Errorf("got incorrect span id from context, expected %q but got %q", tc.spanId, got)
				}
			}

			if tc.traceId != "" {
				sc := trace.SpanContextFromContext(ctx)
				got := sc.TraceID().String()
				if tc.traceId != got {
					t.Errorf("got incorrect trace id from context, expected %q but got %q", tc.traceId, got)
				}
			}
		})
	}
}
