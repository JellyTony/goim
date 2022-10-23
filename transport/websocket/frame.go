package websocket

import (
	"github.com/JellyTony/goim"
	"github.com/gobwas/ws"
)

// Frame 帧协议
type Frame struct {
	raw ws.Frame
}

// NewFrame 创建帧
func NewFrame(raw ws.Frame) *Frame {
	return &Frame{raw: raw}
}

// SetOpCode 设置帧类型
func (f *Frame) SetOpCode(op goim.OpCode) {
	f.raw.Header.OpCode = ws.OpCode(op)
}

// GetOpCode 获取帧类型
func (f *Frame) GetOpCode() goim.OpCode {
	return goim.OpCode(f.raw.Header.OpCode)
}

// SetPayload 设置帧数据
func (f *Frame) SetPayload(payload []byte) {
	f.raw.Payload = payload
}

// GetPayload 获取帧数据
func (f *Frame) GetPayload() []byte {
	if f.raw.Header.Masked {
		ws.Cipher(f.raw.Payload, f.raw.Header.Mask, 0)
	}
	f.raw.Header.Masked = false
	return f.raw.Payload
}
