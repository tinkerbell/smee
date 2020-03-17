package env

import "github.com/packethost/pkg/log"

var envlog log.Logger

func Init(l log.Logger) {
	envlog = l.Package("env")
}
