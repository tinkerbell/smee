package job

import (
	"context"
)

func (j Job) Fatal(err error, args ...interface{}) {
	j.Logger.AddCallerSkip(1).Error(err, args...)
	panic(err)
}

func (j Job) Error(err error, args ...interface{}) {
	j.Logger.AddCallerSkip(1).Error(err, args...)
	j.postEvent(context.Background(), "boots.warning", "Tinkerbell Warning: "+err.Error(), true)
}
