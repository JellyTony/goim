package goim

import (
	"context"
	"net"
	"time"
)

var (
	DefaultReadWait  = time.Minute * 3
	DefaultWriteWait = time.Second * 10
	DefaultLoginWait = time.Second * 10
	DefaultHeartbeat = time.Second * 55
)

var (
	// 定义读取消息的默认goroutine池大小
	DefaultMessageReadPool = 5000
	DefaultConnectionPool  = 5000
)

// OpCode OpCode
type OpCode byte

const (
	OpContinuation OpCode = 0x0
	OpText         OpCode = 0x1
	OpBinary       OpCode = 0x2
	OpClose        OpCode = 0x8
	OpPing         OpCode = 0x9
	OpPong         OpCode = 0xa
)

// Conn Connection
type Conn interface {
	net.Conn
	ReadFrame() (Frame, error)
	WriteFrame(OpCode, []byte) error
	Flush() error
}

// Acceptor
type Acceptor interface {
	Accept(Conn, time.Duration) (string, Metadata, error)
}

type StateListener interface {
	Disconnect(string) error
}

// Channel is interface of client side
type Channel interface {
	Conn
	Agent
	// ReadMessage 读取消息
	ReadMessage(lst MessageListener) error
	// SetWriteWait 设置写超时
	SetWriteWait(time.Duration)
	// SetReadWait 设置读超时
	SetReadWait(time.Duration)
	// Close 关闭连接
	Close() error
}

type Metadata map[string]string

type Agent interface {
	// ID 返回通道ID
	ID() string
	// Push 推送消息
	Push([]byte) error
	GetMetadata() Metadata
}

type MessageListener interface {
	Receive(Agent, []byte)
}

type Frame interface {
	SetOpCode(OpCode)
	GetOpCode() OpCode
	SetPayload([]byte)
	GetPayload() []byte
}

// Service 定义了基础服务的抽象接口
type Service interface {
	ServiceID() string
	ServiceName() string
	GetMeta() map[string]string
}

// ServiceRegistration 定义服务注册的抽象接口
type ServiceRegistration interface {
	Service
	PublicAddress() string
	PublicPort() int
	DialURL() string
	GetTags() []string
	GetProtocol() string
	GetNamespace() string
	String() string
}

// Server 定义了一个tcp/websocket不同协议通用的服务端的接口
type Server interface {
	ServiceRegistration
	// SetAcceptor 设置Acceptor
	SetAcceptor(Acceptor)
	//SetMessageListener 设置上行消息监听器
	SetMessageListener(MessageListener)
	//SetStateListener 设置连接状态监听服务
	SetStateListener(StateListener)
	// SetReadWait 设置读超时
	SetReadWait(time.Duration)
	// ChannelMap 设置Channel管理服务
	SetChannelMap(ChannelMap)

	// Start 用于在内部实现网络端口的监听和接收连接，
	// 并完成一个Channel的初始化过程。
	Start() error
	Push(string, []byte) error
	// Shutdown 服务下线，关闭连接
	Shutdown(context.Context) error
}

// DialerContext 拨号参数
type DialerContext struct {
	Id      string
	Name    string
	Address string
	Timeout time.Duration
}

// Dialer 拨号抽象
type Dialer interface {
	// DialAndHandshake 拨号并握手
	DialAndHandshake(DialerContext) (net.Conn, error)
}

// Client is interface of client side
type Client interface {
	Service
	Connect(string) error
	SetDialer(Dialer)
	Send([]byte) error
	Read() (Frame, error)
	Close()
}
