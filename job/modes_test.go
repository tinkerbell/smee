package job

import (
	"net/url"
	"testing"
)

func TestModesFromQuery(t *testing.T) {
	q := url.Values{
		"modes": []string{"prov,deprov"},
	}
	m := modesFromQuery(q)

	if len(m) != 2 {
		t.Errorf("expected 2 modes to be in the set, got %d", len(m))
	}
	if !m[modeProv] {
		t.Errorf("expected prov to be in the set")
	}
	if !m[modeDeprov] {
		t.Errorf("expected deprov to be in the set")
	}

	if len(m) > 0 && !m[modeManagement] {
		// expected
	} else {
		t.Errorf("wtf?")
	}
}
