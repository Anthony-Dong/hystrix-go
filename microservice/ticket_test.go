package microservice

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
	"testing"
	"time"
)

func TestNewTicketLimiterAcq(t *testing.T) {
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

func TestNewTicketLimiterTryAcq(t *testing.T) {
	limiter := NewTicketLimiter(100, time.Second)
	fmt.Println("start", time.Now().Format("15:04:05.00"))
	count := 0
	for {
		if limiter.TryAcq(1) {
			count = count + 1
			fmt.Println("count", count, "spend", time.Now().Format("15:04:05.00"))
		}
		if count == 200 {
			return
		}
	}
}

func TestNewLimiterTryAcq(t *testing.T) {
	limiter := rate.NewLimiter(100, 1)
	fmt.Println("start", time.Now().Format("15:04:05.00"))
	count := 0
	for {
		if limiter.Allow() {
			count = count + 1
			fmt.Println("count", count, "spend", time.Now().Format("15:04:05.00"))
		}
		if count == 200 {
			return
		}
	}
}

/**
➜  hystrix-go git:(master) ✗ go test -race -run=none -bench=BenchmarkNewLimiterAcq$ -benchmem -benchtime 5s  ./microservice/...
goos: darwin
goarch: amd64
pkg: github.com/afex/hystrix-go/microservice
BenchmarkNewLimiter-12          13213904               487 ns/op               0 B/op          0 allocs/op
PASS
ok      github.com/afex/hystrix-go/microservice 7.907s

➜  hystrix-go git:(master) ✗ go test -run=none -bench=BenchmarkNewLimiterAcq$ -benchmem -benchtime 5s  ./microservice/...
goos: darwin
goarch: amd64
pkg: github.com/afex/hystrix-go/microservice
BenchmarkNewLimiter-12          60008954               100 ns/op               0 B/op          0 allocs/op
PASS
ok      github.com/afex/hystrix-go/microservice 6.108s
*/
func BenchmarkNewLimiterAcq(b *testing.B) {
	limiter := NewTicketLimiter(10000*1000, time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Acq(1)
	}
}

/**
➜  hystrix-go git:(master) ✗ go test -run=none -bench=BenchmarkNewTicketLimiterTryAcq$ -benchmem -benchtime 5s  ./microservice/...
goos: darwin
goarch: amd64
pkg: github.com/afex/hystrix-go/microservice
BenchmarkNewTicketLimiterTryAcq-12      66087710                82.8 ns/op             0 B/op          0 allocs/op
PASS
ok      github.com/afex/hystrix-go/microservice 5.571s
*/
func BenchmarkNewTicketLimiterTryAcq(b *testing.B) {
	limiter := NewTicketLimiter(10000, time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.TryAcq(1)
	}
}

/**
➜  hystrix-go git:(master) ✗ go test -run=none -bench=BenchmarkNewLimiterTryAcq$ -benchmem -benchtime 5s  ./microservice/...
goos: darwin
goarch: amd64
pkg: github.com/afex/hystrix-go/microservice
BenchmarkNewLimiterTryAcq-12            51607381               119 ns/op               0 B/op          0 allocs/op
PASS
ok      github.com/afex/hystrix-go/microservice 6.269s
*/
func BenchmarkNewLimiterTryAcq(b *testing.B) {
	limiter := rate.NewLimiter(10000, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}
