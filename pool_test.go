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
	"context"
	"flag"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shimingyah/pool/example/pb"
	"github.com/stretchr/testify/require"
)

var endpoint = flag.String("endpoint", "127.0.0.1:50000", "grpc server endpoint")

func newPool(op *Options) (Pool, *pool, Options, error) {
	opt := DefaultOptions
	opt.Dial = DialTest
	if op != nil {
		opt = *op
	}
	p, err := New(*endpoint, opt)
	return p, p.(*pool), opt, err
}

func TestNew(t *testing.T) {
	p, nativePool, opt, err := newPool(nil)
	require.NoError(t, err)
	defer p.Close()

	require.EqualValues(t, 0, nativePool.index)
	require.EqualValues(t, 0, nativePool.ref)
	require.EqualValues(t, opt.MaxIdle, nativePool.current)
	require.EqualValues(t, opt.MaxActive, len(nativePool.conns))
}

func TestNew2(t *testing.T) {
	opt := DefaultOptions

	_, err := New("", opt)
	require.Error(t, err)

	opt.Dial = nil
	_, err = New("127.0.0.1:8080", opt)
	require.Error(t, err)

	opt = DefaultOptions
	opt.MaxConcurrentStreams = 0
	_, err = New("127.0.0.1:8080", opt)
	require.Error(t, err)

	opt = DefaultOptions
	opt.MaxIdle = 0
	_, err = New("127.0.0.1:8080", opt)
	require.Error(t, err)

	opt = DefaultOptions
	opt.MaxActive = 0
	_, err = New("127.0.0.1:8080", opt)
	require.Error(t, err)

	opt = DefaultOptions
	opt.MaxIdle = 2
	opt.MaxActive = 1
	_, err = New("127.0.0.1:8080", opt)
	require.Error(t, err)
}

func TestClose(t *testing.T) {
	p, nativePool, opt, err := newPool(nil)
	require.NoError(t, err)
	p.Close()

	require.EqualValues(t, 0, nativePool.index)
	require.EqualValues(t, 0, nativePool.ref)
	require.EqualValues(t, 0, nativePool.current)
	require.EqualValues(t, true, nativePool.conns[0] == nil)
	require.EqualValues(t, true, nativePool.conns[opt.MaxIdle-1] == nil)
}

func TestReset(t *testing.T) {
	p, nativePool, opt, err := newPool(nil)
	require.NoError(t, err)
	defer p.Close()

	nativePool.reset(0)
	require.EqualValues(t, true, nativePool.conns[0] == nil)
	nativePool.reset(opt.MaxIdle + 1)
	require.EqualValues(t, true, nativePool.conns[opt.MaxIdle+1] == nil)
}

func TestBasicGet(t *testing.T) {
	p, nativePool, _, err := newPool(nil)
	require.NoError(t, err)
	defer p.Close()

	conn, err := p.Get()
	require.NoError(t, err)
	require.EqualValues(t, true, conn.Value() != nil)

	require.EqualValues(t, 1, nativePool.index)
	require.EqualValues(t, 1, nativePool.ref)

	conn.Close()

	require.EqualValues(t, 1, nativePool.index)
	require.EqualValues(t, 0, nativePool.ref)
}

func TestGetAfterClose(t *testing.T) {
	p, _, _, err := newPool(nil)
	require.NoError(t, err)
	p.Close()

	_, err = p.Get()
	require.EqualError(t, err, "pool is closed")
}

func TestBasicGet2(t *testing.T) {
	opt := DefaultOptions
	opt.Dial = DialTest
	opt.MaxIdle = 1
	opt.MaxActive = 2
	opt.MaxConcurrentStreams = 2
	opt.Reuse = true

	p, nativePool, _, err := newPool(&opt)
	require.NoError(t, err)
	defer p.Close()

	conn1, err := p.Get()
	require.NoError(t, err)
	defer conn1.Close()

	conn2, err := p.Get()
	require.NoError(t, err)
	defer conn2.Close()

	require.EqualValues(t, 2, nativePool.index)
	require.EqualValues(t, 2, nativePool.ref)
	require.EqualValues(t, 1, nativePool.current)

	// create new connections push back to pool
	conn3, err := p.Get()
	require.NoError(t, err)
	defer conn3.Close()

	require.EqualValues(t, 3, nativePool.index)
	require.EqualValues(t, 3, nativePool.ref)
	require.EqualValues(t, 2, nativePool.current)

	conn4, err := p.Get()
	require.NoError(t, err)
	defer conn4.Close()

	// reuse exists connections
	conn5, err := p.Get()
	require.NoError(t, err)
	defer conn5.Close()

	nativeConn := conn5.(*conn)
	require.EqualValues(t, false, nativeConn.once)
}

func TestBasicGet3(t *testing.T) {
	opt := DefaultOptions
	opt.Dial = DialTest
	opt.MaxIdle = 1
	opt.MaxActive = 1
	opt.MaxConcurrentStreams = 1
	opt.Reuse = false

	p, _, _, err := newPool(&opt)
	require.NoError(t, err)
	defer p.Close()

	conn1, err := p.Get()
	require.NoError(t, err)
	defer conn1.Close()

	// create new connections doesn't push back to pool
	conn2, err := p.Get()
	require.NoError(t, err)
	defer conn2.Close()

	nativeConn := conn2.(*conn)
	require.EqualValues(t, true, nativeConn.once)
}

func TestConcurrentGet(t *testing.T) {
	opt := DefaultOptions
	opt.Dial = DialTest
	opt.MaxIdle = 8
	opt.MaxActive = 64
	opt.MaxConcurrentStreams = 2
	opt.Reuse = false

	p, nativePool, _, err := newPool(&opt)
	require.NoError(t, err)
	defer p.Close()

	var wg sync.WaitGroup
	wg.Add(500)

	for i := 0; i < 500; i++ {
		go func(i int) {
			conn, err := p.Get()
			require.NoError(t, err)
			require.EqualValues(t, true, conn != nil)
			conn.Close()
			wg.Done()
			t.Logf("goroutine: %v, index: %v, ref: %v, current: %v", i,
				atomic.LoadUint32(&nativePool.index),
				atomic.LoadInt32(&nativePool.ref),
				atomic.LoadInt32(&nativePool.current))
		}(i)
	}
	wg.Wait()

	require.EqualValues(t, 0, nativePool.ref)
	require.EqualValues(t, opt.MaxIdle, nativePool.current)
	require.EqualValues(t, true, nativePool.conns[0] != nil)
	require.EqualValues(t, true, nativePool.conns[opt.MaxIdle] == nil)
}

var size = 4 * 1024 * 1024

func BenchmarkPoolRPC(b *testing.B) {
	opt := DefaultOptions
	p, err := New(*endpoint, opt)
	if err != nil {
		b.Fatalf("failed to new pool: %v", err)
	}
	defer p.Close()

	testFunc := func() {
		conn, err := p.Get()
		if err != nil {
			b.Fatalf("failed to get conn: %v", err)
		}
		defer conn.Close()

		client := pb.NewEchoClient(conn.Value())
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		data := make([]byte, size)
		_, err = client.Say(ctx, &pb.EchoRequest{Message: data})
		if err != nil {
			b.Fatalf("unexpected error from Say: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(tpb *testing.PB) {
		for tpb.Next() {
			testFunc()
		}
	})
}

func BenchmarkSingleRPC(b *testing.B) {
	testFunc := func() {
		cc, err := Dial(*endpoint)
		if err != nil {
			b.Fatalf("failed to create grpc conn: %v", err)
		}

		client := pb.NewEchoClient(cc)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		data := make([]byte, size)
		_, err = client.Say(ctx, &pb.EchoRequest{Message: data})
		if err != nil {
			b.Fatalf("unexpected error from Say: %v", err)
		}
	}

	b.RunParallel(func(tpb *testing.PB) {
		for tpb.Next() {
			testFunc()
		}
	})
}
