package target

type LogLevel int

const (
	LevelFatal LogLevel = iota
	LevelPanic
)

type Logger interface {
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})
	Fatal(args ...interface{})
	Panic(args ...interface{})
	FatalFn(fn LogFunction)
	PanicFn(fn LogFunction)
	Fatalln(args ...interface{})
	Panicln(args ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})
	Panicw(msg string, keysAndValues ...interface{})
	Log(lvl LogLevel, args ...interface{})
	Logf(lvl LogLevel, format string, args ...interface{})
	Logln(lvl LogLevel, args ...interface{})
	Logw(lvl LogLevel, msg string, keysAndValues ...interface{})
	fatal(args ...interface{})
	panic(args ...interface{})
	log(lvl LogLevel, args ...interface{})
	WithLevel(lvl LogLevel) Logger
}

type LogFunction func() []interface{}

var l = (Logger)(nil)

var logMsg = "encountered fatal error"

var errorKey = "error"

var logMsgFmt = logMsg + ": %v"
