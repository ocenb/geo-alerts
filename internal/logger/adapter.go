package logger

import (
	"log/slog"
)

type AsynqLogger struct {
	log *slog.Logger
}

func NewAsynqAdapter(log *slog.Logger) *AsynqLogger {
	return &AsynqLogger{log: log}
}

func (a *AsynqLogger) Debug(args ...any) { a.log.Debug("asynq", "msg", args) }
func (a *AsynqLogger) Info(args ...any)  { a.log.Info("asynq", "msg", args) }
func (a *AsynqLogger) Warn(args ...any)  { a.log.Warn("asynq", "msg", args) }
func (a *AsynqLogger) Error(args ...any) { a.log.Error("asynq", "msg", args) }
func (a *AsynqLogger) Fatal(args ...any) { a.log.Error("asynq fatal", "msg", args) }
