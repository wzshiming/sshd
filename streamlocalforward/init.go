package streamlocalforward

import (
	"github.com/wzshiming/sshd"
)

func init() {
	streamLocalForward := &StreamLocalForward{}
	sshd.RegistryHandleRequest("streamlocal-forward@openssh.com", streamLocalForward.Forward)
	sshd.RegistryHandleRequest("cancel-streamlocal-forward@openssh.com", streamLocalForward.Cancel)
}
