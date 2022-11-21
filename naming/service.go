package naming

import (
	"fmt"

	"github.com/JellyTony/goim"
)

type DefaultService struct {
	Id        string
	Name      string
	Address   string
	Port      int
	Protocol  string
	Namespace string
	Tags      []string
	Metadata  map[string]string
}

func NewEntry(id, name, protocol string, address string, port int) goim.ServiceRegistration {
	return &DefaultService{
		Id:       id,
		Name:     name,
		Address:  address,
		Port:     port,
		Protocol: protocol,
	}
}

func (e *DefaultService) ServiceID() string {
	return e.Id
}

func (e *DefaultService) GetNamespace() string { return e.Namespace }

func (e *DefaultService) ServiceName() string { return e.Name }

func (e *DefaultService) PublicAddress() string { return e.Address }

func (e *DefaultService) PublicPort() int { return e.Port }

func (e *DefaultService) GetProtocol() string { return e.Protocol }

func (e *DefaultService) DialURL() string {
	if e.Protocol == "tcp" {
		return fmt.Sprintf("%s:%d", e.Address, e.Port)
	}
	return fmt.Sprintf("%s://%s:%d", e.Protocol, e.Address, e.Port)
}

func (e *DefaultService) GetTags() []string { return e.Tags }

func (e *DefaultService) GetMeta() map[string]string { return e.Metadata }

func (e *DefaultService) String() string {
	return fmt.Sprintf("Id:%s,Name:%s,Address:%s,Port:%d,Ns:%s,Tags:%v,Meta:%v", e.Id, e.Name, e.Address, e.Port, e.Namespace, e.Tags, e.Metadata)
}
