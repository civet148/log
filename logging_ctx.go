package log

import (
	"context"
	"fmt"
	"strings"
	"time"
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
	logger       Logger
	fields       []any
	msgs         []string
	milliSeconds int64
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

func (lc *LogContext) SetFields(kvs ...any) {
	lc.fields = kvs
}

func (lc *LogContext) AppendFields(kvs ...any) {
	lc.fields = append(lc.fields, kvs...)
}

func (lc *LogContext) AppendMsg(msg string, args ...any) {
	lc.msgs = append(lc.msgs, fmt.Sprintf(msg, args...))
}

func NewContext(ctx context.Context, opts ...CtxOption) context.Context {
	lc := &LogContext{
		milliSeconds: time.Now().UnixMilli(),
	}
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

func WithFields(ctx context.Context, kvs ...any) {
	var ok bool
	var lc *LogContext
	if lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext); !ok {
		Warnf("WithPrintf: log context not found")
		return
	}
	lc.AppendFields(kvs...)
}

func WithPrintf(ctx context.Context, msg string, args ...any) {
	var ok bool
	var lc *LogContext
	if lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext); !ok {
		Warnf("WithPrintf: log context not found")
		return
	}
	//查询当前报错的方法名和文件名行号等信息
	caller := getCaller(2)
	lc.AppendFields("caller", caller)
	lc.AppendMsg(msg, args...)
}

func WithError(ctx context.Context, err error) error {
	var ok bool
	var lc *LogContext
	if lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext); !ok {
		Warnf("WithPrintf: log context not found")
		return err
	}
	lc.AppendFields("error", err.Error())
	return err
}

func PrintContext(ctx context.Context) {
	var ok bool
	var lc *LogContext
	if lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext); !ok {
		Warnf("PrintContext: log context not found")
		return
	}

	ms := time.Now().UnixMilli()
	dur := convertMsToDurationStr(ms - lc.milliSeconds)
	lc.AppendFields("duration", dur)

	logger := lc.logger
	if logger == nil {
		Warnf("PrintContext: logger is nil")
		return
	}
	var msg = strings.Join(lc.msgs, "; ")
	logger.WithFields(lc.fields...)
	logger.Infof(msg)
}

// 总毫秒数转为 1d2h30m5s.321 格式
// ms: 总毫秒数，可正负
func convertMsToDurationStr(ms int64) string {
	if ms == 0 {
		return "0s"
	}

	// 处理负号
	neg := false
	if ms < 0 {
		neg = true
		ms = -ms
	}

	const (
		msPerSec  = 1000
		msPerMin  = 60 * msPerSec
		msPerHour = 60 * msPerMin
		msPerDay  = 24 * msPerHour
	)

	days := ms / msPerDay
	ms %= msPerDay

	hours := ms / msPerHour
	ms %= msPerHour

	mins := ms / msPerMin
	ms %= msPerMin

	secs := ms / msPerSec
	ms %= msPerSec

	var builder strings.Builder

	if days > 0 {
		fmt.Fprintf(&builder, "%dd", days)
	}
	if hours > 0 {
		fmt.Fprintf(&builder, "%dh", hours)
	}
	if mins > 0 {
		fmt.Fprintf(&builder, "%dm", mins)
	}

	// 秒+毫秒部分
	if secs > 0 || ms > 0 {
		fmt.Fprint(&builder, secs)
		if ms > 0 {
			fmt.Fprintf(&builder, ".%03d", ms)
		}
		builder.WriteByte('s')
	}

	res := builder.String()
	if neg {
		res = "-" + res
	}
	return res
}
