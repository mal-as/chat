package user

import "net"

type User struct {
	Nick string
	Addr net.Addr
	Conn net.Conn
}
