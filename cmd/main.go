package main

import (
	"fmt"
	"sync/atomic"
)

func main() {
	var (
		x int32 = 1
	)

	fmt.Println(atomic.CompareAndSwapInt32(&x, 0, 1))
}
