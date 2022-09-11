package directstreamlocal

import (
	"github.com/wzshiming/sshd"
)

var name = "direct-streamlocal"

func init() {
	directStreamLocal := DirectStreamLocal{}
	sshd.RegistryHandleChannel(name+"@openssh.com", directStreamLocal.Handle)
}
