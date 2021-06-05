package microservice

import (
	"sync"
	"time"
)

type NumberSlice struct {
	Buckets []*numberSliceBucket
	lock    *sync.RWMutex

	windowsSize int64
}

func NewNumberSlice(windowsSize int) *NumberSlice {
	slice := &NumberSlice{
		Buckets:     make([]*numberSliceBucket, windowsSize),
		windowsSize: int64(windowsSize),
		lock:        &sync.RWMutex{},
	}
	for index := range slice.Buckets {
		slice.Buckets[index] = &numberSliceBucket{}
	}
	return slice
}

type numberSliceBucket struct {
	start int64
	value uint64
}

func (r *NumberSlice) Increment(i uint64) {
	now := time.Now().Unix()
	index := now % r.windowsSize
	bucket := r.Buckets[index]
	r.lock.Lock()
	if bucket.start != now { // 如果不加锁会存在并发安全的问题
		bucket.start = now
		bucket.value = 0
	}
	bucket.value = bucket.value + i
	r.lock.Unlock()
}

func (r *NumberSlice) Sum() uint64 {
	begin := time.Now().Unix() - r.windowsSize // 比如当前时间是12s，窗口是10s，那么计算>2s开始的时候的全部
	r.lock.RLock()
	defer r.lock.RUnlock()
	var sum uint64 = 0
	for _, elem := range r.Buckets {
		if elem.start > begin {
			sum = sum + elem.value
		}
	}
	return sum
}
