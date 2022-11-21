package container

import (
	"hash/crc32"

	"github.com/JellyTony/goim"
	"github.com/JellyTony/goim/pkg/pkt"
)

type HashSelector struct{}

// HashCode generated a hash code
func HashCode(key string) int {
	hash32 := crc32.NewIEEE()
	hash32.Write([]byte(key))
	return int(hash32.Sum32())
}

func (s *HashSelector) Lookup(header *pkt.Header, srvs []goim.Service) string {
	ll := len(srvs)
	code := HashCode(header.ChannelId)
	return srvs[code%ll].ServiceID()
}
