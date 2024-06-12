package logger

import (
	"log/slog"

	"github.com/petermattis/goid"
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

func GetGID() uint64 {
	return uint64(goid.Get())
}
