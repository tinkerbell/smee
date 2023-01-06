package client

import (
	"testing"
)

func TestServicesVersion(t *testing.T) {
	for _, test := range []struct {
		desc     string
		SV       ServicesVersion
		userdata string
		osie     string
	}{
		{desc: "empty"},
		{desc: "SV", SV: ServicesVersion{OSIE: "SV osie"}, osie: "SV osie"},
		{desc: "userdata", userdata: `#services={"osie":"userdata osie"}`, osie: "userdata osie"},
		{desc: "userdata:junk-text", userdata: `I'm a little teapot` + "\n" + `#services={"osie":"userdata osie"}` + "\n" + `short and stout!`, osie: "userdata osie"},
		{desc: "userdata:cloud-config", userdata: `#cloud-config` + "\n" + `#services={"osie":"userdata osie"}`, osie: "userdata osie"},
		{desc: "userdata:bash", userdata: `#!/usr/bin/bash` + "\n" + `#services={"osie":"userdata osie"}`, osie: "userdata osie"},
		{desc: "invalid userdata, not commented", userdata: `services={"osie":"userdata osie"}`},
		{desc: "invalid userdata, garbage at end commented", userdata: `services={"osie":"userdata osie"}blah`},
		{desc: "SV over userdata", SV: ServicesVersion{OSIE: "SV over osie"}, userdata: `#services={"osie":"userdata osie"}`, osie: "SV over osie"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			i := Instance{
				ServicesVersion: test.SV,
				UserData:        test.userdata,
			}
			got := i.GetServicesVersion().OSIE
			if got != test.osie {
				t.Fatalf("incorrect services version returned, want=%q, got=%q", test.osie, got)
			}
		})
	}
}
