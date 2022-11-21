package consul

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/naming"
	"github.com/JellyTony/goim/pkg/logger"
	"github.com/hashicorp/consul/api"
)

const (
	KeyProtocol  = "protocol"
	KeyHealthURL = "health_url"
)

type Watch struct {
	Service   string
	Callback  func([]goim.ServiceRegistration)
	WaitIndex uint64
	Quit      chan struct{}
}

type Naming struct {
	sync.RWMutex
	cli    *api.Client
	watchs map[string]*Watch
}

func NewNaming(address string) (*Naming, error) {
	config := api.DefaultConfig()
	config.Address = address
	cli, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &Naming{
		cli:    cli,
		watchs: make(map[string]*Watch, 1),
	}, nil
}

func (n *Naming) Register(s goim.ServiceRegistration) error {
	reg := &api.AgentServiceRegistration{
		ID:      s.ServiceID(),
		Name:    s.ServiceName(),
		Address: s.PublicAddress(),
		Port:    s.PublicPort(),
		Tags:    s.GetTags(),
		Meta:    s.GetMeta(),
	}
	if reg.Meta == nil {
		reg.Meta = make(map[string]string)
	}
	reg.Meta[KeyProtocol] = s.GetProtocol()

	// consul健康检查
	healthURL := s.GetMeta()[KeyHealthURL]
	if healthURL != "" {
		check := new(api.AgentServiceCheck)
		check.CheckID = fmt.Sprintf("%s_normal", s.ServiceID())
		check.HTTP = healthURL
		check.Timeout = "1s" // http timeout
		check.Interval = "10s"
		check.DeregisterCriticalServiceAfter = "20s"
		reg.Check = check
	}

	err := n.cli.Agent().ServiceRegister(reg)
	return err
}

func (n *Naming) Deregister(serviceID string) error {
	return n.cli.Agent().ServiceDeregister(serviceID)
}

func (n *Naming) Find(name string, tags ...string) ([]goim.ServiceRegistration, error) {
	services, _, err := n.load(name, 0, tags...)
	if err != nil {
		return nil, err
	}
	return services, nil
}

func (n *Naming) load(name string, waitIndex uint64, tags ...string) ([]goim.ServiceRegistration, *api.QueryMeta, error) {
	opts := &api.QueryOptions{
		UseCache:  true,
		MaxAge:    time.Minute, // MaxAge limits how old a cached value will be returned if UseCache is true.
		WaitIndex: waitIndex,
	}
	catalogServices, meta, err := n.cli.Catalog().ServiceMultipleTags(name, tags, opts)
	if err != nil {
		return nil, meta, err
	}

	services := make([]goim.ServiceRegistration, 0, len(catalogServices))
	for _, s := range catalogServices {
		if s.Checks.AggregatedStatus() != api.HealthPassing {
			logger.Debugf("load service: id:%s name:%s %s:%d Status:%s", s.ServiceID, s.ServiceName, s.ServiceAddress, s.ServicePort, s.Checks.AggregatedStatus())
			continue
		}
		services = append(services, &naming.DefaultService{
			Id:       s.ServiceID,
			Name:     s.ServiceName,
			Address:  s.ServiceAddress,
			Port:     s.ServicePort,
			Protocol: s.ServiceMeta[KeyProtocol],
			Tags:     s.ServiceTags,
			Metadata: s.ServiceMeta,
		})
	}
	logger.Debugf("load service: %v, meta:%v", services, meta)
	return services, meta, nil
}

func (n *Naming) Subscribe(serviceName string, callback func([]goim.ServiceRegistration)) error {
	n.Lock()
	defer n.Unlock()
	if _, ok := n.watchs[serviceName]; ok {
		return errors.New("serviceName has already been registered")
	}

	w := &Watch{
		Service:  serviceName,
		Callback: callback,
		Quit:     make(chan struct{}, 1),
	}

	n.watchs[serviceName] = w

	go n.watch(w)
	return nil
}

func (n *Naming) watch(wh *Watch) {
	stopped := false
	var doWatch = func(service string, callback func([]goim.ServiceRegistration)) {
		services, meta, err := n.load(service, wh.WaitIndex)
		if err != nil {
			logger.Warn(err)
			return
		}
		select {
		case <-wh.Quit:
			stopped = true
			logger.Infof("watch %s stopped", wh.Service)
			return
		default:
		}

		wh.WaitIndex = meta.LastIndex
		if callback != nil {
			callback(services)
		}
	}

	// build WaitIndex
	doWatch(wh.Service, nil)
	for !stopped {
		doWatch(wh.Service, wh.Callback)
	}
}

func (n *Naming) Unsubscribe(serviceName string) error {
	n.Lock()
	defer n.Unlock()
	wh, ok := n.watchs[serviceName]

	delete(n.watchs, serviceName)
	if ok {
		close(wh.Quit)
	}
	return nil
}
