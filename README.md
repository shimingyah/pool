# Pool
this project is forked from  https://github.com/shimingyah/pool

Connection pool for Go's grpc client that supports connection reuse.

Pool provides additional features:

* `Connection reuse` supported by specific MaxConcurrentStreams param.
* `Failure reconnection` supported by grpc's keepalive.

# Getting started

## Install

Import package:

```
import (
    "github.com/dk-laosiji/pool"
)
```

```
go get github.com/dk-laosiji/pool
```

# Usage

```
p, err := pool.New("127.0.0.1:8080", pool.DefaultOptions)
if err != nil {
    log.Fatalf("failed to new pool: %v", err)
}
defer p.Close()

conn, err := p.Get()
if err != nil {
    log.Fatalf("failed to get conn: %v", err)
}
defer conn.Close()

// cc := conn.Value()
// client := pb.NewClient(conn.Value())
```
See the complete example: [https://github.com/dk-laosiji/pool/tree/master/example](https://github.com/dk-laosiji/pool/tree/master/example)

# Reference 
* [https://github.com/fatih/pool](https://github.com/fatih/pool)
* [https://github.com/silenceper/pool](https://github.com/silenceper/pool)

# License

Pool is under the Apache 2.0 license. See the [LICENSE](https://github.com/shimingyah/pool/blob/master/LICENSE) file for details.
