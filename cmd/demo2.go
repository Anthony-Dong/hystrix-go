package main

import (
	"errors"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"time"
)

/**
判断是否会触发熔断
其实可以看到结果很明显，当第100次的时候，由于我们此时流量已经大于等于100了，所以已经突破了请求阀值，而且我们的异常量此时是10/100=10，然后可以看到的就是此时已经等于等于阀值了，所以会触发熔断
*/
func main() {
	hystrix.ConfigureCommand("my_command", hystrix.CommandConfig{
		RequestVolumeThreshold: 100,
		SleepWindow:            1000,
		ErrorPercentThreshold:  10, // 如果10s内出现10%的流量异，且10s内流量大于等于100就会触发熔断
	})
	errorCount := 0
	for count := 1; count <= 1000; count++ {
		time.Sleep(time.Millisecond * 10)
		_ = hystrix.Do("my_command", func() error {
			if count%10 == 0 { // 10%
				errorCount++
				return errors.New(fmt.Sprintf("出现业务异常，count:%d, errorCount: %d, errorCount/count: %d, time: %s\n", count, errorCount, int(((float64(errorCount)/float64(count))*100)+0.5), time.Now().Format("2006-01-02 15:04:05.000")))
			}
			fmt.Printf("正常处理, count: %d, time: %s\n", count, time.Now().Format("2006-01-02 15:04:05.000"))
			return nil
		}, func(e error) error {
			if e == hystrix.ErrCircuitOpen {
				fmt.Printf("出现熔断异常, count: %d, time: %s\n", count, time.Now().Format("2006-01-02 15:04:05.000"))
				return nil
			}
			fmt.Printf("%v", e)
			return nil
		})
	}
}
