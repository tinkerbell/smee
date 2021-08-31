package job

import (
	"context"

	"github.com/packethost/pkg/log"
)

var joblog log.Logger

func Init(l log.Logger) {
	joblog = l.Package("http")
	initRSA()
}

func (j Job) Fatal(err error, args ...interface{}) {
	j.Logger.AddCallerSkip(1).Error(err, args...)
	panic(err)
}

func (j Job) Error(err error, args ...interface{}) {
	j.Logger.AddCallerSkip(1).Error(err, args...)
	j.postEvent(context.Background(), "boots.warning", "Tinkerbell Warning: "+err.Error(), true)
}
