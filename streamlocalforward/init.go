package streamlocalforward

import (
	"context"
	"sync"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

func init() {
	sshd.RegistryHandleRequest("streamlocal-forward@openssh.com", Forward)
	sshd.RegistryHandleRequest("cancel-streamlocal-forward@openssh.com", Cancel)
}

var (
	mut     sync.Mutex
	cancels = map[string]context.CancelFunc{}
)

func cancelPath(path string) {
	if cancel, ok := cancels[path]; ok {
		cancel()
		delete(cancels, path)
	}
}

func Forward(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	d := &StreamLocalForward{
		ServerConn: serverConn,
	}
	mut.Lock()
	defer mut.Unlock()
	d.Forward(ctx, req)

}

func Cancel(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	d := &StreamLocalForwardCancel{
		ServerConn: serverConn,
	}
	mut.Lock()
	defer mut.Unlock()
	d.Cancel(ctx, req)
}
