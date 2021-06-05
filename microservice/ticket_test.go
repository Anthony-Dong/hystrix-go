package microservice

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"testing"
	"time"
)

func TestNewTicketLimiter(t *testing.T) {
	limiter := NewTicketLimiter(50, time.Second)
	group, _ := errgroup.WithContext(context.Background())
	fmt.Println("start", time.Now().Format("15:04:05.00"))
	for x := 0; x < 500; x++ {
		group.Go(func() error {
			limiter.Acq(1)
			fmt.Println("spend", time.Now().Format("15:04:05.00"))
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		t.Fatal(err)
	}
}

/**
➜  hystrix-go git:(master) ✗ go test -race -run=none -bench=BenchmarkNewLimiter -benchmem -benchtime 5s  ./microservice/...
goos: darwin
goarch: amd64
pkg: github.com/afex/hystrix-go/microservice
BenchmarkNewLimiter-12          13213904               487 ns/op               0 B/op          0 allocs/op
PASS
ok      github.com/afex/hystrix-go/microservice 7.907s

➜  hystrix-go git:(master) ✗ go test -run=none -bench=BenchmarkNewLimiter -benchmem -benchtime 5s  ./microservice/...
goos: darwin
goarch: amd64
pkg: github.com/afex/hystrix-go/microservice
BenchmarkNewLimiter-12          60008954               100 ns/op               0 B/op          0 allocs/op
PASS
ok      github.com/afex/hystrix-go/microservice 6.108s
*/
func BenchmarkNewLimiter(b *testing.B) {
	limiter := NewTicketLimiter(10000*1000, time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Acq(1)
	}
}
