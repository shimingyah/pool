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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/shimingyah/pool"
	"github.com/shimingyah/pool/example/pb"
)

var addr = flag.String("addr", "127.0.0.1:50000", "the address to connect to")

func main() {
	flag.Parse()

	p, err := pool.New(*addr, pool.DefaultOptions)
	if err != nil {
		log.Fatalf("failed to new pool: %v", err)
	}
	defer p.Close()

	conn, err := p.Get()
	if err != nil {
		log.Fatalf("failed to get conn: %v", err)
	}
	defer conn.Close()

	client := pb.NewEchoClient(conn.Value())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.Say(ctx, &pb.EchoRequest{Message: []byte("hi")})
	if err != nil {
		log.Fatalf("unexpected error from Say: %v", err)
	}
	fmt.Println("rpc response:", res)
}
