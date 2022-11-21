package goim

import (
	"errors"

	"github.com/JellyTony/goim/pkg/pkt"
)

var (
	ErrSessionNil = errors.New("err:session nil")
)

// SessionStorage defined a session storage which provides based functions as save,delete,find a session
type SessionStorage interface {
	Add(session *pkt.Session) error
	Delete(account string, channelId string) error
	Get(channelId string) (*pkt.Session, error)
	GetLocations(account ...string) ([]*Location, error)
	GetLocation(account string, device string) (*Location, error)
}
