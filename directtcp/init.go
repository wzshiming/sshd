package directtcp

import (
	"github.com/wzshiming/sshd"
)

func init() {
	directTcp := DirectTCP{}
	sshd.RegistryHandleChannel("direct-tcpip", directTcp.Handle)
}
