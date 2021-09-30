package sshd

// BytesPool is an interface for getting and returning temporary
// bytes for use by io.CopyBuffer.
type BytesPool interface {
	Get() []byte
	Put([]byte)
}

type Logger interface {
	Println(v ...interface{})
}
