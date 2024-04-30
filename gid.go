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

func (g *GIDContext) Debugw() func(msg string, args ...any) {
	return g.reqid().Debug
}

func (g *GIDContext) Infow() func(msg string, args ...any) {
	return g.reqid().Debug
}

func (g *GIDContext) Warnw() func(msg string, args ...any) {
	return g.reqid().Debug
}

func (g *GIDContext) Errorw() func(msg string, args ...any) {
	return g.reqid().Error
}

func GetGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
