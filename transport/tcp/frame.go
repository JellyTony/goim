package tcp

import (
	"github.com/JellyTony/goim"
)

// Frame Frame
type Frame struct {
	OpCode  goim.OpCode
	Payload []byte
}

// SetOpCode SetOpCode
func (f *Frame) SetOpCode(code goim.OpCode) {
	f.OpCode = code
}

// GetOpCode GetOpCode
func (f *Frame) GetOpCode() goim.OpCode {
	return f.OpCode
}

// SetPayload SetPayload
func (f *Frame) SetPayload(payload []byte) {
	f.Payload = payload
}

// GetPayload GetPayload
func (f *Frame) GetPayload() []byte {
	return f.Payload
}
