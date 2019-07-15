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
	"sync"
	"time"

	"google.golang.org/grpc"
)

// ErrClosed is the error resulting if the pool is closed via pool.Close().
var ErrClosed = errors.New("pool is closed")

// Pool interface describes a pool implementation.
// An ideal pool is threadsafe and easy to use.
type Pool interface {
	// Get returns a new connection from the pool. Closing the connections puts
	// it back to the Pool. Closing it when the pool is destroyed or full will
	// be counted as an error.
	Get() (*grpc.ClientConn, error)

	// Close closes the pool and all its connections. After Close() the pool is
	// no longer usable.
	Close() error

	// Len returns the current number of connections of the pool.
	Len() int
}

// Options are params for creating connect pool.
type Options struct {
	// Dial is an application supplied function for creating and configuring a connection.
	Dial func() (*grpc.ClientConn, error)

	// Close the connect instead of put it back to pool.
	Close func(*grpc.ClientConn) error

	// Ping is an optional param for checking the health of idle connection
	Ping func(*grpc.ClientConn) error

	// Maximum number of idle connections in the pool.
	MaxIdle int

	// Maximum number of connections allocated by the pool at a given time.
	// When zero, there is no limit on the number of connections in the pool.
	MaxActive int

	// Close connections after remaining idle for this duration. If the value
	// is zero, then idle connections are not closed. Applications should set
	// the timeout to a value less than the server's timeout.
	IdleTimeout time.Duration

	// If Wait is true and the pool is at the MaxActive limit, then Get() waits
	// for a connection to be returned to the pool before returning.
	Wait bool
}

type pool struct {
	maxIdle     int
	maxActive   int
	wait        bool
	idleTimeout time.Duration
	conns       chan Conn
	dial        func() (*grpc.ClientConn, error)
	close       func(*grpc.ClientConn) error
	ping        func(*grpc.ClientConn) error
	sync.RWMutex
}

// New return a connection pool.
func New(option Options) (Pool, error) {
	if option.MaxIdle < 0 || option.MaxActive <= 0 || option.MaxIdle > option.MaxActive {
		return nil, errors.New("invalid maximum settings")
	}
	if option.Dial == nil {
		return nil, errors.New("invalid dial settings")
	}
	if option.Close == nil {
		return nil, errors.New("invalid close settings")
	}

	p := &pool{
		maxIdle:     option.MaxIdle,
		maxActive:   option.MaxActive,
		wait:        option.Wait,
		idleTimeout: option.IdleTimeout,
		dial:        option.Dial,
		close:       option.Close,
		ping:        option.Ping,
		conns:       make(chan Conn, option.MaxActive),
	}

	for i := 0; i < p.maxIdle; i++ {
		c, err := p.dial()
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("dial is not able to fill the pool: %s", err)
		}
		p.conns <- p.wrapConn(c)
	}

	return p, nil
}

// put puts the connection back to the pool. If the pool is full or closed,
// conn is simply closed. A nil conn will be rejected.
func (p *pool) put(c *Conn) error {
	if c == nil {
		return errors.New("connection is nil. rejecting")
	}
	if p.conns == nil {
		return p.close(c.ClientConn)
	}

	select {
	case p.conns <- c:
		return nil
	default:
		return p.close(c.ClientConn)
	}
}

// Get see Pool interface.
func (p *pool) Get() (*grpc.ClientConn, error) {
	for {
		select {
		case c := <-p.conns:
			if c == nil {
				return nil, ErrClosed
			}
			if timeout := p.idleTimeout; timeout > 0 {
				if c.createAt.Add(timeout).Before(time.Now()) {
				}
			}
		}
	}
	return nil, nil
}

// Close see Pool interface.
func (p *pool) Close() error {
	return nil
}

// Len see Pool interface.
func (p *pool) Len() int {
	return 0
}
