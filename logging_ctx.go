package log

import (
	"context"
	"fmt"
	"strings"
)

const (
	LOG_CTX_KEY = "LOG_CTX_KEY"
)

type CtxOption func(lc *LogContext)

func WithLogger(logger Logger) CtxOption {
	return func(lc *LogContext) {
		lc.logger = logger
	}
}

type LogContext struct {
	logger Logger
	fields []any
	msgs   []string
}

func (lc *LogContext) GetLogger() Logger {
	return lc.logger
}

func (lc *LogContext) SetLogger(logger Logger) {
	lc.logger = logger
}

func (lc *LogContext) GetFields() []any {
	return lc.fields
}

func (lc *LogContext) SetFields(fields ...any) {
	lc.fields = fields
}

func (lc *LogContext) AppendFields(fields ...any) {
	lc.fields = append(lc.fields, fields...)
}

func (lc *LogContext) AppendMsg(msg string, args ...any) {
	lc.msgs = append(lc.msgs, fmt.Sprintf(msg, args...))
}

func NewContext(ctx context.Context, opts ...CtxOption) context.Context {
	lc := &LogContext{}
	for _, opt := range opts {
		opt(lc)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if lc.logger == nil {
		lc.logger = getDefaultLogger()
	}
	ctx = context.WithValue(ctx, LOG_CTX_KEY, lc)
	return ctx
}

func GetLogContext(ctx context.Context) (lc *LogContext, ok bool) {
	lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext)
	return lc, ok
}

func WithPrintFields(ctx context.Context, fields ...any) {
	var ok bool
	var lc *LogContext
	if lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext); !ok {
		Warnf("WithPrintf: log context not found")
		return
	}
	lc.AppendFields(fields...)
}

func WithPrintf(ctx context.Context, msg string, args ...any) {
	var ok bool
	var lc *LogContext
	if lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext); !ok {
		Warnf("WithPrintf: log context not found")
		return
	}
	lc.AppendMsg(msg, args...)
}

func PrintContext(ctx context.Context) {
	var ok bool
	var lc *LogContext
	if lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext); !ok {
		Warnf("PrintContext: log context not found")
		return
	}
	logger := lc.logger
	if logger == nil {
		Warnf("PrintContext: logger is nil")
		return
	}
	var msg = strings.Join(lc.msgs, " ")
	logger.WithFields(lc.fields...)
	logger.Infof(msg)
}
