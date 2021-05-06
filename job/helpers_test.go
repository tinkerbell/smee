package job

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tinkerbell/boots/packet"
)

func TestPasswordHash(t *testing.T) {
	tests := map[string]struct {
		input Job
		want  string
	}{
		"job instance is nil": {
			want:  "",
			input: Job{},
		},
		"only CryptedRootPassword is populated": {
			want: "supersecret",
			input: Job{
				instance: &packet.Instance{
					CryptedRootPassword: "supersecret",
				},
			},
		},
		"only PasswordHash is populated": {
			want: "supersecret",
			input: Job{
				instance: &packet.Instance{
					PasswordHash: "supersecret",
				},
			},
		},
		"CryptedRootPassword is preferred over PasswordHash": {
			want: "cryptedrootpassword",
			input: Job{
				instance: &packet.Instance{
					CryptedRootPassword: "cryptedrootpassword",
					PasswordHash:        "passwordhash",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.input.PasswordHash()
			diff := cmp.Diff(tc.want, got)
			if diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}
