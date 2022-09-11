package tcpforward

import (
	"github.com/wzshiming/sshd"
)

var name = "tcpip-forward"

func init() {
	tcpForward := &TCPForward{}
	sshd.RegistryHandleRequest(name, tcpForward.Forward)
	sshd.RegistryHandleRequest("cancel-"+name, tcpForward.Cancel)
}
