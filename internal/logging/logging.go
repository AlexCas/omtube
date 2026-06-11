// Package logging crea un logger Zap que escribe a un archivo, evitando contaminar
// la salida de la TUI en stdout/stderr.
package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// New crea un logger que escribe JSON al archivo indicado.
func New(logFile string) *zap.Logger {
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    5, // MB
		MaxBackups: 2,
		MaxAge:     14, // días
	})
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(encoder, w, zapcore.InfoLevel)
	return zap.New(core)
}
