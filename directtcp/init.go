package directtcp

import (
	"github.com/wzshiming/sshd"
)

var name = "direct-tcpip"

func init() {
	directTcp := DirectTCP{}
	sshd.RegistryHandleChannel(name, directTcp.Handle)
}
