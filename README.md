# pool

Connection pool for Go's grpc interface.

# Usage

Import package:

```
import (
	"github.com/shimingyah/pool"
)
```

```
go get github.com/shimingyah/pool
```

# example

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