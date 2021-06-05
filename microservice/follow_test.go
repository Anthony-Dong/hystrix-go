package microservice

import (
	"context"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"testing"
	"time"
)

func TestNumberSlice_Increment(t *testing.T) {
}

func TestNumberSlice_bench(t *testing.T) {
	number := NewNumberSlice(10)

	group, _ := errgroup.WithContext(context.Background())
	for x := 0; x < 7000; x++ {
		group.Go(func() error {
			time.Sleep(time.Second * time.Duration(rand.Int31n(5)))
			number.Increment(1)
			number.Increment(1)
			number.Increment(1)
			number.Increment(1)
			number.Increment(1)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		t.Fatal(err)
	}
	t.Log(number.Sum())
}

func BenchmarkNumberSliceIncrement(b *testing.B) {
	n := NewNumberSlice(10)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		n.Increment(1)
		n.Sum()
	}
}
