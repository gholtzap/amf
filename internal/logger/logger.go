package logger

import (
	"log"
	"os"
)

// Logger represents a component logger
type Logger struct {
	prefix string
	logger *log.Logger
}

var (
	// Component loggers
	MainLog    *Logger
	CfgLog     *Logger
	NasLog     *Logger
	NgapLog    *Logger
	CtxLog     *Logger
	HandlerLog *Logger
	ConsumerLog *Logger
	SbiLog     *Logger
	UtilLog    *Logger
)

func init() {
	MainLog = newLogger("AMF", "Main")
	CfgLog = newLogger("AMF", "Config")
	NasLog = newLogger("AMF", "NAS")
	NgapLog = newLogger("AMF", "NGAP")
	CtxLog = newLogger("AMF", "Context")
	HandlerLog = newLogger("AMF", "Handler")
	ConsumerLog = newLogger("AMF", "Consumer")
	SbiLog = newLogger("AMF", "SBI")
	UtilLog = newLogger("AMF", "Util")
}

// newLogger creates a new logger with prefix
func newLogger(module, component string) *Logger {
	prefix := "[" + module + "][" + component + "] "
	return &Logger{
		prefix: prefix,
		logger: log.New(os.Stdout, prefix, log.LstdFlags|log.Lmsgprefix),
	}
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Println("[INFO] " + msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, v ...interface{}) {
	l.logger.Printf("[INFO] "+format, v...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Println("[WARN] " + msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.logger.Printf("[WARN] "+format, v...)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Println("[ERROR] " + msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.logger.Printf("[ERROR] "+format, v...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Println("[DEBUG] " + msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.logger.Printf("[DEBUG] "+format, v...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.logger.Fatalln("[FATAL] " + msg)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatalf("[FATAL] "+format, v...)
}
