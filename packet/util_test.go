package packet

import (
	"encoding/json"
	"testing"
)

func TestMACAddr(t *testing.T) {
	examples := map[string]string{
		"mac":      `{"data":{"mac":"0c:c4:7a:c6:2f:1c"}}`,
		"null mac": `{"data":{"mac":null}}`,
		"no data":  `{"data":{}}`,
	}
	for name, example := range examples {
		t.Run(name, func(t *testing.T) {
			var res struct {
				Data struct {
					MAC *MACAddr `json:"mac"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(example), &res); err != nil {
				t.Errorf("parsing failed for %s: %v", example, err)
			}
		})
	}
}
