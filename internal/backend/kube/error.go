package kube

type hardwareNotFoundError struct{}

func (hardwareNotFoundError) NotFound() bool { return true }

func (hardwareNotFoundError) Error() string { return "hardware not found" }
