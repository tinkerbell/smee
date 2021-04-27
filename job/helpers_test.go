package job

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tinkerbell/boots/packet"
)

func TestGetPasswordHash(t *testing.T) {
	tests := map[string]struct {
		input Job
		want  string
	}{
		"job instance is nil": {input: Job{}, want: ""},
		"password hash has a value": {input: Job{instance: &packet.Instance{PasswordHash: "supersecret"}}, want: "supersecret"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.input.GetPasswordHash()
			diff := cmp.Diff(tc.want, got)
			if diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}


func TestCryptedPassword(t *testing.T) {
	tests := map[string]struct {
		input Job
		want  string
	}{
		"job instance is nil": {input: Job{}, want: ""},
		"CryptedRootPassword has a value": {input: Job{instance: &packet.Instance{CryptedRootPassword: "supersecret"}}, want: "supersecret"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.input.CryptedPassword()
			diff := cmp.Diff(tc.want, got)
			if diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}