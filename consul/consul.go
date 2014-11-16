package consul

import (
	"github.com/armon/consul-api"
        "net/http"
)

// ConsulClient is a simple wrapper around the underlying consul client to facilitate
// testing and minimize the interface between locksmith and the actual consul client
type ConsulClient interface {
        Get(key string, q *consulapi.QueryOptions) (*consulapi.KVPair, *consulapi.QueryMeta, error)
        Put(p *consulapi.KVPair, q *consulapi.WriteOptions) (*consulapi.WriteMeta, error)
        CAS(p *consulapi.KVPair, q *consulapi.WriteOptions) (bool, *consulapi.WriteMeta, error)
}

const (
	ErrorKeyNotFound = 100
	ErrorNodeExist   = 105
)

// NewClient creates a new ConsulClient
func NewClient(Address string) (*consulapi.KV, error) {
	config := &consulapi.Config{
		Address:    Address,
		Scheme:     "http",
		HttpClient: http.DefaultClient,
	}

        client, _ := consulapi.NewClient(config)
        kv := client.KV()
        return kv,nil

}
