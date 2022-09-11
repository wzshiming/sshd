package streamlocalforward

import (
	"github.com/wzshiming/sshd"
)

var name = "streamlocal-forward"

func init() {
	streamLocalForward := &StreamLocalForward{}
	sshd.RegistryHandleRequest(name+"@openssh.com", streamLocalForward.Forward)
	sshd.RegistryHandleRequest("cancel-"+name+"@openssh.com", streamLocalForward.Cancel)
}
