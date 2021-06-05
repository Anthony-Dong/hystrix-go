package microservice

import (
	"sync"
	"time"
)

type Limiter interface {
	Acq(req int64)
	TryAcq(req int64) bool
	TryAcqWithTimeOut(req int64, waitTime time.Duration) bool
}

type ticketLimiter struct {
	maxTicket    int64 // 最大存储的ticket
	storedTicket int64 // 当前存储的ticket
	periodTime   int64 // 每个ticket的生产周期

	nextTime int64 // 下一次被批准的时间，也就是在这个时间之前你可以在storedTicket>acq的情况下不等待

	lock sync.Mutex
}

// limiter令牌桶的大小
// limierTime 表示时间
// limierTime=time.Second，limiter=100，表示每s生产100个令牌
// 允许预消费
func NewTicketLimiter(limiter int64, limierTime time.Duration) Limiter {
	return &ticketLimiter{
		periodTime: int64(limierTime) / limiter,
		maxTicket:  limiter,
		nextTime:   time.Now().UnixNano(),
	}
}

func (l *ticketLimiter) resync(now int64) {
	// 如果当前时间大于下一次被批准的时间，其实含义就是你可以制作一些令牌，但是最大数不能超过maxTicket
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
	// 1、storedTicket>req，其实不需要等待，因为我满足你的需求，只需要将storedTicket-req
	// 2、storedTicket<req，由于允许预先消费，其实需要让下次来的人等待到(req-storedTicket)*periodTime+nextTime时间就行了
	storedTicketToSpend := min(req, l.storedTicket)
	l.nextTime = l.nextTime + (req-storedTicketToSpend)*l.periodTime
	l.storedTicket = l.storedTicket - storedTicketToSpend
	return returnTime
}

func (l *ticketLimiter) wait(returnValue int64, now int64) {
	if returnValue > now {
		time.Sleep(time.Duration(returnValue - now))
	}
}

func (l *ticketLimiter) Acq(req int64) {
	l.lock.Lock()
	now := time.Now().UnixNano()
	// 获取被批准放行的时间
	returnValue := l.acq(now, req)
	l.lock.Unlock()
	// 这里只需要判断被批准的时间和当前时间比较就行了，如果被批准放行的时间大于当前时间，说明你需要等待才能通过
	l.wait(returnValue, now)
}

// try其实就是上面判断，被批转的时间和当前时间就行了，如果当前时间+waitTime大于上次被批准放行的时间，其实就可以去获取，否则就不能获取
func (l *ticketLimiter) TryAcqWithTimeOut(req int64, waitTime time.Duration) bool {
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

// try其实就是上面判断，被批转的时间和当前时间就行了，如果当前时间+waitTime大于上次被批准放行的时间，其实就可以去获取，否则就不能获取
func (l *ticketLimiter) TryAcq(req int64) bool {
	return l.TryAcqWithTimeOut(req, 0)
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
