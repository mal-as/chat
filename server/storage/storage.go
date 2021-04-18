package storage

import (
	"context"
	"net"

	"github.com/mal-as/chat/user"
)

type Storage interface {
	UserByName(context.Context, string) (*user.User, error)
	UserByAddr(context.Context, net.Addr) (*user.User, error)
	StoreUser(context.Context, *user.User) error
}
