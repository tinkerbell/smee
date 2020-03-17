package httplog

import (
	"github.com/packethost/pkg/log"
)

var httplog log.Logger

func Init(l log.Logger) {
	httplog = l.Package("http")
}
