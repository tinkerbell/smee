package conf

import (
	"os"

	"github.com/pkg/errors"
)

func Require(name string) string {
	if s, ok := os.LookupEnv(name); ok {
		return s
	}
	panic(errors.Errorf("missing required environment variable: %s", name))
}
