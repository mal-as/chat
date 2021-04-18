package mem

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/mal-as/chat/user"
)

var errCtxCanceled = errors.New("context is canceled")
var errUserNotFound = errors.New("user not found")

type Storage struct {
	sync.RWMutex
	nameAddrMap map[string]net.Addr
	addrNameMap map[string]string
}

func NewStorage() *Storage {
	return &Storage{
		nameAddrMap: make(map[string]net.Addr),
		addrNameMap: make(map[string]string),
	}
}

func (s *Storage) UserByName(ctx context.Context, name string) (*user.User, error) {
	select {
	case <-ctx.Done():
		return nil, errCtxCanceled
	default:
		s.RLock()
		addr, ok := s.nameAddrMap[name]
		s.RUnlock()

		if !ok {
			return nil, errUserNotFound
		}

		return &user.User{Nick: name, Addr: addr}, nil
	}
}

func (s *Storage) UserByAddr(ctx context.Context, addr net.Addr) (*user.User, error) {
	select {
	case <-ctx.Done():
		return nil, errCtxCanceled
	default:
		s.RLock()
		name, ok := s.addrNameMap[addr.String()]
		s.RUnlock()

		if !ok {
			return nil, errUserNotFound
		}

		return &user.User{Nick: name, Addr: addr}, nil
	}
}

func (s *Storage) StoreUser(ctx context.Context, usr *user.User) error {
	select {
	case <-ctx.Done():
		return errCtxCanceled
	default:
		s.Lock()
		s.nameAddrMap[usr.Nick] = usr.Addr
		s.addrNameMap[usr.Addr.String()] = usr.Nick
		s.Unlock()
		return nil
	}
}
