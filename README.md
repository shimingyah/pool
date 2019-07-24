# Pool

Connection pool for Go's grpc client that supports connection reuse.

# Getting started

## Install

Import package:

```
import (
    "github.com/shimingyah/pool"
)
```

```
go get github.com/shimingyah/pool
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

# Reference
* [https://github.com/fatih/pool](https://github.com/fatih/pool)
* [https://github.com/silenceper/pool](https://github.com/silenceper/pool)

# License

Pool is under the Apache 2.0 license. See the [LICENSE](https://github.com/shimingyah/pool/blob/master/LICENSE) file for details.