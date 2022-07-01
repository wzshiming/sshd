package session

import (
	"github.com/wzshiming/sshd"
)

func init() {
	session := &Session{}
	sshd.RegistryHandleChannel("session", session.Handle)
}
