package logger

import (
	"bytes"
	"log/slog"
	"runtime"
	"strconv"
	"sync"

	"github.com/pkg/errors"
)

type GIDContext struct {
	l *instance
}

func (g *GIDContext) reqid() *slog.Logger {
	if goroutineMap != nil {
		gid := GetGID()
		if reqid, ok := goroutineMap.Load(gid); ok {
			l := g.l.logger.With(slog.Group("trace",
				slog.Any("req_id", reqid),
				slog.Uint64("gid", gid),
			))
			return l
		}
	}
	return g.l.logger
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
	return curGoroutineID()
}

// https://github.com/golang/net/blob/master/http2/gotrack.go#L22

var goroutineSpace = []byte("goroutine ")

func curGoroutineID() uint64 {
	bp := littleBuf.Get().(*[]byte)
	defer littleBuf.Put(bp)
	b := *bp
	b = b[:runtime.Stack(b, false)]
	// Parse the 4707 out of "goroutine 4707 ["
	b = bytes.TrimPrefix(b, goroutineSpace)
	i := bytes.IndexByte(b, ' ')
	if i < 0 {
		return 0
	}
	b = b[:i]
	n, err := parseUintBytes(b, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

var littleBuf = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 64)
		return &buf
	},
}

// parseUintBytes is like strconv.ParseUint, but using a []byte.
func parseUintBytes(s []byte, base int, bitSize int) (n uint64, err error) {
	var cutoff, maxVal uint64

	if bitSize == 0 {
		bitSize = int(strconv.IntSize)
	}

	s0 := s
	switch {
	case len(s) < 1:
		err = strconv.ErrSyntax
		goto Error

	case 2 <= base && base <= 36:
		// valid base; nothing to do

	case base == 0:
		// Look for octal, hex prefix.
		switch {
		case s[0] == '0' && len(s) > 1 && (s[1] == 'x' || s[1] == 'X'):
			base = 16
			s = s[2:]
			if len(s) < 1 {
				err = strconv.ErrSyntax
				goto Error
			}
		case s[0] == '0':
			base = 8
		default:
			base = 10
		}

	default:
		err = errors.New("invalid base " + strconv.Itoa(base))
		goto Error
	}

	n = 0
	cutoff = cutoff64(base)
	maxVal = 1<<uint(bitSize) - 1

	for i := 0; i < len(s); i++ {
		var v byte
		d := s[i]
		switch {
		case '0' <= d && d <= '9':
			v = d - '0'
		case 'a' <= d && d <= 'z':
			v = d - 'a' + 10
		case 'A' <= d && d <= 'Z':
			v = d - 'A' + 10
		default:
			n = 0
			err = strconv.ErrSyntax
			goto Error
		}
		if int(v) >= base {
			n = 0
			err = strconv.ErrSyntax
			goto Error
		}

		if n >= cutoff {
			// n*base overflows
			n = 1<<64 - 1
			err = strconv.ErrRange
			goto Error
		}
		n *= uint64(base)

		n1 := n + uint64(v)
		if n1 < n || n1 > maxVal {
			// n+v overflows
			n = 1<<64 - 1
			err = strconv.ErrRange
			goto Error
		}
		n = n1
	}

	return n, nil

Error:
	return n, &strconv.NumError{Func: "ParseUint", Num: string(s0), Err: err}
}

// Return the first number n such that n*base >= 1<<64.
func cutoff64(base int) uint64 {
	if base < 2 {
		return 0
	}
	return (1<<64-1)/uint64(base) + 1
}
