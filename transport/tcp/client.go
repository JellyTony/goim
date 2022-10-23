package tcp

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/pkg/logger"
)

// ClientOptions ClientOptions
type ClientOptions struct {
	Heartbeat time.Duration //登陆超时
	ReadWait  time.Duration //读超时
	WriteWait time.Duration //写超时
}

type Client struct {
	sync.Mutex
	goim.Dialer
	once    sync.Once
	id      string
	name    string
	conn    goim.Conn
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

func (c *Client) ID() string {
	return c.id
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Connect(addr string) error {
	_, err := url.Parse(addr)
	if err != nil {
		return err
	}

	// 这里是一个CAS原子操作，对比并设置值，是并发安全的。
	if !atomic.CompareAndSwapInt32(&c.state, 0, 1) {
		return fmt.Errorf("client has connected")
	}

	rawconn, err := c.Dialer.DialAndHandshake(goim.DialerContext{
		Id:      c.id,
		Name:    c.name,
		Address: addr,
		Timeout: goim.DefaultLoginWait,
	})
	if err != nil {
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
		return err
	}

	if rawconn == nil {
		return fmt.Errorf("conn is nil")
	}
	c.conn = NewConn(rawconn)

	if c.options.Heartbeat > 0 {
		go func() {
			err := c.heartbeatLoop()
			if err != nil {
				logger.WithField("module", "tcp.client").Warn("heartbealoop stopped - ", err)
			}
		}()
	}

	return nil
}

// SetDialer 设置握手逻辑
func (c *Client) SetDialer(dialer goim.Dialer) {
	c.Dialer = dialer
}

func (c *Client) Send(payload []byte) error {
	if atomic.LoadInt32(&c.state) == 0 {
		return fmt.Errorf("connection is nil")
	}
	c.Lock()
	defer c.Unlock()
	err := c.conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}
	return c.conn.WriteFrame(goim.OpBinary, payload)
}

func (c *Client) Read() (goim.Frame, error) {
	if c.conn == nil {
		return nil, errors.New("connection is nil")
	}
	if c.options.Heartbeat > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.options.ReadWait))
	}

	frame, err := c.conn.ReadFrame()
	if err != nil {
		return nil, err
	}
	if frame.GetOpCode() == goim.OpClose {
		return nil, errors.New("remote side close the channel")
	}
	return frame, nil
}

func (c *Client) heartbeatLoop() error {
	tick := time.NewTicker(c.options.Heartbeat)
	defer tick.Stop()
	for range tick.C {
		// 发送一个ping的心跳包给服务端
		if err := c.ping(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ping() error {
	logger.WithField("module", "tcp.client").Tracef("%s send ping to server", c.id)
	err := c.conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}
	return c.conn.WriteFrame(goim.OpPing, nil)
}

func (c *Client) Close() {
	c.once.Do(func() {
		if c.conn == nil {
			return
		}
		// graceful close connection
		_ = WriteFrame(c.conn, goim.OpClose, nil)

		c.conn.Close()
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
	})
}
