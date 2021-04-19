package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/mal-as/chat/cmd"
	"github.com/mal-as/chat/server/storage"
	"github.com/mal-as/chat/user"
)

type Server struct {
	db          storage.Storage
	listener    net.Listener
	stopCh      chan struct{}
	wg          sync.WaitGroup
	activeConns map[net.Addr]net.Conn
	isClosed    bool
}

func New(addr string, db storage.Storage) (*Server, error) {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	srv := &Server{listener: listen, db: db, stopCh: make(chan struct{}, 1), activeConns: make(map[net.Addr]net.Conn)}

	return srv, nil
}

func (s *Server) Start(ctx context.Context) {
	go s.serve(ctx)
}

func (s *Server) Stop(cancel context.CancelFunc) {
	cancel()

	<-s.stopCh
}

func (s *Server) serve(ctx context.Context) {
	defer func() { s.stopCh <- struct{}{} }()

	connCh := make(chan net.Conn)

	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				if s.isClosed {
					return
				}
				log.Printf("ошибка установления нового соединения: %s", err)
				continue
			}
			connCh <- conn
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("контест закрыт, ожидание завершения всех горутин...")
			s.wg.Wait()
			s.isClosed = true
			s.listener.Close()
			return
		case conn := <-connCh:
			s.wg.Add(1)
			go s.handleConn(ctx, conn)
		}
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()
	defer delete(s.activeConns, conn.RemoteAddr())

	s.activeConns[conn.RemoteAddr()] = conn
	log.Printf("установлено новое соединение с %s\n", conn.RemoteAddr())
	buf := bufio.NewReader(conn)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := buf.ReadBytes('\n')
			if err == io.EOF {
				log.Printf("соединение с %s закрыто\n", conn.RemoteAddr())
				return
			} else if err != nil {
				log.Printf("ошибка чтения из соединения %s: %s", conn.RemoteAddr(), err)
				continue
			}

			command, arg := cmd.Parse(data)

			if arg == "" {
				s.writeToConn(conn, "отсутствует аргумент для команды")
				continue
			}

			switch command {
			case cmd.RegisterNewUser:
				if err = s.addUser(ctx, conn.RemoteAddr(), arg); err != nil {
					s.writeToConn(conn, "ошибка регистрации пользователя")
				} else {
					s.writeToConn(conn, fmt.Sprintf("пользователь %s успешно зарегестрирован", arg))
				}
			case cmd.NewChat:
				s.newChat(ctx, conn, arg)
				s.writeToConn(conn, "чат завершен")
			default:
				log.Print(arg)
			}
		}
	}
}

func (s *Server) addUser(ctx context.Context, addr net.Addr, userData string) error {
	usr := &user.User{Nick: userData, Addr: addr}
	if err := s.db.StoreUser(ctx, usr); err != nil {
		return err
	}

	return nil
}

func (s *Server) newChat(ctx context.Context, conn net.Conn, name string) {
	you, err := s.db.UserByAddr(ctx, conn.RemoteAddr())
	if err != nil {
		s.writeToConn(conn, "похоже, что вы не зарегестрированы, создайте аккаунт, если он уже есть - попробуйте позже")
		return
	}
	you.Conn = conn

	contact, err := s.db.UserByName(ctx, string(name))
	if err != nil {
		s.writeToConn(conn, fmt.Sprintf("ошибка поиска аккаунта %s", string(name)))
		return
	}
	var ok bool
	contact.Conn, ok = s.activeConns[contact.Addr]
	if !ok {
		s.writeToConn(conn, fmt.Sprintf("отсутствует соединение с аккаунтом %s", string(name)))
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go s.chatRoom(ctx, you, contact, &wg)
	wg.Wait()
}

func (s *Server) chatRoom(ctx context.Context, you, contact *user.User, wg *sync.WaitGroup) {
	defer wg.Done()

	buf := bufio.NewReader(you.Conn)

	you.Conn.Write([]byte(fmt.Sprintf("начат чат с %s\n", contact.Nick)))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			text, err := buf.ReadString('\n')
			if err != nil && err == io.EOF {
				return
			} else if err != nil {
				log.Printf("ошибка чтения: %s\n", err)
			}

			text = strings.TrimSuffix(text, "\n")

			if text == cmd.CloseChat {
				return
			}

			_, err = contact.Conn.Write([]byte(you.Nick + ": " + text + "\n"))
			if err != nil {
				log.Printf("ошибка записи: %s\n", err)
			}
		}
	}
}

func (s *Server) writeToConn(conn net.Conn, msg string) {
	if _, err := conn.Write([]byte(msg + "\n")); err != nil {
		log.Printf("ошибка записи в соединение %s: %s", conn.RemoteAddr(), err)
	}
}
