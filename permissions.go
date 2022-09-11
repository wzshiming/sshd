package sshd

// Permissions specifies the permissions that the user has
type Permissions interface {
	Allow(req string, args string) bool
}
