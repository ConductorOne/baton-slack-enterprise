package target

func fatalMethods() {
	var err error
	l.Fatal(logMsg, err)            // want `\QnoFatal: propagate errors rather than calling l.Fatal(logMsg, err)`
	l.Fatalf(logMsgFmt, err)        // want `\QnoFatal: propagate errors rather than calling l.Fatalf(logMsgFmt, err)`
	l.Fatalln(logMsg, err)          // want `\QnoFatal: propagate errors rather than calling l.Fatalln(logMsg, err)`
	l.Fatalw(logMsg, errorKey, err) // want `\QnoFatal: propagate errors rather than calling l.Fatalw(logMsg, errorKey, err)`
	l.FatalFn(func() []interface{} { // want `noFatal: propagate errors rather than calling l.FatalFn\(func\(\).*`
		return []interface{}{
			logMsg, err,
		}
	})
	l.fatal(logMsg, err) // want `\QnoFatal: propagate errors rather than calling l.fatal(logMsg, err)`
}

func fatalLevel() {
	var err error
	l.Log(LevelFatal, logMsg, err)     // want `\QnoFatal: propagate errors rather than logging at LevelFatal`
	l.Logf(LevelFatal, logMsgFmt, err) // want `\QnoFatal: propagate errors rather than logging at LevelFatal`
	l.Logln(LevelFatal, logMsg, err)   // want `\QnoFatal: propagate errors rather than logging at LevelFatal`
	l.Logw(LevelFatal, logMsg, err)    // want `\QnoFatal: propagate errors rather than logging at LevelFatal`
	l.log(LevelFatal, logMsg, err)     // want `\QnoFatal: propagate errors rather than logging at LevelFatal`
	l.WithLevel(LevelFatal)            // want `\QnoFatal: propagate errors rather than logging at LevelFatal`
}
