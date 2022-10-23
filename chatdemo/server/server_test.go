package server

import (
	"encoding/binary"
	"testing"
)

func TestNewServer(t *testing.T) {
	// golang
	a := uint32(0x01020304)
	arr := make([]byte, 4)
	binary.BigEndian.PutUint32(arr, a)
	t.Log(arr) //[1 2 3 4]

	binary.LittleEndian.PutUint32(arr, a)
	t.Log(arr) //[4 3 2 1]

	var pkt = struct {
		Source   uint32
		Sequence uint64
		Data     []byte
	}{
		Source:   257,
		Sequence: 5,
		Data:     []byte("hello world"),
	}

	// 为了方便观看，使用大端序
	endian := binary.BigEndian

	buf := make([]byte, 1024) // buffer
	i := 0
	endian.PutUint32(buf[i:i+4], pkt.Source)
	i += 4
	endian.PutUint64(buf[i:i+8], pkt.Sequence)
	i += 8
	// 由于data长度不确定，必须先把长度写入buf, 这样在反序列化时就可以正确的解析出data
	dataLen := len(pkt.Data)
	endian.PutUint32(buf[i:i+4], uint32(dataLen))
	i += 4

	// 写入数据data
	copy(buf[i:i+dataLen], pkt.Data)
	i += dataLen
	t.Log(buf[0:i])
	t.Log("length", i)

}
