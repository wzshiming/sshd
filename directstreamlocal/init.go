package directstreamlocal

import (
	"github.com/wzshiming/sshd"
)

func init() {
	directStreamLocal := DirectStreamLocal{}
	sshd.RegistryHandleChannel("direct-streamlocal@openssh.com", directStreamLocal.Handle)
}
