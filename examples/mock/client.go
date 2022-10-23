package mock

import (
	"context"
	"net"
	"time"

	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/pkg/logger"
	"github.com/JellyTony/goim/transport/tcp"
	"github.com/JellyTony/goim/transport/websocket"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type ClientDemo struct {
}

//入口方法
func (c *ClientDemo) Start(userID, protocol, addr string) {
	var cli goim.Client

	// step1: 初始化客户端
	if protocol == "ws" {
		cli = websocket.NewClient(userID, "client", websocket.ClientOptions{})
		// set dialer
		cli.SetDialer(&WebsocketDialer{})
	} else if protocol == "tcp" {
		cli = tcp.NewClient("test1", "client", tcp.ClientOptions{})
		cli.SetDialer(&TCPDialer{})
	}

	// step2: 建立连接
	err := cli.Connect(addr)
	if err != nil {
		logger.Error(err)
	}
	count := 10
	go func() {
		// step3: 发送消息然后退出
		for i := 0; i < count; i++ {
			err = cli.Send([]byte("hello"))
			if err != nil {
				logger.Error(err)
				return
			}
			time.Sleep(time.Second)
		}
	}()

	// step4: 接收消息
	recv := 0
	for {
		frame, err := cli.Read()
		if err != nil {
			logger.Info(err)
			break
		}
		if frame.GetOpCode() != goim.OpBinary {
			continue
		}
		recv++
		logger.Warnf("%s receive message [%s]", cli.ID(), frame.GetPayload())
		if recv == count { // 接收完消息
			break
		}
	}

	//退出
	cli.Close()
}

// WebsocketDialer WebsocketDialer
type WebsocketDialer struct {
	userID string
}

// DialAndHandshake DialAndHandshake
func (d *WebsocketDialer) DialAndHandshake(ctx goim.DialerContext) (net.Conn, error) {
	// 1 调用ws.Dial拨号
	conn, _, _, err := ws.Dial(context.TODO(), ctx.Address)
	if err != nil {
		return nil, err
	}
	// 2. 发送用户认证信息，示例就是userid
	err = wsutil.WriteClientBinary(conn, []byte(ctx.Id))
	if err != nil {
		return nil, err
	}
	// 3. return conn
	return conn, nil
}

type TCPDialer struct {
	userID string
}

// DialAndHandshake DialAndHandshake
func (d *TCPDialer) DialAndHandshake(ctx goim.DialerContext) (net.Conn, error) {
	logger.Info("start dial: ", ctx.Address)
	// 1 调用net.Dial拨号
	conn, err := net.DialTimeout("tcp", ctx.Address, ctx.Timeout)
	if err != nil {
		return nil, err
	}
	// 2. 发送用户认证信息，示例就是userid
	err = tcp.WriteFrame(conn, goim.OpBinary, []byte(ctx.Id))
	if err != nil {
		return nil, err
	}
	// 3. return conn
	return conn, nil
}

type ClientHandler struct {
}

// Receive default listener
func (h *ClientHandler) Receive(ag goim.Agent, payload []byte) {
	logger.Warnf("%s receive message [%s]", ag.ID(), string(payload))
}

// Disconnect default listener
func (h *ClientHandler) Disconnect(id string) error {
	logger.Warnf("disconnect %s", id)
	return nil
}
