package websocket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/naming"
	"github.com/JellyTony/goim/pkg/logger"
	"github.com/gobwas/ws"
	"github.com/segmentio/ksuid"
)

// ServerOptions ServerOptions
type ServerOptions struct {
	loginwait time.Duration //登录超时
	readwait  time.Duration //读超时
	writewait time.Duration //写超时
}

// Server is a websocket implement of the Server
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

// Start server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	log := logger.WithFields(logger.Fields{
		"module": "ws.server",
		"listen": s.listen,
		"id":     s.ServiceID(),
	})

	if s.Acceptor == nil {
		s.Acceptor = new(defaultAcceptor)
	}
	if s.StateListener == nil {
		return fmt.Errorf("StateListener is nil")
	}

	// 连接管理器
	if s.ChannelMap == nil {
		s.ChannelMap = goim.NewChannels(100)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// step 1
		rawconn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			resp(w, http.StatusBadRequest, err.Error())
			return
		}

		// step 2 包装conn
		conn := NewConn(rawconn)

		// step 3
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

		// step 4
		channel := goim.NewChannel(id, metadata, conn)
		channel.SetWriteWait(s.options.writewait)
		channel.SetReadWait(s.options.readwait)
		s.Add(channel)

		go func(ch goim.Channel) {
			// step 5
			err = ch.ReadMessage(s.MessageListener)
			if err != nil {
				log.Error(err)
			}
			// step 6
			s.Remove(ch.ID())
			err = s.Disconnect(ch.ID())
			if err != nil {
				log.Warn(err)
			}
			ch.Close()
		}(channel)
	})

	log.Infoln("started")
	return http.ListenAndServe(s.listen, mux)
}

// Shutdown Shutdown
func (s *Server) Shutdown(ctx context.Context) error {
	log := logger.WithFields(logger.Fields{
		"module": "ws.server",
		"id":     s.ServiceID(),
	})
	s.once.Do(func() {
		defer func() {
			log.Infoln("shutdown")
		}()
		// close channels
		c := s.ChannelMap.All()
		for _, ch := range c {
			ch.Close()

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

// string channelID
// []byte data
func (s *Server) Push(id string, data []byte) error {
	ch, ok := s.ChannelMap.Get(id)
	if !ok {
		return errors.New("channel no found")
	}
	return ch.Push(data)
}

// SetAcceptor SetAcceptor
func (s *Server) SetAcceptor(acceptor goim.Acceptor) {
	s.Acceptor = acceptor
}

// SetMessageListener SetMessageListener
func (s *Server) SetMessageListener(listener goim.MessageListener) {
	s.MessageListener = listener
}

// SetStateListener SetStateListener
func (s *Server) SetStateListener(listener goim.StateListener) {
	s.StateListener = listener
}

// SetChannels SetChannels
func (s *Server) SetChannelMap(channels goim.ChannelMap) {
	s.ChannelMap = channels
}

// SetReadWait set read wait duration
func (s *Server) SetReadWait(readwait time.Duration) {
	s.options.readwait = readwait
}

func resp(w http.ResponseWriter, code int, body string) {
	w.WriteHeader(code)
	if body != "" {
		_, _ = w.Write([]byte(body))
	}
	logger.Warnf("response with code:%d %s", code, body)
}

type defaultAcceptor struct {
}

// Accept defaultAcceptor
func (a *defaultAcceptor) Accept(conn goim.Conn, timeout time.Duration) (string, goim.Metadata, error) {
	return ksuid.New().String(), goim.Metadata{}, nil
}
