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
		mirrorBaseURL string
		mirrorHost    string
		want          *url.URL
		wantErr       bool
	}{
		{
			name:       "Accepts MIRROR_HOST value that contains a port number",
			mirrorHost: "10.10.10.10:8080",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10:8080",
			},
		},
		{
			name:       "Accepts MIRROR_HOST value that does not contain a port number",
			mirrorHost: "10.10.10.10",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10",
			},
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that does not contain a port number",
			mirrorBaseURL: "http://10.10.10.10",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10",
			},
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that contains a port number",
			mirrorBaseURL: "http://10.10.10.10:8080",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10:8080",
			},
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that contains a port number",
			mirrorBaseURL: "http://10.10.10.10:8080",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10:8080",
			},
		},
		{
			name:          "Rejects MIRROR_BASE_URL value that contains a path",
			mirrorBaseURL: "http://10.10.10.10/path",
			wantErr:       true,
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that contains a port number",
			mirrorBaseURL: "http://10.10.10.10:8080",
			mirrorHost:    "http://172.30.10.0",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10:8080",
			},
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
			if tt.mirrorBaseURL != "" {
				os.Setenv("MIRROR_BASE_URL", tt.mirrorBaseURL)
			} else {
				os.Unsetenv("MIRROR_BASE_URL")
			}

			if tt.mirrorHost != "" {
				os.Setenv("MIRROR_HOST", tt.mirrorHost)
			} else {
				os.Unsetenv("MIRROR_HOST")
			}

			if tt.facilityCode != "" {
				FacilityCode = tt.facilityCode
			}
			got, err := buildMirrorBaseURL()
			if (err != nil) != tt.wantErr {
				t.Errorf("buildMirrorBaseURL() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildMirrorBaseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildMirrorURL(t *testing.T) {
	tests := []struct {
		name          string
		facilityCode  string
		mirrorBaseURL string
		mirrorHost    string
		mirrorPath    string
		want          *url.URL
		wantErr       bool
	}{
		{
			name:       "Accepts MIRROR_HOST value that contains a port number",
			mirrorHost: "10.10.10.10:8080",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10:8080",
				Path:   defaultMirrorPath,
			},
		},
		{
			name:       "Accepts MIRROR_HOST value that does not contain a port number",
			mirrorHost: "10.10.10.10",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10",
				Path:   defaultMirrorPath},
		},
		{
			name:       "Specifying MIRROR_HOST and MIRROR_PATH returns the correct result",
			mirrorHost: "10.10.10.10",
			mirrorPath: "/my/special/path",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10",
				Path:   "/my/special/path"},
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that does not contain a port number",
			mirrorBaseURL: "http://10.10.10.10",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10",
				Path:   defaultMirrorPath},
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that contains a port number",
			mirrorBaseURL: "http://10.10.10.10:8080",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10:8080",
				Path:   defaultMirrorPath},
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that contains a port number",
			mirrorBaseURL: "http://10.10.10.10:8080",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10:8080",
				Path:   defaultMirrorPath},
		},
		{
			name:          "Specifying MIRROR_BASE_URL and MIRROR_PATH returns the correct result",
			mirrorBaseURL: "http://10.10.10.10",
			mirrorPath:    "/my/special/path",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10",
				Path:   "/my/special/path"},
		},
		{
			name:          "Rejects MIRROR_BASE_URL value that contains a path",
			mirrorBaseURL: "http://10.10.10.10/path",
			wantErr:       true,
		},
		{
			name:          "Accepts MIRROR_BASE_URL value that contains a port number",
			mirrorBaseURL: "http://10.10.10.10:8080",
			mirrorHost:    "http://172.30.10.0",
			want: &url.URL{
				Scheme: "http",
				Host:   "10.10.10.10:8080",
				Path:   defaultMirrorPath},
		},
		{
			name: "Default value",
			want: &url.URL{
				Scheme: "http",
				Host:   "install.ewr1.packet.net",
				Path:   defaultMirrorPath},
		},
		{
			name:         "Default value, override facility",
			facilityCode: "blue",
			want: &url.URL{
				Scheme: "http",
				Host:   "install.blue.packet.net",
				Path:   defaultMirrorPath},
		},
		{
			name:       "Default value, override MIRROR_PATH",
			mirrorPath: "/my/special/path",
			want: &url.URL{
				Scheme: "http",
				Host:   "install.ewr1.packet.net",
				Path:   "/my/special/path"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mirrorBaseURL != "" {
				os.Setenv("MIRROR_BASE_URL", tt.mirrorBaseURL)
			} else {
				os.Unsetenv("MIRROR_BASE_URL")
			}

			if tt.mirrorHost != "" {
				os.Setenv("MIRROR_HOST", tt.mirrorHost)
			} else {
				os.Unsetenv("MIRROR_HOST")
			}

			if tt.mirrorPath != "" {
				os.Setenv("MIRROR_PATH", tt.mirrorPath)
			} else {
				os.Unsetenv("MIRROR_PATH")
			}

			if tt.facilityCode != "" {
				FacilityCode = tt.facilityCode
			} else {
				FacilityCode = defaultFacility
			}

			got, err := buildMirrorURL()
			if (err != nil) != tt.wantErr {
				t.Errorf("buildMirrorURL() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildMirrorURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
