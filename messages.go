package sshd

//ExitStatusMsg copy from golang.org/x/crypto/ssh.exitStatusMsg
type ExitStatusMsg struct {
	Status uint32
}

//PtyRequestMsg copy from golang.org/x/crypto/ssh.ptyRequestMsg
type PtyRequestMsg struct {
	Term     string
	Columns  uint32
	Rows     uint32
	Width    uint32
	Height   uint32
	Modelist string
}

//PtyWindowChangeMsg copy from golang.org/x/crypto/ssh.ptyWindowChangeMsg
type PtyWindowChangeMsg struct {
	Columns uint32
	Rows    uint32
	Width   uint32
	Height  uint32
}

//SetenvRequest copy from golang.org/x/crypto/ssh.setenvRequest
type SetenvRequest struct {
	Name  string
	Value string
}

//ExecMsg copy from golang.org/x/crypto/ssh.execMsg
type ExecMsg struct {
	Command string
}

//ChannelOpenDirectMsg copy from golang.org/x/crypto/ssh.channelOpenDirectMsg
type ChannelOpenDirectMsg struct {
	RAddr string
	RPort uint32
	LAddr string
	LPort uint32
}
