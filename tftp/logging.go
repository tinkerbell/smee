package tftp

import (
	"github.com/packethost/pkg/log"
)

var tftplog log.Logger

func Init(l log.Logger) {
	tftplog = l.Package("tftp")
}
