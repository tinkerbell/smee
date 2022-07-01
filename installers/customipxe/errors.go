package customipxe

import "errors"

var ErrEmptyIPXEConfig = errors.New("ipxe config URL or Script must be defined")
