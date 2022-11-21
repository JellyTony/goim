package container

import (
	"sync"

	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/pkg/logger"
)

// ClientMap Clients
type ClientMap interface {
	Add(client goim.Client)
	Remove(id string)
	Get(id string) (client goim.Client, ok bool)
	Services(kvs ...string) []goim.Service
}

type ClientsImpl struct {
	clients *sync.Map
}

// NewClients NewClients
func NewClients(num int) ClientMap {
	return &ClientsImpl{
		clients: new(sync.Map),
	}
}

// Add addChannel
func (ch *ClientsImpl) Add(client goim.Client) {
	if client.ServiceID() == "" {
		logger.WithFields(logger.Fields{
			"module": "ClientsImpl",
		}).Error("client id is required")
	}
	ch.clients.Store(client.ServiceID(), client)
}

// Remove addChannel
func (ch *ClientsImpl) Remove(id string) {
	ch.clients.Delete(id)
}

// Get Get
func (ch *ClientsImpl) Get(id string) (goim.Client, bool) {
	if id == "" {
		logger.WithFields(logger.Fields{
			"module": "ClientsImpl",
		}).Error("client id is required")
	}

	val, ok := ch.clients.Load(id)
	if !ok {
		return nil, false
	}
	return val.(goim.Client), true
}

// Services 返回服务列表，可以传一对
func (ch *ClientsImpl) Services(kvs ...string) []goim.Service {
	kvLen := len(kvs)
	if kvLen != 0 && kvLen != 2 {
		return nil
	}
	arr := make([]goim.Service, 0)
	ch.clients.Range(func(key, val interface{}) bool {
		ser := val.(goim.Service)
		if kvLen > 0 && ser.GetMeta()[kvs[0]] != kvs[1] {
			return true
		}
		arr = append(arr, ser)
		return true
	})
	return arr
}
