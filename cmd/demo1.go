package main

import (
	"errors"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"time"
)

/**
1、测试 RequestVolumeThreshold 值，如果10s内流量一直小于100那是不会触发熔断策略的
2、配置 arg为 90 或者 110查看效果
*/
func main() {
	arg := 90
	hystrix.ConfigureCommand("my_command", hystrix.CommandConfig{
		RequestVolumeThreshold: 100, // 10s 如果10s内流量一直小于100，那么就不会触发熔断，不管你异常阀值是多少
		SleepWindow:            1000,
		ErrorPercentThreshold:  10,
	})
	for count := 0; count < 300; count++ {
		time.Sleep(time.Millisecond * time.Duration(arg))
		_ = hystrix.Do("my_command", func() error {
			return errors.New(fmt.Sprintf("异常，time: %s", time.Now().Format("2006-01-02 15:04:05.000")))
		}, func(e error) error {
			if e == hystrix.ErrCircuitOpen {
				fmt.Println("出现熔断异常", count)
				return nil
			}
			fmt.Printf("出现业务异常,count: %d, err:%v\n", count, e)
			return nil
		})
	}
}
