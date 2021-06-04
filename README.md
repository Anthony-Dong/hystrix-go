> ​	这个为我个人学习`hystrix`熔断器的一个项目，他原来项目不是go mod 项目，所以我把它改造成了go mod项目，原项目地址：[https://github.com/afex/hystrix-go](https://github.com/afex/hystrix-go) 

## 1、快速开始

1、参数

- `RequestVolumeThreshold` 表示过去10s内多少个请求通过才开始触发熔断机制，比如设置为100，那么只有当过去10s内流量大于100次，以后才开始触发熔断逻辑判断策略，不管你前面是不是全部失败了！

  > ​	 这个比较适合流量比较小的，可以完全忽略熔断它！其次就是基数越小，随机性越大，假如我10s某个接口流量就是10个，那么10个出错的概率也很难去量化，因为它只有1/10 ->1 ，10个，但是如果流量很大那种概率其实就不一样了！
  >
  > ​	而且熔断主要是应对毛刺，也就是突发的情况，所以熔断的值一般会设置的比较低一些，那么其实假如这台服务器有10%的流量现在有问题了，那么是不是代表它很可能已经有问题了！正常是多少呢？

- `ErrorPercentThreshold` 表示错误率大于这个值就会触发熔断；比如我过去10s一共1000流量，假如出现了前9s出现100次异常，配置是10%，那么第10s假如出现异常，就会触发熔断机制！

  > ​	但是其实你会发现了冲突，是因为不好控制`RequestVolumeThreshold`它的大小具体多少呢？通用配置么

2、注意点

我们需要学会配置这些参数！

3、代码demo

- 测试`RequestVolumeThreshold` 值 [cmd/demo1.go](cmd/demo1.go)
- 测试`ErrorPercentThreshold`值 [cmd/demo2.go](cmd/demo2.go)

## 3、源码分析

代码位置在：[hystrix/hystrix.go](hystrix/hystrix.go)

```go
func GoC(ctx context.Context, name string, run runFuncC, fallback fallbackFuncC) chan error {
	cmd := &command{ // 组装对象，这里其实完全可以使用sync.Pool去减少对象的内存
		run:      run,      // 	运行函数
		fallback: fallback, //  降级接口
		start:    time.Now(),
		errChan:  make(chan error, 1),
		finished: make(chan bool, 1),
	}
	circuit, _, err := GetCircuit(name) // 核心的熔断器，其实就是根据name获取配置
	if err != nil {
		cmd.errChan <- err
		return cmd.errChan
	}
	cmd.circuit = circuit
	ticketCond := sync.NewCond(cmd)
	ticketChecked := false
  
	returnTicket := func() { // 返回令牌（依靠buffer channel 来保证最大并发的流量）
		cmd.Lock()
		// Avoid releasing before a ticket is acquired.
		for !ticketChecked {
			ticketCond.Wait()
		}
		cmd.circuit.executorPool.Return(cmd.ticket)
		cmd.Unlock()
	}
  
	returnOnce := &sync.Once{}
	reportAllEvent := func() {
		err := cmd.circuit.ReportEvent(cmd.events, cmd.start, cmd.runDuration)
		if err != nil {
			log.Printf(err.Error())
		}
	}

  // 这里是核心逻辑！
	go func() {
		defer func() { cmd.finished <- true }()

    if !cmd.circuit.AllowRequest() { // 是否可以执行，这是如果是否走下面(这里包含对于熔断器的一个判断)
			cmd.Lock() // 否则就是放回returnTicket，降级接口，报告异常
			ticketChecked = true
			ticketCond.Signal()
			cmd.Unlock()
			returnOnce.Do(func() {
				returnTicket()
				cmd.errorWithFallback(ctx, ErrCircuitOpen) // 抛出熔断器已经打开了
				reportAllEvent()
			})
			return
		}

		cmd.Lock()
		select {
		case cmd.ticket = <-circuit.executorPool.Tickets: // 限流器，如果拿到令牌就继续执行了
			ticketChecked = true
			ticketCond.Signal()
			cmd.Unlock()
		default:
			ticketChecked = true
			ticketCond.Signal()
			cmd.Unlock()
			returnOnce.Do(func() {
				returnTicket()
				cmd.errorWithFallback(ctx, ErrMaxConcurrency) // 否则就抛出了一个最大并发异常
				reportAllEvent()
			})
			return
		}

		runStart := time.Now()
		runErr := run(ctx) // 执行代码
		returnOnce.Do(func() {
			defer reportAllEvent()
			cmd.runDuration = time.Since(runStart)
			returnTicket()
			if runErr != nil {
				cmd.errorWithFallback(ctx, runErr) // 执行失败
				return
			}
			cmd.reportEvent("success") // 报告执行成功
		})
	}()

  // 这里其实就是一个超时控制器，实现很简单！
	go func() {
		timer := time.NewTimer(getSettings(name).Timeout)
		defer timer.Stop()

		select {
		case <-cmd.finished:
		case <-ctx.Done():
			returnOnce.Do(func() {
				returnTicket()
				cmd.errorWithFallback(ctx, ctx.Err())
				reportAllEvent()
			})
			return
		case <-timer.C:
			returnOnce.Do(func() {
				returnTicket()
				cmd.errorWithFallback(ctx, ErrTimeout)
				reportAllEvent()
			})
			return
		}
	}()

	return cmd.errChan
}
```

主要是还是关于核心的逻辑 `cmd.circuit.AllowRequest`

```go
func (circuit *CircuitBreaker) AllowRequest() bool {
	return !circuit.IsOpen() || circuit.allowSingleTest()
}
```

继续看第一个, `IsOpen` 返回true表示开启了熔断器

- 半开或者开都返回 true
- 过去10s流量小于`RequestVolumeThreshold` 返回false
- 过去10s异常率大于等于ErrorPercentThreshold` 返回true，并且设置熔断器为开的状态
- 否则就是返回false

```go
func (circuit *CircuitBreaker) IsOpen() bool {
	circuit.mutex.RLock()
	o := circuit.forceOpen || circuit.open
	circuit.mutex.RUnlock()

	if o { // 半开或者开都是返回开
		return true
	}

	// 假如过去10s的数据超过了请求 小于 请求的阀值，所以就是不care
	// 假如过去请求大于等于这个值，就继续走下面逻辑了
	if uint64(circuit.metrics.Requests().Sum(time.Now())) < getSettings(circuit.Name).RequestVolumeThreshold {
		return false
	}

	// 判断是健康，其实就是判断error率
	if !circuit.metrics.IsHealthy(time.Now()) {
		// too many failures, open the circuit
		circuit.setOpen()
		return true
	}
	return false
}
```

继续看，如果我们`IsOpen` 返回false其实就代表了是可以继续执行，但是假如返回true就需要下一步了  `allowSingleTest`方法去判断是否可以尝试了！

```go
func (circuit *CircuitBreaker) allowSingleTest() bool {
	circuit.mutex.RLock()
	defer circuit.mutex.RUnlock()

	now := time.Now().UnixNano()
	openedOrLastTestedTime := atomic.LoadInt64(&circuit.openedOrLastTestedTime)
	// 如果当前时间>打开熔断器的时间+SleepWindow时间，表示我已经过去了那个窗口期，就可以尝试去试流量了
  //
	if circuit.open && now > openedOrLastTestedTime+getSettings(circuit.Name).SleepWindow.Nanoseconds() {
		// 如果交换成功，其实就是解决并发问题
		swapped := atomic.CompareAndSwapInt64(&circuit.openedOrLastTestedTime, openedOrLastTestedTime, now)
		if swapped {
			log.Printf("hystrix-go: allowing single test to possibly close circuit %v", circuit.Name)
		}
		return swapped
	}

	return false
}
```

到这里，其实就能看到如果大于窗口期，就会释放当前请求，不过如果并发比较严重的话，其实就只有少数能过去！

假如这个流量通过，就会返回true！

- 假如熔断器关闭，直接返回true
- 假如熔断器开启，那么去看看允不允许尝试放部分流量，如果运行就返回true

我们以第一二种情况的话，那么会直接成功，所以会执行函数，假如函数执行成功，可以看到他会执行到`ReportEvent`函数，因此继续执行就到了关闭熔断器这(前提是熔断器打开)

```go
reportAllEvent := func() {
  err := cmd.circuit.ReportEvent(cmd.events, cmd.start, cmd.runDuration)
  if err != nil {
    log.Printf(err.Error())
  }
}
```

继续其实如果你的函数

```go
// ReportEvent records command metrics for tracking recent error rates and exposing data to the dashboard.
func (circuit *CircuitBreaker) ReportEvent(eventTypes []string, start time.Time, runDuration time.Duration) error {
	if len(eventTypes) == 0 {
		return fmt.Errorf("no event types sent for metrics")
	}

	circuit.mutex.RLock()
	o := circuit.open // 你可以看这里，这里去拿到open，然后发现成功后，就关闭了熔断器
	circuit.mutex.RUnlock()
	if eventTypes[0] == "success" && o {
		circuit.setClose()
	}

///。。。。。。。。。。收集信息

	return nil
}
```

#### 问题

```go
type CircuitBreaker struct {
	Name                   string
	open                   bool // 开启
  forceOpen              bool // 强制开启(业务逻辑中其实这个一直是false，因为没有地方set)
	mutex                  *sync.RWMutex
	openedOrLastTestedTime int64 // 打开熔断器的时间

	executorPool *executorPool
	metrics      *metricExchange
}

```

所以go版本的熔断器，只有一个过程那就是开和关

> ​	开启后会过了一个时间段后，尝试释放一部分流量过去如何成功就立马关闭熔断器，如果还有异常继续判断异常率等！
>
> 所以半开状态并不是一个很好探测流量的方式，个性化配置这块比较小！

#### 总结

其实这个代码很多地方值得优化，可以看到大量的使用channle和g，使得go的cpu还是内存都会提高一个level！

其次就是限流器做的很简单，没有做很细致的优化！

## 3、值得学习的地方

1、流量统计

代码位于：[hystrix/rolling/rolling.go](hystrix/rolling/rolling.go)

假如自己设计一个流量统计模块，统计10s内全部的流量，咋设计呢？带着疑问自己思考，可以直接看hystrix

你可以看一下他是咋设计的，其实就是`Buckets` 存放的统计数据，我需要统计的数据只有距离当前10s内的全部key，所以很简单！而我插入数据的时候，只需要最后删除那些大于距离当前10s的key！

```go
type Number struct {
	Buckets map[int64]*numberBucket // key是最近10s，value是数量
	Mutex   *sync.RWMutex
}
```

所以对于写多读多的场景下，都符合我们的要求！