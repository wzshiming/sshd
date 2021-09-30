package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/wzshiming/sshd/directtcp"
	_ "github.com/wzshiming/sshd/session"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

var address string
var username string
var password string
var authorized string
var hostkey string

func init() {
	flag.StringVar(&address, "a", ":22", "listen on the address")
	flag.StringVar(&username, "u", "", "username")
	flag.StringVar(&password, "p", "", "password")
	flag.StringVar(&authorized, "f", "", "authorized file")
	flag.StringVar(&hostkey, "h", "", "hostkey file")
	flag.Parse()
}

func main() {
	logger := log.New(os.Stderr, "[sshd] ", log.LstdFlags)
	svc := sshd.NewServer()
	svc.Logger = logger
	if hostkey != "" {
		key, err := os.ReadFile(hostkey)
		if err != nil {
			logger.Println(err)
			return
		}
		err = svc.AddHostkey(key)
		if err != nil {
			logger.Println(err)
			return
		}
	} else {
		err := svc.RandomHostkey()
		if err != nil {
			logger.Println(err)
			return
		}
	}
	if username != "" {
		svc.ServerConfig.PasswordCallback = func(conn ssh.ConnMetadata, pwd []byte) (*ssh.Permissions, error) {
			if conn.User() == username && password == string(pwd) {
				return nil, nil
			}
			return nil, fmt.Errorf("denied")
		}
	}
	if authorized != "" {
		keys, err := sshd.GetAuthorizedFile(authorized)
		if err != nil {
			logger.Println(err)
			return
		}
		svc.ServerConfig.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			k := string(key.Marshal())
			if _, ok := keys[k]; ok {
				return nil, nil
			}
			return nil, fmt.Errorf("denied")
		}
	}
	if username == "" && authorized == "" {
		svc.ServerConfig.NoClientAuth = true
	}
	err := svc.ListenAndServe("tcp", address)
	if err != nil {
		logger.Println(err)
	}
}
