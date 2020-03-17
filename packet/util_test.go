package packet

import (
	"encoding/json"
	"testing"
)

func TestMACAddr(t *testing.T) {
	examples := []string{
		`{"data":{"mac":"0c:c4:7a:c6:2f:1c"}}`,
		`{"data":{"mac":null}}`,
		`{"data":{}}`,
	}
	for _, example := range examples {
		var res struct {
			Data struct {
				MAC *MACAddr `json:"mac"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(example), &res); err != nil {
			t.Errorf("parsing failed for %s: %v", example, err)
		}
	}
}
