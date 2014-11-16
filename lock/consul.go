/*
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package lock

import (
	"encoding/json"
	"errors"
	"github.com/armon/consul-api"
	"github.com/coreos/locksmith/consul"
)

const (
	consulKeyPrefix       = "coreos.com/updateengine/rebootlock"
	consulHoldersPrefix   = consulKeyPrefix + "/holders"
	consulSemaphorePrefix = consulKeyPrefix + "/semaphore"
)

// ConsulKVLockClient is a wrapper around the consul client that provides
// simple primitives to operate on the internal semaphore and holders
// structs through consul.
type ConsulKVLockClient struct {
	client consul.ConsulClient
}

func NewConsulKVLockClient(kvc consul.ConsulClient) (client *ConsulKVLockClient, err error) {
	client = &ConsulKVLockClient{kvc}
	err = client.Init()
	return
}

// Init sets an initial copy of the semaphore if it doesn't exist yet.
func (c *ConsulKVLockClient) Init() (err error) {
	qd, _ , err := c.client.Get(consulSemaphorePrefix, nil)
        if err == nil && qd != nil {
            return
	} else {
	    sem := newSemaphore()
	    b, err := json.Marshal(sem)
	    if err != nil {
		    return err
	    }
            p := &consulapi.KVPair{Key: consulSemaphorePrefix, Value: []byte(b)}
	    _, err = c.client.Put(p,nil)
	    return err
        }
}

// Get fetches the Semaphore from consul.
func (c *ConsulKVLockClient) Get() (sem *Semaphore, err error) {
	resp, _, err := c.client.Get(consulSemaphorePrefix, nil)
	if err != nil {
		return nil, err
	}

	sem = &Semaphore{}
	err = json.Unmarshal([]byte(resp.Value), sem)
	if err != nil {
		return nil, err
	}

	sem.Index = resp.ModifyIndex

	return sem, nil
}

// Set sets a Semaphore in consul.
func (c *ConsulKVLockClient) Set(sem *Semaphore) (err error) {
	if sem == nil {
		return errors.New("cannot set nil semaphore")
	}
	b, err := json.Marshal(sem)
	if err != nil {
		return err
	}
        p := &consulapi.KVPair{Key: consulSemaphorePrefix, Value: []byte(b)}
        p.ModifyIndex = sem.Index
	_, _, err = c.client.CAS(p,nil)

	return err
}
