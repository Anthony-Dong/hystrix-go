package main

import (
	"errors"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"golang.org/x/sync/errgroup"
	"time"
)

func main() {
	// 所以在配置这个参数的时候需要注意，你需要确定一下你们的一个业务量级，你可以配置RequestVolumeThreshold为你们的qps，ErrorPercentThreshold配置为你们认为的一个错误率
	// 比如我的qps是100，我的要求熔断的错误率为10%才开启
	hystrix.ConfigureCommand("my_command", hystrix.CommandConfig{
		RequestVolumeThreshold: 100,
		SleepWindow:            1000,
		ErrorPercentThreshold:  10,
	})
	group := errgroup.Group{}
	count := 0
	ticker := time.NewTicker(time.Millisecond * 10)
	for {
		count++
		if count == 200 {
			break
		}
		<-ticker.C
		count := count
		group.Go(func() error {
			_ = hystrix.Do("my_command", func() error {
				if count%11 == 0 {
					return errors.New("发生业务异常")
				}
				fmt.Println("正常请求:", count)
				return nil
			}, func(e error) error {
				if e == hystrix.ErrCircuitOpen {
					fmt.Println("出现熔断！", count)
					return nil
				}
				fmt.Println("出现业务异常", count)
				return nil
			})
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		panic(err)
	}
}
