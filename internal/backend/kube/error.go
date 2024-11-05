package kube

import (
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type hardwareNotFoundError struct{}

func (hardwareNotFoundError) NotFound() bool { return true }

func (hardwareNotFoundError) Error() string { return "hardware not found" }

// Status() implements the APIStatus interface from apimachinery/pkg/api/errors
// so that IsNotFound function could be used against this error type.
func (hardwareNotFoundError) Status() metav1.Status {
	return metav1.Status{
		Reason: metav1.StatusReasonNotFound,
		Code:   http.StatusNotFound,
	}
}
