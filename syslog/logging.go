package syslog

import (
	"github.com/packethost/pkg/log"
)

var sysloglog log.Logger

func Init(l log.Logger) {
	sysloglog = l.Package("syslog")

	Error := func(args ...interface{}) {
		var err error
		if e, ok := args[0].(error); ok {
			err = e
			args = args[1:]
		}
		sysloglog.Error(err, args...)

	}
	loggerFuncs = map[byte]func(...interface{}){
		0: Error,
		1: Error,
		2: Error,
		3: Error,
		4: sysloglog.Info,
		5: sysloglog.Info,
		6: sysloglog.Info,
		7: sysloglog.Debug,
	}
}
