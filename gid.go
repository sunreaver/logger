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
	return g.reqid().Info
}

func (g *GIDContext) Warnw() func(msg string, args ...any) {
	return g.reqid().Warn
}

func (g *GIDContext) Errorw() func(msg string, args ...any) {
	return g.reqid().Error
}

// GetGID 函数返回当前 goroutine 的 ID。
//
// 如果无法获取到 goroutine ID，则返回 0。
func GetGID() uint64 {
	// return uint64(goid.Get())
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	if idx := bytes.IndexByte(b, ' '); idx != -1 {
		b = b[:idx]
	} else {
		return 0
	}
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
