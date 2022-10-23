package goim

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JellyTony/goim/pkg/logger"
)

var (
	ErrChannelClosed = errors.New("channel has closed")
)

// ChannelImpl is a websocket/tcp implement of channel
type ChannelImpl struct {
	id        string
	metadata  Metadata
	writechan chan []byte
	once      sync.Once
	readWait  time.Duration
	writeWait time.Duration
	state     int32 // 0 init 1 start 2 closed

	Conn
	sync.Mutex
}

func NewChannel(id string, metadata Metadata, conn Conn) Channel {
	log := logger.WithFields(logger.Fields{
		"module": "channel",
		"id":     id,
	})
	ch := &ChannelImpl{
		id:        id,
		Conn:      conn,
		metadata:  metadata,
		readWait:  DefaultReadWait,
		writeWait: DefaultWriteWait,
		writechan: make(chan []byte, 5),
	}
	go func() {
		err := ch.writeLoop()
		if err != nil {
			log.Info(err)
		}
	}()
	return ch
}

func (c *ChannelImpl) ReadMessage(lst MessageListener) error {
	if !atomic.CompareAndSwapInt32(&c.state, 0, 1) {
		return fmt.Errorf("channel has started")
	}

	log := logger.WithFields(logger.Fields{
		"struct": "ChannelImpl",
		"func":   "Readloop",
		"id":     c.id,
	})

	for {
		_ = c.SetReadDeadline(time.Now().Add(c.readWait))
		frame, err := c.ReadFrame()
		if err != nil {
			return err
		}
		if frame.GetOpCode() == OpClose {
			return errors.New("remote side close the channel")
		}
		if frame.GetOpCode() == OpPing {
			log.Trace("recv a ping; resp with a pong")
			_ = c.WriteFrame(OpPong, nil)
			continue
		}

		payload := frame.GetPayload()
		if len(payload) == 0 {
			continue
		}
		go lst.Receive(c, payload)
	}
}

func (c *ChannelImpl) writeLoop() error {
	for {
		select {
		case payload, ok := <-c.writechan:
			if !ok {
				return ErrChannelClosed
			}

			err := c.WriteFrame(OpBinary, payload)
			if err != nil {
				return err
			}

			// 批量写
			chanLen := len(c.writechan)
			for i := 0; i < chanLen; i++ {
				payload = <-c.writechan
				err = c.WriteFrame(OpBinary, payload)
				if err != nil {
					return err
				}
			}

			err = c.Conn.Flush()
			if err != nil {
				return err
			}
		}
	}
}

func (c *ChannelImpl) Push(payload []byte) error {
	if atomic.LoadInt32(&c.state) != 1 {
		return fmt.Errorf("channel %s has closed", c.id)
	}

	c.writechan <- payload
	return nil
}

// WriteFrame 写入帧数据
func (c *ChannelImpl) WriteFrame(code OpCode, payload []byte) error {
	_ = c.Conn.SetWriteDeadline(time.Now().Add(c.writeWait))
	return c.Conn.WriteFrame(code, payload)
}

// ID id simpling server
func (c *ChannelImpl) ID() string { return c.id }

func (c *ChannelImpl) GetMetadata() Metadata {
	return c.metadata
}

// SetWriteWait 设置写超时
func (c *ChannelImpl) SetWriteWait(writeWait time.Duration) {
	if writeWait == 0 {
		return
	}
	c.writeWait = writeWait
}

func (c *ChannelImpl) SetReadWait(readwait time.Duration) {
	if readwait == 0 {
		return
	}
	c.writeWait = readwait
}

func (c *ChannelImpl) Close() error {
	if !atomic.CompareAndSwapInt32(&c.state, 1, 2) {
		return fmt.Errorf("channel has closed")
	}
	close(c.writechan)
	return nil
}
