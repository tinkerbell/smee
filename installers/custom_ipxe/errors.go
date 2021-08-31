package custom_ipxe

import "errors"

var (
	ErrEmptyIpxeConfig = errors.New("ipxe config URL or Script must be defined")
)
