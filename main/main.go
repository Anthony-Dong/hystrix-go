package main

import (
	"errors"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"golang.org/x/sync/errgroup"
	"time"
)

func main() {
	// RequestVolumeThreshold表示从20个请求才开始计算熔断
	// ErrorPercentThreshold表示错误率，比如为50%
	// 比如我现在请求了1s请求了100次，计算值为20，错误率是50%，前15次都失败了，但是熔断器会在20次打开，原因是因为15/20是75%，尽管你的后面全面正常

	// 所以在配置这个参数的时候需要注意，你需要确定一下你们的一个业务量级，你可以配置RequestVolumeThreshold为你们的qps，ErrorPercentThreshold配置为你们认为的一个错误率
	hystrix.ConfigureCommand("my_command", hystrix.CommandConfig{
		RequestVolumeThreshold: 20,
		SleepWindow:            1000,
		ErrorPercentThreshold:  50,
	})
	group := errgroup.Group{}
	count := 0
	ticker := time.NewTicker(time.Millisecond * 10)
	for {
		count++
		if count == 100 {
			break
		}
		<-ticker.C
		count := count
		group.Go(func() error {
			_ = hystrix.Do("my_command", func() error {
				if count < 15 {
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
