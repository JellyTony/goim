package serv

import (
	"net"

	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/pkg/logger"
	"github.com/JellyTony/goim/pkg/pkt"
	"github.com/JellyTony/goim/transport/tcp"
	"google.golang.org/protobuf/proto"
)

type TcpDialer struct {
	ServiceId string
}

func NewDialer(serviceId string) goim.Dialer {
	return &TcpDialer{
		ServiceId: serviceId,
	}
}

func (d *TcpDialer) DialAndHandshake(ctx goim.DialerContext) (net.Conn, error) {
	// 1. 拨号建立连接
	conn, err := net.DialTimeout("tcp", ctx.Address, ctx.Timeout)
	if err != nil {
		return nil, err
	}

	req := &pkt.InnerHandshakeReq{
		ServiceId: d.ServiceId,
	}
	logger.Infof("send req: %v", req)
	// 2. 把自己的 ServiceId 发送给对象
	bts, _ := proto.Marshal(req)
	err = tcp.WriteFrame(conn, goim.OpBinary, bts)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
