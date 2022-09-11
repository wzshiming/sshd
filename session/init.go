package session

import (
	"github.com/wzshiming/sshd"
)

var name = "session"

func init() {
	session := &Session{}
	sshd.RegistryHandleChannel(name, session.Handle)
}
