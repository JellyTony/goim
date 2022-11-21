package container

import (
	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/pkg/pkt"
)

// Selector is used to select a Service
type Selector interface {
	Lookup(*pkt.Header, []goim.Service) string
}
