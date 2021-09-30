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
type Session struct {
	NewChan ssh.NewChannel
	// Logger error log
	Logger sshd.Logger
	// Default environment
	Environ []string
	// Default workdir
	Dir string
}

func (s *Session) Handle(ctx context.Context) {
	ch, reqs, err := s.NewChan.Accept()
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("unable to accept NewChan:", err)
		}
		return
	}
	defer func() {
		b := ssh.Marshal(sshd.ExitStatusMsg{})
		ch.SendRequest("exit-status", false, b)
		ch.Close()
	}()

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
			switch req.Type {
			case "pty-req":
				ptyreq := &sshd.PtyRequestMsg{}
				if err := ssh.Unmarshal(req.Payload, ptyreq); err != nil {
					if s.Logger != nil {
						s.Logger.Println("error unmarshalling pty-req:", err)
					}
					return
				}
				ptyReq = ptyreq
				winChangeChan = make(chan *sshd.PtyWindowChangeMsg, 1)
				s.Setenv("TERM", ptyreq.Term)
			case "window-change":
				winchangereq := &sshd.PtyWindowChangeMsg{}
				if err := ssh.Unmarshal(req.Payload, winchangereq); err != nil {
					if s.Logger != nil {
						s.Logger.Println("error unmarshalling window-change:", err)
					}
					return
				}
				winChangeChan <- winchangereq
			case "env":
				envreq := &sshd.SetenvRequest{}
				if err := ssh.Unmarshal(req.Payload, envreq); err != nil {
					if s.Logger != nil {
						s.Logger.Println("error unmarshalling env:", err)
					}
					return
				}
				s.Setenv(envreq.Name, envreq.Value)
			case "shell":
				err := s.Shell(ctx, ch, cancel, ptyReq, winChangeChan)
				if err != nil {
					if s.Logger != nil {
						s.Logger.Println("error execute:", err)
					}
					sess = false
				}
			case "exec":
				execReq := &sshd.ExecMsg{}
				if err := ssh.Unmarshal(req.Payload, execReq); err != nil {
					if s.Logger != nil {
						s.Logger.Println("error unmarshalling exec:", err)
					}
					return
				}
				err := s.Execute(ctx, ch, cancel, execReq.Command)
				if err != nil {
					if s.Logger != nil {
						s.Logger.Println("error execute:", err)
					}
					sess = false
				}
			default:
				if s.Logger != nil {
					s.Logger.Println("unknown session request:", req.Type)
				}
				sess = false
			}
			if req.WantReply {
				req.Reply(sess, nil)
			}
		}
	}
}

func (s *Session) Setenv(key, val string) {
	for i, env := range s.Environ {
		se := strings.SplitN(env, "=", 2)
		if len(se) == 2 {
			if se[0] == key {
				s.Environ[i] = fmt.Sprintf("%s=%s", key, val)
				return
			}
		}
	}
	s.Environ = append(s.Environ, fmt.Sprintf("%s=%s", key, val))
}

// Shell a process for the channel.
func (s *Session) Shell(ctx context.Context, ch ssh.Channel, cancel func(), ptyReq *sshd.PtyRequestMsg, winChangeChan chan *sshd.PtyWindowChangeMsg) error {
	return fmt.Errorf("not support shell")
}

// Execute a process for the channel.
func (s *Session) Execute(ctx context.Context, ch ssh.Channel, cancel func(), cmd string) error {
	c, err := shlex.Split(cmd)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, c[0], c[1:]...)
	command.Env = s.Environ
	command.Dir = s.Dir
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
