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
	"container/heap"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// Conn single connection inerface
type Conn interface {
	// Value return the actual connection type, can be net.Conn, *grpc.ClientConn...
	Value() interface{}

	// Close put back to pool, instead of close it.
	Close() error

	// MarkUnusable marks the connection not usable any more
	// to let the pool close it instead of returning it to pool.
	MarkUnusable()

	// Less used to impl connection priority.
	Less(other Conn) bool
}

type minHeap []Conn

func (mh minHeap) Len() int            { return len(mh) }
func (mh minHeap) Swap(i, j int)       { mh[i], mh[j] = mh[j], mh[i] }
func (mh minHeap) Less(i, j int) bool  { return mh[i].Less(mh[j]) }
func (mh *minHeap) Push(x interface{}) { *mh = append(*mh, x.(Conn)) }
func (mh *minHeap) Pop() interface{} {
	old := *mh
	n := len(old)
	x := old[n-1]
	*mh = old[0 : n-1]
	return x
}

// PriorityQueue structure
type PriorityQueue struct {
	mh *minHeap
}

// NewPriorityQueue return priority queue
func NewPriorityQueue() *PriorityQueue {
	queue := &PriorityQueue{mh: new(minHeap)}
	heap.Init(queue.mh)
	return queue
}

// Push an item to priority queue
func (queue *PriorityQueue) Push(x Conn) {
	heap.Push(queue.mh, x)
}

// Pop an item from priority queue
func (queue *PriorityQueue) Pop() Conn {
	if x, ok := heap.Pop(queue.mh).(Conn); ok {
		return x
	}
	return nil
}

// Top return an item from priority queue
func (queue *PriorityQueue) Top() Conn {
	if len(*queue.mh) == 0 {
		return nil
	}
	if x, ok := (*queue.mh)[0].(Conn); ok {
		return x
	}
	return nil
}

// Fix re-establishes the heap ordering after the element at index i has changed its value.
// Changing the value of the element at index i and then calling Fix is equivalent to,
// but less expensive than, calling Remove(i) followed by a Push of the new value.
func (queue *PriorityQueue) Fix(i int) {
	heap.Fix(queue.mh, i)
}

// Remove a specified index from priority queue
func (queue *PriorityQueue) Remove(i int) Conn {
	if x, ok := heap.Remove(queue.mh, i).(Conn); ok {
		return x
	}
	return nil
}

// Len return the length of priority queue
func (queue *PriorityQueue) Len() int {
	return queue.mh.Len()
}

// Conn is wrapped grpc.ClientConn.
// add create time to support idle timeout and modify close method.
type conn struct {
	*grpc.ClientConn
	createAt time.Time
	unusable bool
	wait     int
	sync.RWMutex
}

// MarkUnusable marks the connection not usable any more
// to let the pool close it instead of returning it to pool.
func (c *conn) MarkUnusable() {
	c.Lock()
	c.unusable = true
	c.Unlock()
}

func (p *pool) wrapConn(cc *grpc.ClientConn) *conn {
	return &conn{
		ClientConn: cc,
		createAt:   time.Now(),
		unusable:   false,
	}
}
