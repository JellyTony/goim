package websocket

import (
	"net"

	"github.com/JellyTony/goim"
	"github.com/gobwas/ws"
)

// WsConn WebSocket连接
type WsConn struct {
	net.Conn
}

// NewConn 创建连接
func NewConn(conn net.Conn) *WsConn {
	return &WsConn{
		Conn: conn,
	}
}

// ReadFrame 读取帧
func (c *WsConn) ReadFrame() (goim.Frame, error) {
	f, err := ws.ReadFrame(c.Conn)
	if err != nil {
		return nil, err
	}

	return NewFrame(f), nil
}

// WriteFrame 写入帧
func (c *WsConn) WriteFrame(code goim.OpCode, payload []byte) error {
	f := ws.NewFrame(ws.OpCode(code), true, payload)
	return ws.WriteFrame(c.Conn, f)
}

func (c *WsConn) Flush() error {
	return nil
}
