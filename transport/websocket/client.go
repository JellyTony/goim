package websocket

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/pkg/logger"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// ClientOptions ClientOptions
type ClientOptions struct {
	ReadWait  time.Duration // 读超时
	WriteWait time.Duration // 写超时
	Heartbeat time.Duration // 心跳超时
}

// Client is a websocket implement of the terminal
type Client struct {
	sync.Mutex
	goim.Dialer
	once    sync.Once
	id      string
	name    string
	conn    net.Conn
	state   int32
	options ClientOptions
}

// NewClient NewClient
func NewClient(id, name string, opts ClientOptions) goim.Client {
	if opts.WriteWait == 0 {
		opts.WriteWait = goim.DefaultWriteWait
	}
	if opts.ReadWait == 0 {
		opts.ReadWait = goim.DefaultReadWait
	}
	cli := &Client{
		id:      id,
		name:    name,
		options: opts,
	}
	return cli
}

// Connect to server
func (c *Client) Connect(addr string) error {
	_, err := url.Parse(addr)
	if err != nil {
		return err
	}

	if !atomic.CompareAndSwapInt32(&c.state, 0, 1) {
		return fmt.Errorf("client has connected")
	}

	// 拨号与握手
	conn, err := c.Dialer.DialAndHandshake(goim.DialerContext{
		Id:      c.id,
		Name:    c.name,
		Address: addr,
		Timeout: goim.DefaultLoginWait,
	})
	if err != nil {
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
		return err
	}
	if conn == nil {
		return fmt.Errorf("conn is nil")
	}
	c.conn = conn

	if c.options.Heartbeat > 0 {
		go func() {
			err = c.heartbeatLoop(conn)
			if err != nil {
				logger.Error("heartbealoop stopped ", err)
			}
		}()
	}

	return nil
}

func (c *Client) heartbeatLoop(conn net.Conn) error {
	tick := time.NewTicker(c.options.Heartbeat)
	defer tick.Stop()

	// 发送一个ping的心跳包给服务端
	for range tick.C {
		if err := c.ping(conn); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ping(conn net.Conn) error {
	err := conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}
	logger.Tracef("%s send ping to server", c.id)
	return wsutil.WriteClientMessage(conn, ws.OpPing, nil)
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) SetDialer(dialer goim.Dialer) {
	c.Dialer = dialer
}

func (c *Client) Send(payload []byte) error {
	if atomic.LoadInt32(&c.state) == 0 {
		return fmt.Errorf("connection is nil")
	}

	err := c.conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}

	// 客户端消息需要使用MASK
	return wsutil.WriteClientMessage(c.conn, ws.OpBinary, payload)
}

// Read client
func (c *Client) Read() (goim.Frame, error) {
	if c.conn == nil && atomic.LoadInt32(&c.state) == 0 {
		return nil, errors.New("connection is nil")
	}
	if c.options.ReadWait > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.options.ReadWait))
	}

	frame, err := ws.ReadFrame(c.conn)
	if err != nil {
		return nil, err
	}
	if frame.Header.OpCode == ws.OpClose {
		return nil, errors.New("remote side close the channel")
	}

	return NewFrame(frame), nil
}

func (c *Client) Close() {
	c.once.Do(func() {
		if c.conn == nil {
			return
		}
		// graceful close connection
		_ = wsutil.WriteClientMessage(c.conn, ws.OpClose, nil)

		c.conn.Close()
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
	})
}
