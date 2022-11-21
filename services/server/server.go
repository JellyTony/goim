package server

import (
	"context"

	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/container"
	"github.com/JellyTony/goim/naming"
	"github.com/JellyTony/goim/naming/consul"
	wire "github.com/JellyTony/goim/pkg"
	"github.com/JellyTony/goim/pkg/logger"
	"github.com/JellyTony/goim/services/server/conf"
	"github.com/JellyTony/goim/services/server/handler"
	"github.com/JellyTony/goim/transport/tcp"
)

type ServerStartOptions struct {
	config      string
	serviceName string
}

func RunServerStart(ctx context.Context, opts *ServerStartOptions, version string) error {
	config, err := conf.Init(opts.config)
	if err != nil {
		return err
	}
	_ = logger.Init(logger.Settings{
		Level: "trace",
	})

	// 指令路由
	r := goim.NewRouter()

	// login
	loginHandler := handler.NewLoginHandler()
	r.Handle(wire.CommandLoginSignIn, loginHandler.DoSysLogin)
	r.Handle(wire.CommandLoginSignOut, loginHandler.DoSysLogout)

	rdb, err := conf.InitRedis(config.RedisAddrs, "")
	if err != nil {
		return err
	}
	// 会话管理
	cache := storage.NewRedisStorage(rdb)
	servhandler := serv.NewServHandler(r, cache)

	service := &naming.DefaultService{
		Id:       config.ServiceID,
		Name:     opts.serviceName,
		Address:  config.PublicAddress,
		Port:     config.PublicPort,
		Protocol: string(wire.ProtocolTCP),
		Tags:     config.Tags,
	}
	srv := tcp.NewServer(config.Listen, service)

	srv.SetReadWait(goim.DefaultReadWait)
	srv.SetAcceptor(servhandler)
	srv.SetMessageListener(servhandler)
	srv.SetStateListener(servhandler)

	if err := container.Init(srv); err != nil {
		return err
	}

	ns, err := consul.NewNaming(config.ConsulURL)
	if err != nil {
		return err
	}
	container.SetServiceNaming(ns)

	return container.Start()
}
