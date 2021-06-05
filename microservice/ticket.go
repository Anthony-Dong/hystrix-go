package microservice

import (
	"sync"
	"time"
)

type Limiter interface {
	Acq(req int64)
	TryAcq(req int64, waitTime time.Duration) bool
}

type ticketLimiter struct {
	maxTicket    int64 // 最大存储的ticket
	storedTicket int64 // 当前存储的ticket
	periodTime   int64 // 生产周期

	nextTime int64 // 下一次被批准的时间

	lock sync.Mutex
}

func NewTicketLimiter(limiter int64, limierTime time.Duration) Limiter {
	return &ticketLimiter{
		periodTime: int64(limierTime) / limiter,
		maxTicket:  limiter,
		nextTime:   time.Now().UnixNano(),
	}
}

func (l *ticketLimiter) resync(now int64) {
	// 计算当前时间到上次消费完时间的差值，如果当前时间大于上次的时间，其实就可以认为这个期间生产了多少个令牌，但是最大只能是maxTicket
	// 最大只能是maxTicket是因为令牌桶时间内最大生产量就是maxTicket，假如我0s开始，我2s才开始获取令牌，但是我规定maxTicket=10/s，
	// 所以我不能因为2s的时候因为(2-0)*maxTicket=20 就放行
	// 假如放行会造成这个时间段内虽然执行了20个，但是假如我立马再申请20个呢，我其实会将它阻塞到了4s，虽然可行但是不符合令牌桶的规矩，就是单位时间内的令牌
	// 所以你超过我最大令牌数，你可以放行，但是你需要等待剩余的生产完毕！
	if now > l.nextTime {
		newTicket := (now - l.nextTime) / l.periodTime
		l.storedTicket = min(l.maxTicket, newTicket+l.storedTicket)
		l.nextTime = now
	}
}
func (l *ticketLimiter) acq(now int64, req int64) int64 {
	l.resync(now)
	returnTime := l.nextTime
	// 消费逻辑：
	// 1、现有的如果大于需求的，其实不需要等待，只需要 将现有的减去需要的
	// 2、现有的如果小于需求的，其实需要等待(需要的-现有的)，然后最后将现有的清空
	storedTicketToSpend := min(req, l.storedTicket)
	l.nextTime = l.nextTime + (req-storedTicketToSpend)*l.periodTime
	l.storedTicket = l.storedTicket - storedTicketToSpend
	return returnTime
}

// 什么情况下会阻塞
// 1、你下一次被批准的时间大于你进去的时间，你需要等待到消费完成后才可以继续执行
func (l *ticketLimiter) Acq(req int64) {
	l.lock.Lock()
	now := time.Now().UnixNano()
	returnValue := l.acq(now, req)
	l.lock.Unlock()
	l.wait(returnValue, now)
}

func (l *ticketLimiter) wait(returnValue int64, now int64) {
	if returnValue > now {
		time.Sleep(time.Duration(returnValue - now))
	}
}

// 尝试去获取，就是去看你下次被批准的时间 和 你当前想要消费的时间
// 假如你 你下次被批准的时间 比你当前想要再去时间 大，其实就是你不能去了
// 怎么才能去呢，你想去的时间加上你等待的时间如果大于被批准的时间，说明你等得起，获取机会好，你就可以直接去了
func (l *ticketLimiter) TryAcq(req int64, waitTime time.Duration) bool {
	l.lock.Lock()
	now := time.Now().UnixNano()
	if now+int64(waitTime) > l.nextTime {
		returnValue := l.acq(now, req)
		l.lock.Unlock()
		if returnValue > now {
			time.Sleep(time.Duration(returnValue - now))
		}
		return true
	} else {
		l.lock.Unlock()
		return false
	}
}

func min(num1, num2 int64) int64 {
	if num1 < num2 {
		return num1
	}
	return num2
}

func max(num1, num2 int64) int64 {
	if num1 > num2 {
		return num1
	}
	return num2
}
