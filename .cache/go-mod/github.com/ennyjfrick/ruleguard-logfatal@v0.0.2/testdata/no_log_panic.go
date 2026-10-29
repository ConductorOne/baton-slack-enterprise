package target

func panicMethods() {
	var err error
	l.Panic(logMsg, err)            // want `\QnoPanic: propagate errors rather than calling l.Panic(logMsg, err)`
	l.Panicf(logMsgFmt, err)        // want `\QnoPanic: propagate errors rather than calling l.Panicf(logMsgFmt, err)`
	l.Panicln(logMsg, err)          // want `\QnoPanic: propagate errors rather than calling l.Panicln(logMsg, err)`
	l.Panicw(logMsg, errorKey, err) // want `\QnoPanic: propagate errors rather than calling l.Panicw(logMsg, errorKey, err)`
	l.PanicFn(func() []interface{} { // want `noPanic: propagate errors rather than calling l.PanicFn\(func\(\).*`
		return []interface{}{
			logMsg, err,
		}
	})
	l.panic(logMsg, err) // want `\QnoPanic: propagate errors rather than calling l.panic(logMsg, err)`
}

func panicLevel() {
	var err error
	l.Log(LevelPanic, logMsg, err)     // want `\QnoPanic: propagate errors rather than logging at LevelPanic`
	l.Logf(LevelPanic, logMsgFmt, err) // want `\QnoPanic: propagate errors rather than logging at LevelPanic`
	l.Logln(LevelPanic, logMsg, err)   // want `\QnoPanic: propagate errors rather than logging at LevelPanic`
	l.Logw(LevelPanic, logMsg, err)    // want `\QnoPanic: propagate errors rather than logging at LevelPanic`
	l.log(LevelPanic, logMsg, err)     // want `\QnoPanic: propagate errors rather than logging at LevelPanic`
	l.WithLevel(LevelPanic)            // want `\QnoPanic: propagate errors rather than logging at LevelPanic`
}
