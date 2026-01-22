package log

import (
	"context"
)

type logCtxOption func(lc *LogContext)

type LogContext struct {
	printReply bool
	logger     Logger
}

func NewLogContext(opts ...logCtxOption) *LogContext {
	logger, _ := NewLogger()
	lc := &LogContext{
		logger: logger,
	}

	return lc
}
func CtxPrintf(ctx context.Context, format string, args ...any) {
	//var lc *LogContext

}

func WithCtxFields(ctx context.Context, kvs ...any) {

}
