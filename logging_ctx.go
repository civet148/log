package log

import (
	"context"
)

const (
	LOG_CTX_KEY = "LOG_CTX_KEY"
)

type CtxOption func(lc *LogContext)

func WithTitle(title string) CtxOption {
	return func(lc *LogContext) {
		lc.title = title
	}
}

func WithFields(fields ...any) CtxOption {
	return func(lc *LogContext) {
		lc.fields = fields
	}
}

func WithLogger(logger Logger) CtxOption {
	return func(lc *LogContext) {
		lc.logger = logger
	}
}

type LogContext struct {
	title  string
	logger Logger
	fields []any
}

func (lc *LogContext) GetTitle() string {
	return lc.title
}

func (lc *LogContext) SetTitle(title string) {
	lc.title = title
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

func (lc *LogContext) SetFields(fields []any) {
	lc.fields = fields
}

func NewContext(ctx context.Context, opts ...CtxOption) context.Context {
	lc := &LogContext{}
	for _, opt := range opts {
		opt(lc)
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

func WithPrintf(ctx context.Context, title string, fields ...any) {
	var ok bool
	var lc *LogContext
	var opts []CtxOption
	opts = append(opts, WithTitle(title))
	opts = append(opts, WithFields(fields...))

	if lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext); !ok {
		Warnf("WithPrintf: log context not found")
		return
	}
	for _, opt := range opts {
		opt(lc)
	}
}

func PrintContext(ctx context.Context) (err error) {

	return nil
}
