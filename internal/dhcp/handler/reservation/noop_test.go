package reservation

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNoop(t *testing.T) {
	want := errors.New("no backend specified, please specify a backend")
	_, _, got := noop{}.GetByMac(context.TODO(), nil)
	if diff := cmp.Diff(want.Error(), got.Error()); diff != "" {
		t.Fatal(diff)
	}
	_, _, got = noop{}.GetByIP(context.TODO(), nil)
	if diff := cmp.Diff(want.Error(), got.Error()); diff != "" {
		t.Fatal(diff)
	}
}
