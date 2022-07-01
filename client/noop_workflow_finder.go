package client

import "context"

// NoOpWokrflowFinder is used to always return false. This is used when no workflow engine applies.
type NoOpWorkflowFinder struct{}

// HasActiveWorkflow always returns false without error.
func (f *NoOpWorkflowFinder) HasActiveWorkflow(context.Context, HardwareID) (bool, error) {
	return false, nil
}
