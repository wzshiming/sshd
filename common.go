package sshd

import (
	"context"
	"io"
	"net"
	"os"
	"reflect"
	"runtime"
	"strings"
)

// BytesPool is an interface for getting and returning temporary
// bytes for use by io.CopyBuffer.
type BytesPool interface {
	Get() []byte
	Put([]byte)
}

type Logger interface {
	Println(v ...interface{})
}

// IsClosedConnError reports whether err is an error from use of a closed
// network connection.
func IsClosedConnError(err error) bool {
	if err == nil {
		return false
	}

	str := err.Error()
	if strings.Contains(str, "use of closed network connection") {
		return true
	}

	if runtime.GOOS == "windows" {
		if oe, ok := err.(*net.OpError); ok && oe.Op == "read" {
			if se, ok := oe.Err.(*os.SyscallError); ok && se.Syscall == "wsarecv" {
				const WSAECONNABORTED = 10053
				const WSAECONNRESET = 10054
				if n := errno(se.Err); n == WSAECONNRESET || n == WSAECONNABORTED {
					return true
				}
			}
		}
	}
	return false
}

func errno(v error) uintptr {
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Uintptr {
		return uintptr(rv.Uint())
	}
	return 0
}

// Tunnel create tunnels for two io.ReadWriteCloser
func Tunnel(ctx context.Context, c1, c2 io.ReadWriteCloser, buf1, buf2 []byte) error {
	errCh := make(chan error, 2)
	go func() {
		_, err := io.CopyBuffer(c1, c2, buf1)
		errCh <- err
	}()
	go func() {
		_, err := io.CopyBuffer(c2, c1, buf2)
		errCh <- err
	}()
	defer func() {
		_ = c1.Close()
		_ = c2.Close()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
