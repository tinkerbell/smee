package conf

import (
	"net/url"
	"os"

	"github.com/pkg/errors"
)

func Default(name, value string) string {
	if s, ok := os.LookupEnv(name); ok {
		return s
	}
	return value
}

func DefaultURL(name, value string) *url.URL {
	str := Default(name, value)
	u, err := url.Parse(str)
	if err != nil {
		panic(errors.Errorf("invalid %s %q: %s", name, str, err))
	}
	return u
}

func Require(name string) string {
	if s, ok := os.LookupEnv(name); ok {
		return s
	}
	panic(errors.Errorf("missing required environment variable: %s", name))
}
