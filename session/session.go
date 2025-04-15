package session

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/shlex"
	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

// Session Handling for a single incoming connection
type Session struct{}

func (s *Session) Handle(ctx context.Context, newChan ssh.NewChannel, serverConn *sshd.ServerConn) {
	ch, reqs, err := newChan.Accept()
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("unable to accept NewChan:", err)
		}
		return
	}
	defer func() {
		b := ssh.Marshal(sshd.ExitStatusMsg{})
		ch.SendRequest("exit-status", false, b)
		ch.Close()
	}()

	if serverConn.Permissions != nil && !serverConn.Permissions.Allow(name, "") {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("prohibited:", name)
		}
		newChan.Reject(ssh.Prohibited, "Error administratively prohibited")
		return
	}

	var (
		ptyReq        *sshd.PtyRequestMsg
		winChangeChan chan *sshd.PtyWindowChangeMsg
	)
	ctx, cancel := context.WithCancel(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-reqs:
			if !ok {
				return
			}
			sess := true

			if serverConn.Permissions != nil && !serverConn.Permissions.Allow(name, req.Type) {
				if serverConn.Logger != nil {
					serverConn.Logger.Println("error administratively req:", req.Type)
				}
				continue
			}

			switch req.Type {
			case "pty-req":
				ptyreq := &sshd.PtyRequestMsg{}
				if err := ssh.Unmarshal(req.Payload, ptyreq); err != nil {
					if serverConn.Logger != nil {
						serverConn.Logger.Println("error unmarshalling pty-req:", err)
					}
					return
				}
				ptyReq = ptyreq
				winChangeChan = make(chan *sshd.PtyWindowChangeMsg, 1)
				s.Setenv(serverConn, "TERM", ptyreq.Term)
			case "window-change":
				winchangereq := &sshd.PtyWindowChangeMsg{}
				if err := ssh.Unmarshal(req.Payload, winchangereq); err != nil {
					if serverConn.Logger != nil {
						serverConn.Logger.Println("error unmarshalling window-change:", err)
					}
					return
				}
				winChangeChan <- winchangereq
			case "env":
				envreq := &sshd.SetenvRequest{}
				if err := ssh.Unmarshal(req.Payload, envreq); err != nil {
					if serverConn.Logger != nil {
						serverConn.Logger.Println("error unmarshalling env:", err)
					}
					return
				}
				s.Setenv(serverConn, envreq.Name, envreq.Value)
			case "shell":
				err := s.Shell(ctx, serverConn, ch, cancel, ptyReq, winChangeChan)
				if err != nil {
					if serverConn.Logger != nil {
						serverConn.Logger.Println("error execute:", err)
					}
					sess = false
				}
			case "exec":
				execReq := &sshd.ExecMsg{}
				if err := ssh.Unmarshal(req.Payload, execReq); err != nil {
					if serverConn.Logger != nil {
						serverConn.Logger.Println("error unmarshalling exec:", err)
					}
					return
				}
				err := s.Execute(ctx, serverConn, ch, cancel, execReq.Command)
				if err != nil {
					if serverConn.Logger != nil {
						serverConn.Logger.Println("error execute:", err)
					}
					sess = false
				}
			case "subsystem":
				subsystemReq := &sshd.SubsystemRequestMsg{}
				if err := ssh.Unmarshal(req.Payload, subsystemReq); err != nil {
					if serverConn.Logger != nil {
						serverConn.Logger.Println("error unmarshalling subsystem:", err)
					}
					return
				}

				if serverConn.Logger != nil {
					serverConn.Logger.Println("unknown subsystem request:", subsystemReq.Subsystem)
				}
				sess = false
			default:
				if serverConn.Logger != nil {
					serverConn.Logger.Println("unknown session request:", req.Type, req.Payload)
				}
				sess = false
			}
			if req.WantReply {
				req.Reply(sess, nil)
			}
		}
	}
}

func (s *Session) Setenv(serverConn *sshd.ServerConn, key, val string) {
	for i, env := range serverConn.Environ {
		se := strings.SplitN(env, "=", 2)
		if len(se) == 2 {
			if se[0] == key {
				serverConn.Environ[i] = fmt.Sprintf("%s=%s", key, val)
				return
			}
		}
	}
	serverConn.Environ = append(serverConn.Environ, fmt.Sprintf("%s=%s", key, val))
}

// Shell a process for the channel.
func (s *Session) Shell(ctx context.Context, serverConn *sshd.ServerConn, ch ssh.Channel, cancel func(), ptyReq *sshd.PtyRequestMsg, winChangeChan chan *sshd.PtyWindowChangeMsg) error {
	return fmt.Errorf("not support shell")
}

// Execute a process for the channel.
func (s *Session) Execute(ctx context.Context, serverConn *sshd.ServerConn, ch ssh.Channel, cancel func(), cmd string) error {
	c, err := shlex.Split(cmd)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, c[0], c[1:]...)
	command.Env = serverConn.Environ
	command.Dir = serverConn.Dir
	command.Stdout = ch
	command.Stdin = ch
	command.Stderr = ch.Stderr()

	err = command.Start()
	if err != nil {
		return err
	}
	go func() {
		command.Wait()
		cancel()
	}()
	return nil
}
