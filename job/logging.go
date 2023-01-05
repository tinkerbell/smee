package job

func (j Job) Fatal(err error, args ...interface{}) {
	j.Logger.AddCallerSkip(1).Error(err, args...)
	panic(err)
}

func (j Job) Error(err error, args ...interface{}) {
	j.Logger.AddCallerSkip(1).Error(err, args...)
}
