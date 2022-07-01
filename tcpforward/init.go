package tcpforward

import (
	"github.com/wzshiming/sshd"
)

func init() {
	tcpForward := &TCPForward{}
	sshd.RegistryHandleRequest("tcpip-forward", tcpForward.Forward)
	sshd.RegistryHandleRequest("cancel-tcpip-forward", tcpForward.Cancel)
}
