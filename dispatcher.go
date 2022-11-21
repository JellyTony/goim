package goim

import (
	"github.com/JellyTony/goim/pkg/pkt"
)

// Dispatcher defined a component how a message be dispatched to gateway
type Dispatcher interface {
	Push(gateway string, channels []string, p *pkt.LogicPkt) error
}
