package tcpforward

import (
	"context"
	"sync"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

func init() {
	sshd.RegistryHandleRequest("tcpip-forward", Forward)
	sshd.RegistryHandleRequest("cancel-tcpip-forward", Cancel)
}

var (
	mut sync.Mutex

	cancelsMut sync.Mutex
	cancels    = map[uint32]context.CancelFunc{}
)

func cancelPort(port uint32) {
	cancelsMut.Lock()
	defer cancelsMut.Unlock()
	if cancel, ok := cancels[port]; ok {
		cancel()
		delete(cancels, port)
	}
}

func setCancelPort(port uint32, cf context.CancelFunc) {
	cancelsMut.Lock()
	defer cancelsMut.Unlock()
	cancels[port] = cf
}

func Forward(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	d := &TCPForward{
		ServerConn: serverConn,
	}
	mut.Lock()
	defer mut.Unlock()
	d.Forward(ctx, req)

}

func Cancel(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	d := &TCPForwardCancel{
		ServerConn: serverConn,
	}
	mut.Lock()
	defer mut.Unlock()
	d.Cancel(ctx, req)
}
