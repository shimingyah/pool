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
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
)

var endpoint = flag.String("endpoint", "127.0.0.1:8080", "grpc server endpoint")

func TestNew(t *testing.T) {
	p, err := New(*endpoint, DefaultOptions)
	require.NoError(t, err)
	defer p.Close()

	nativePool := p.(*pool)
	require.EqualValues(t, 0, nativePool.index)
	require.EqualValues(t, 0, nativePool.ref)
	require.EqualValues(t, DefaultOptions.MaxIdle, nativePool.current)
	require.EqualValues(t, DefaultOptions.MaxActive, len(nativePool.conns))
}

func TestReset(t *testing.T) {
	p, err := New(*endpoint, DefaultOptions)
	require.NoError(t, err)
	defer p.Close()

	nativePool := p.(*pool)

	nativePool.reset(0)
	require.EqualValues(t, true, nativePool.conns[0] == nil)
	nativePool.reset(DefaultOptions.MaxIdle + 1)
	require.EqualValues(t, true, nativePool.conns[DefaultOptions.MaxIdle+1] == nil)
}

func TestBasicGet(t *testing.T) {
	p, err := New(*endpoint, DefaultOptions)
	require.NoError(t, err)
	defer p.Close()

	nativePool := p.(*pool)
	conn, err := p.Get()
	require.NoError(t, err)

	require.EqualValues(t, 1, nativePool.index)
	require.EqualValues(t, 1, nativePool.ref)

	conn.Close()

	require.EqualValues(t, 1, nativePool.index)
	require.EqualValues(t, 0, nativePool.ref)
}

func TestConcurrentGet(t *testing.T) {
	p, err := New(*endpoint, DefaultOptions)
	require.NoError(t, err)
	defer p.Close()

	nativePool := p.(*pool)
	t.Log(nativePool)
}
