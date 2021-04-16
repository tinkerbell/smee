// +build tools

package tools

import (
	_ "github.com/golang/mock/mockgen"
	_ "github.com/kevinburke/go-bindata/go-bindata"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/cmd/stringer"
)
