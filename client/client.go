package client

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/mal-as/chat/cmd"
)

var errCtxCanceled = errors.New("context is canceled")

type Client struct {
	conn     net.Conn
	inCh     chan []byte
	stopCh   chan struct{}
	isClosed bool
}

func New(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn, inCh: make(chan []byte, 1), stopCh: make(chan struct{}, 1)}, nil
}

func (c *Client) Start(ctx context.Context) {
	go c.scanInput(ctx)
}

func (c *Client) Stop(cancel context.CancelFunc) {
	c.isClosed = true
	cancel()

	<-c.stopCh
}

func (c *Client) readStdin(ctx context.Context) {
	buf := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Print("> ")
			data, err := buf.ReadBytes('\n')
			if err == io.EOF {
				return
			} else if err != nil {
				log.Printf("ошибка чтения из консоли: %s", err)
				continue
			}

			c.inCh <- data[:len(data)-1]
		}
	}
}

func (c *Client) readSrvConn(ctx context.Context) {
	buf := bufio.NewReader(c.conn)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := buf.ReadBytes('\n')
			if err == io.EOF {
				return
			} else if err != nil {
				log.Printf("ошибка чтения из буфера: %s", err)
				continue
			}
			fmt.Print(string(data[:len(data)-1]) + "\n> ")
		}
	}
}

func (c *Client) scanInput(ctx context.Context) {
	defer func() { c.stopCh <- struct{}{} }()

	go c.readStdin(ctx)
	go c.readSrvConn(ctx)

	for {
		select {
		case <-ctx.Done():
			c.conn.Close()
			return
		case data := <-c.inCh:
			command, arg := cmd.Parse(data)

			switch command {
			case cmd.RegisterNewUser:
				if arg == "" {
					fmt.Print("введите ваше имя\n> ")
					continue
				}
				if err := c.registerUser(ctx, arg); err != nil {
					fmt.Printf("ошибка регистрации пользователя %s: %s\n> ", arg, err)
				}
			case cmd.NewChat:
				if arg == "" {
					fmt.Print("введите имя собеседника\n> ")
					continue
				}
				c.chat(ctx, arg)
			default:
				fmt.Printf("неверная команда %s: доступны команды login и chat с аргументом <name>\n", command)
			}
		}
	}
}

func (c *Client) registerUser(ctx context.Context, name string) error {
	select {
	case <-ctx.Done():
		return errCtxCanceled
	default:
		return c.sendCmd(cmd.RegisterNewUser, name)
	}
}

func (c *Client) chat(ctx context.Context, name string) {
	if err := c.sendCmd(cmd.NewChat, name); err != nil {
		log.Println("не удалось начать чат ", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case data := <-c.inCh:
			if string(data) == cmd.CloseChat {
				if _, err := c.conn.Write([]byte(cmd.CloseChat + "\n")); err != nil {
					log.Printf("ошибка отправки данных в чат %s\n", err)
				}
				return
			}
			if _, err := c.conn.Write(append(data, '\n')); err != nil {
				log.Printf("ошибка отправки данных в чат %s\n", err)
				continue
			}
		}
	}
}

func (c *Client) sendCmd(command, name string) error {
	data := make([]byte, 0, len(command)+len(name)+3)

	data = append(data, command[:]...)
	data = append(data, ' ')
	data = append(data, name[:]...)
	data = append(data, '\n')

	n, err := c.conn.Write(data)
	if n < len(data) {
		return fmt.Errorf("количество отправленных байт %d не совпадает с нужным %d", n, len(data))
	}
	return err
}
