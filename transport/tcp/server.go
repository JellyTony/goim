package tcp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/naming"
	"github.com/JellyTony/goim/pkg/logger"
	"github.com/segmentio/ksuid"
)

// ServerOptions ServerOptions
type ServerOptions struct {
	loginwait time.Duration //登陆超时
	readwait  time.Duration //读超时
	writewait time.Duration //写超时
}

type Server struct {
	listen string
	naming.ServiceRegistration
	goim.ChannelMap
	goim.Acceptor
	goim.MessageListener
	goim.StateListener
	once    sync.Once
	options ServerOptions
}

// NewServer NewServer
func NewServer(listen string, service naming.ServiceRegistration) goim.Server {
	return &Server{
		listen:              listen,
		ServiceRegistration: service,
		options: ServerOptions{
			loginwait: goim.DefaultLoginWait,
			readwait:  goim.DefaultReadWait,
			writewait: time.Second * 10,
		},
	}
}

func (s *Server) SetAcceptor(acceptor goim.Acceptor) {
	s.Acceptor = acceptor
}

func (s *Server) SetMessageListener(listener goim.MessageListener) {
	s.MessageListener = listener
}

func (s *Server) SetStateListener(listener goim.StateListener) {
	s.StateListener = listener
}

func (s *Server) SetReadWait(duration time.Duration) {
	s.options.readwait = duration
}

func (s *Server) SetChannelMap(channelMap goim.ChannelMap) {
	s.ChannelMap = channelMap
}

func (s *Server) Start() error {
	log := logger.WithFields(logger.Fields{
		"module": "tcp.server",
		"listen": s.listen,
		"id":     s.ServiceID(),
	})

	if s.StateListener == nil {
		return fmt.Errorf("StateListener is nil")
	}
	if s.Acceptor == nil {
		s.Acceptor = new(defaultAcceptor)
	}

	// 连接管理器
	if s.ChannelMap == nil {
		s.ChannelMap = goim.NewChannels(100)
	}

	// 创建监听器
	listen, err := net.Listen("tcp", s.listen)
	if err != nil {
		return err
	}

	log.Info("start tcp server")

	for {
		// 等待连接
		rawconn, err := listen.Accept()
		if err != nil {
			rawconn.Close()
			log.Warn(err)
			continue
		}

		go func() {
			conn := NewConn(rawconn)

			id, metadata, err := s.Accept(conn, s.options.loginwait)
			if err != nil {
				_ = conn.WriteFrame(goim.OpClose, []byte(err.Error()))
				conn.Close()
				return
			}

			if _, ok := s.Get(id); ok {
				log.Warnf("channel %s existed", id)
				_ = conn.WriteFrame(goim.OpClose, []byte("channelId is repeated"))
				conn.Close()
				return
			}

			channel := goim.NewChannel(id, metadata, conn)
			channel.SetReadWait(s.options.readwait)
			channel.SetWriteWait(s.options.writewait)

			s.Add(channel)

			log.Info("accept ", channel)
			err = channel.ReadMessage(s.MessageListener)
			if err != nil {
				log.Info(err)
			}
			s.Remove(channel.ID())
			_ = s.Disconnect(channel.ID())
			channel.Close()
		}()
	}
}

// Push push message to channel
func (s *Server) Push(id string, payload []byte) error {
	c, ok := s.ChannelMap.Get(id)
	if !ok {
		return goim.ErrChannelNotFound
	}

	return c.Push(payload)
}

func (s *Server) Shutdown(ctx context.Context) error {
	log := logger.WithFields(logger.Fields{
		"module": "tcp.server",
		"id":     s.ServiceID(),
	})

	s.once.Do(func() {
		log.Info("shutdown tcp server")
		// close channels
		chanels := s.ChannelMap.All()
		for _, c := range chanels {
			c.Close()

			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
	})

	return nil
}

type defaultAcceptor struct {
}

// Accept defaultAcceptor
func (a *defaultAcceptor) Accept(conn goim.Conn, timeout time.Duration) (string, goim.Metadata, error) {
	return ksuid.New().String(), goim.Metadata{}, nil
}
