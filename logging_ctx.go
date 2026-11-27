package log

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	LOG_CTX_KEY   = "LOG_CTX_KEY"
	CALLER_KEY    = "@caller"
	STACK_KEY     = "@stack"
	DURATION_KEY  = "@duration"
	TIMESTAMP_KEY = "@timestamp"
)

type CtxOption func(lc *LogContext)

func WithLogger(logger Logger) CtxOption {
	return func(lc *LogContext) {
		lc.logger = logger
	}
}

type logCtxMsg struct {
	level LogLevel
	msg   string
}
type LogContext struct {
	logger       Logger
	fields       []any
	msgs         []logCtxMsg
	caller       string
	milliSeconds int64
	hasError     bool
	startTime    time.Time
	level        LogLevel
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

func (lc *LogContext) AppendLevelMsg(level LogLevel, msg string, args ...any) {
	lc.msgs = append(lc.msgs, lc.makeLogCtxMsg(level, msg, args...))
}

func (lc *LogContext) makeLogCtxMsg(level LogLevel, msg string, args ...any) logCtxMsg {
	return logCtxMsg{
		level: level,
		msg:   fmt.Sprintf(msg, args...),
	}
}

func (lc *LogContext) withCtxPrintf(level LogLevel, msg string, args ...any) {
	//查询当前报错的方法名和文件名行号等信息
	lc.caller = getCaller(4)
	lc.AppendFields(CALLER_KEY, lc.caller)
	if level > lc.level {
		lc.level = level
	}
	if msg != "" {
		lc.AppendLevelMsg(level, msg, args...)
	}
}

func NewContext(ctx context.Context, opts ...CtxOption) context.Context {
	lc := &LogContext{
		startTime:    time.Now(),
		milliSeconds: time.Now().UnixMilli(),
		level:        -1,
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

func WithError(ctx context.Context, err error) error {
	var ok bool
	var lc *LogContext
	if lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext); !ok {
		Warnf("WithPrintf: log context not found")
		return err
	}
	if !lc.hasError {
		lc.hasError = true
		lc.AppendFields("error", err.Error())
		lc.AppendFields(STACK_KEY, getCallStack(2))
	}
	return err
}

func (lc *LogContext) getLogLevelMsg() string {
	var levelMsgs []string
	for _, m := range lc.msgs {
		levelMsgs = append(levelMsgs, fmt.Sprintf("[%v] %s", LevelShortNames[m.level], m.msg))
	}
	return strings.Join(levelMsgs, " | ")
}

func PrintContext(ctx context.Context, levels ...LogLevel) {
	var ok bool
	var lc *LogContext
	if lc, ok = ctx.Value(LOG_CTX_KEY).(*LogContext); !ok {
		Warnf("PrintContext: log context not found")
		return
	}

	ms := time.Now().UnixMilli()
	dur := convertMsToDurationStr(ms - lc.milliSeconds)
	lc.AppendFields(TIMESTAMP_KEY, lc.startTime.Format(DateTimeFormat))
	lc.AppendFields(DURATION_KEY, dur)
	logger := lc.logger
	if logger == nil {
		Warnf("PrintContext: logger is nil")
		return
	}
	var msg = lc.getLogLevelMsg()
	logger.WithFields(lc.fields...)
	if lc.hasError {
		_ = logger.Errorf(msg)
	} else {
		level := LevelInfo
		if len(levels) > 0 {
			level = levels[0]
		} else if lc.level != -1 {
			level = lc.level
		}
		switch level {
		case LevelFatal:
			logger.Fatalf(msg)
		case LevelError:
			_ = logger.Errorf(msg)
		case LevelWarn:
			logger.Warnf(msg)
		case LevelInfo:
			logger.Infof(msg)
		case LevelDebug:
			logger.Debugf(msg)
		case LevelTrace:
			logger.Tracef(msg)
		default:
			logger.Infof(msg)
		}
	}
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
