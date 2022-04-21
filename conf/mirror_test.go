package conf

import (
	"net/url"
	"os"
	"reflect"
	"testing"
)

func Test_buildMirrorBaseURL(t *testing.T) {
	tests := []struct {
		name          string
		facilityCode  string
		MirrorBaseURL string
		want          *url.URL
		wantErr       bool
	}{
		{
			name:          "Accepts MIRROR_BASE_URL value that does not contain a port number",
			MirrorBaseURL: "http://10.10.10.10",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10",
			},
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that contains a port number",
			MirrorBaseURL: "http://10.10.10.10:8080",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10:8080",
			},
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that contains https scheme",
			MirrorBaseURL: "https://10.10.10.10",
			want: &url.URL{
				Scheme: "https",
				Host:   "10.10.10.10",
			},
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that contains a path",
			MirrorBaseURL: "http://10.10.10.10/path",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10",
				Path:   "/path",
			},
		},
		{
			name:          "Rejects MIRROR_BASE_URL value that is missing a scheme",
			MirrorBaseURL: "10.10.10.10",
			wantErr:       true,
		},
		{
			name:          "Rejects MIRROR_BASE_URL value that contains an unexpected scheme",
			MirrorBaseURL: "ssh://10.10.10.10",
			wantErr:       true,
		},
		{
			name:          "Rejects MIRROR_BASE_URL value that contains a query",
			MirrorBaseURL: "http://10.10.10.10/path?hello=world",
			wantErr:       true,
		},
		{
			name:          "Rejects MIRROR_BASE_URL value that contains a fragmet",
			MirrorBaseURL: "http://10.10.10.10/path#greeting",
			wantErr:       true,
		},
		{
			name: "Default value",
			want: &url.URL{
				Scheme: "http",
				Host:   "install.ewr1.packet.net",
			},
		},
		{
			name:         "Default value, override facility",
			facilityCode: "blue",
			want: &url.URL{
				Scheme: "http",
				Host:   "install.blue.packet.net",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.MirrorBaseURL != "" {
				os.Setenv("MIRROR_BASE_URL", tt.MirrorBaseURL)
			} else {
				os.Unsetenv("MIRROR_BASE_URL")
			}

			if tt.facilityCode != "" {
				FacilityCode = tt.facilityCode
			}
			got, err := buildMirrorBaseURL()
			if tt.wantErr {
				if err != nil {
					// pass
					return
				}
				t.Fatalf("buildMirrorBaseURL() did not return an error, instead returned: %s", got)
			}
			if err != nil {
				t.Fatalf("buildMirrorBaseURL() returned an unexpected error: %s", err)
			}

			want := tt.want.String()
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("buildMirrorBaseURL() mismatch, want=%v, got=%v", want, got)
			}
		})
	}
}
