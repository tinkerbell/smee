package installers

import (
	"sync"

	"github.com/packethost/pkg/log"
)

var (
	installerslog log.Logger
	loggers       sync.Map
)

func Init(l log.Logger) {
	installerslog = l
}

func Logger(os string) log.Logger {
	logger, ok := loggers.Load(os)
	if !ok {
		logger = installerslog.Package("installers/" + os)
		logger, _ = loggers.LoadOrStore(os, logger)
	}

	return logger.(log.Logger)
}
