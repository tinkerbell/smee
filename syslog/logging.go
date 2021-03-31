package syslog

import (
	"github.com/packethost/pkg/log"
)

var sysloglog log.Logger

func Init(l log.Logger) {
	sysloglog = l.Package("syslog")
}
