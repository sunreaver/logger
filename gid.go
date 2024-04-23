package logger

import (
	"bytes"
	"log/slog"
	"runtime"
	"strconv"
)

type GIDContext struct {
	l *slog.Logger
}

func (g *GIDContext) reqid() *slog.Logger {
	if goroutineMap != nil {
		if reqid, ok := goroutineMap.Load(GetGID()); ok {
			return g.l.With("req_id", reqid)
		}
	}
	return g.l
}

func (g *GIDContext) Debugw(msg string, kv ...interface{}) {
	g.reqid().Debug(msg, kv...)
}

func (g *GIDContext) Infow(msg string, kv ...interface{}) {
	g.reqid().Info(msg, kv...)
}

func (g *GIDContext) Warnw(msg string, kv ...interface{}) {
	g.reqid().Warn(msg, kv...)
}

func (g *GIDContext) Errorw(msg string, kv ...interface{}) {
	g.reqid().Error(msg, kv...)
}

func (g *GIDContext) Panicw(msg string, kv ...interface{}) {
	g.reqid().Error(msg, kv...)
	panic(msg)
}

func GetGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
