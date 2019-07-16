// Copyright 2019 shimingyah. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// ee the License for the specific language governing permissions and
// limitations under the License.

package pool

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
)

// ErrClosed is the error resulting if the pool is closed via pool.Close().
var ErrClosed = errors.New("pool is closed")

// Pool interface describes a pool implementation.
// An ideal pool is threadsafe and easy to use.
type Pool interface {
	// Get returns a new connection from the pool. Closing the connections puts
	// it back to the Pool. Closing it when the pool is destroyed or full will
	// be counted as an error.
	Get() (Conn, error)

	// Close closes the pool and all its connections. After Close() the pool is
	// no longer usable.
	Close() error

	// String returns the current status of the pool.
	String() string
}

type pool struct {
	// atomic, used to get connection random
	index uint32

	// atomic, the current physical connection of pool
	current int32

	// atomic, the using logic connection of pool
	// logic connection = physical connection * MaxConcurrentStreams
	ref int32

	// pool options
	opt Options

	// all of created physical connections
	conns []*conn

	// the server address is to create connection.
	address string

	sync.RWMutex
}

// New return a connection pool.
func New(address string, option Options) (Pool, error) {
	if address == "" {
		return nil, errors.New("invalid address settings")
	}
	if option.Dial == nil {
		return nil, errors.New("invalid dial settings")
	}
	if option.MaxIdle < 0 || option.MaxActive <= 0 || option.MaxIdle > option.MaxActive {
		return nil, errors.New("invalid maximum settings")
	}
	if option.MaxConcurrentStreams <= 0 {
		return nil, errors.New("invalid maximun settings")
	}

	p := &pool{
		index:   0,
		current: int32(option.MaxIdle),
		ref:     0,
		opt:     option,
		conns:   make([]*conn, option.MaxActive),
		address: address,
	}

	for i := 0; i < p.opt.MaxIdle; i++ {
		c, err := p.opt.Dial(address)
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("dial is not able to fill the pool: %s", err)
		}
		p.conns[i] = p.wrapConn(c)
	}

	return p, nil
}

func (p *pool) incrRef() int32 {
	newRef := atomic.AddInt32(&p.ref, 1)
	if newRef == math.MaxInt32 {
		panic(fmt.Sprintf("overflow ref: %d", newRef))
	}
	return newRef
}

func (p *pool) decrRef() {
	newRef := atomic.AddInt32(&p.ref, -1)
	if newRef < 0 {
		panic(fmt.Sprintf("negative ref: %d", newRef))
	}
	if newRef == 0 && atomic.LoadInt32(&p.current) > int32(p.opt.MaxIdle) {
		p.Lock()
		if atomic.LoadInt32(&p.ref) == 0 {
			atomic.StoreInt32(&p.current, int32(p.opt.MaxIdle))
			p.deleteFrom(p.opt.MaxIdle)
		}
		p.Unlock()
	}
}

func (p *pool) deleteFrom(begin int) {
	for i := begin; i < p.opt.MaxActive; i++ {
		conn := p.conns[i]
		if conn == nil {
			break
		}
		p.conns[i] = nil
		if cc := conn.Value(); cc != nil {
			_ = cc.Close()
		}
	}
}

// Get see Pool interface.
func (p *pool) Get() (Conn, error) {
	// first select from the created connection
	nextRef := p.incrRef()
	p.RLock()
	current := atomic.LoadInt32(&p.current)
	p.RUnlock()
	limit := current * int32(p.opt.MaxConcurrentStreams)
	if nextRef < limit {
		next := atomic.AddUint32(&p.index, 1) % uint32(current)
		return p.conns[next], nil
	}

	// second create new connections
	if current < int32(p.opt.MaxActive) {
		increment := current
		if current+increment > int32(p.opt.MaxActive) {
			increment = int32(p.opt.MaxActive) - current
		}
		for i := current; i < current+increment; i++ {
			c, err := p.opt.Dial(p.address)
			if err != nil {
				return nil, err
			}
			p.conns[i] = p.wrapConn(c)
		}
		atomic.StoreInt32(&p.current, current+increment)
		next := atomic.AddUint32(&p.index, 1) % uint32(current+increment)
		return p.conns[next], nil
	}

	// third check wait
	if p.opt.Wait {
		next := atomic.AddUint32(&p.index, 1) % uint32(atomic.LoadInt32(&p.current))
		return p.conns[next], nil
	}

	// fourth create new connection
	c, err := p.opt.Dial(p.address)
	return p.wrapConn(c), err
}

// Close see Pool interface.
func (p *pool) Close() error {
	atomic.StoreUint32(&p.index, 0)
	atomic.StoreInt32(&p.ref, 0)
	p.deleteFrom(0)
	return nil
}

// String see Pool interface.
func (p *pool) String() string {
	return fmt.Sprintf("address:%s, index:%d, current:%d, ref:%d. option:%v",
		p.address, p.index, p.current, p.ref, p.opt)
}
