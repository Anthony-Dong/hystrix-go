> ​	这个为我个人学习`hystrix`熔断器的一个项目，他原来项目不是go mod 项目，所以我把它改造成了go mod项目，原项目地址：[https://github.com/afex/hystrix-go](https://github.com/afex/hystrix-go)

### 1、简单使用

1、参数

- `RequestVolumeThreshold` 表示多少个请求才开始计算错误率，比如为100，那么100次以后才开始熔断策略，不管你前面是不是圈失败了
- `ErrorPercentThreshold` 表示多个请求经过的错误率，比如你20个请求开始计算，但是你剩余的流量假如全OK，但是你前20次全部错误，这也会使你剩余的流量出现熔断！ 剩余的部分需要半开状态慢慢开启恢复！

2、注意点

我们需要学会配置这些参数！

3、代码demo

```go
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
```

### 2、如何配置参数

其实你完全可以理解`RequestVolumeThreshold`表示你么业务的一个qps，你的`ErrorPercentThreshold` 表示你想要的一个错误率

比如下面这个例子，我的qps是100，所以我配置的`RequestVolumeThreshold`为100，我要求如果10%的流量出现异常我就立马熔断，所以我配置的是`ErrorPercentThreshold`为10；

所以我当`count%10 == 0` 时会发现后面100个流量全部熔断，但是如果为 `count%11 == 0`其实并不会熔断！所以也完全符合我们的配置参数！

```go
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
				if count%10 == 0 {
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
```


