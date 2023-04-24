package timer

import (
	"container/heap"
	"math/rand"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MinRunTime            = 10 * time.Millisecond
	QueueSize             = 10000 //队列长长度
	Unlimited       int32 = -1
	DefaultPriority       = 1
)

// CallBackFunc 定义回调函数
type CallBackFunc func(interface{})

type HeapTimer struct {
	Locker        *sync.Mutex
	CloseTag      chan bool
	TimerSchedule []*Schedule
}

type Schedule struct {
	Index        int64         `json:"index"`          //元素在堆中的索引
	RunningTime  time.Time     `json:"running_time"`   //执行的时间
	GrowthTime   time.Duration `json:"growth_time"`    //每次增加的时间
	Priority     int           `json:"priority"`       //优先级
	Params       interface{}   `json:"params"`         //回调的参数
	CallBack     CallBackFunc  `json:"call_back"`      //回调函数
	ExecRunTimes int32         `json:"exec_run_times"` //已运行次数
	MaxRunTimes  int32         `json:"max_run_times"`  //最大运行次数
}

var (
	HandleQueue chan *Schedule
	IndexMap    sync.Map
)

type HeapInterface interface {
	WorkChan()
	StarTimer()
	ScheduleLoop()
	Cancel()
	StopIdx(int64)
	AddScheduleFunc(time.Duration, CallBackFunc, interface{}, ...int) int64
	AddCallBack(time.Duration, CallBackFunc, interface{}, ...int) int64
}

func GetInstance(queueSize int) HeapInterface {
	//define queueSize
	if queueSize <= 0 {
		queueSize = QueueSize
	}

	//定义处理队列长度
	HandleQueue = make(chan *Schedule, queueSize)

	//init
	var t HeapTimer
	heap.Init(&t)

	//define Heap Timer
	h := HeapTimer{
		Locker:   new(sync.Mutex),
		CloseTag: make(chan bool),
	}

	//启动goroutine
	for i := 0; i < runtime.NumCPU(); i++ {
		go h.WorkChan()
	}

	//HeapInstance *HeapTimer
	return HeapInterface(&h)
}

// WorkChan 运行任务，遍历queue,函数回调
func (h *HeapTimer) WorkChan() {
	//panic recover
	defer handelPanic()

	//range queue
	for hq := range HandleQueue {
		if hq.Index != 0 {
			//回调函数
			hq.CallBack(hq.Params)
		}
	}
}

// Len heap get TimerSchedule length
func (h *HeapTimer) Len() int {
	return len(h.TimerSchedule)
}

// Less heap judge Priority
func (h *HeapTimer) Less(i, j int) bool {
	p1 := h.TimerSchedule[i].Priority
	p2 := h.TimerSchedule[j].Priority
	return p1 > p2
}

func (h *HeapTimer) Swap(i, j int) {
	h.TimerSchedule[i], h.TimerSchedule[j] = h.TimerSchedule[j], h.TimerSchedule[i]
}

// Push  heap push value
func (h *HeapTimer) Push(v interface{}) {
	h.TimerSchedule = append(h.TimerSchedule, v.(*Schedule))
}

// Pop heap pop data
func (h *HeapTimer) Pop() interface{} {
	old := h.TimerSchedule
	n := len(old)
	x := old[n-1]
	h.TimerSchedule = old[0 : n-1]
	//return x
	return x
}

/*
	   @ AddScheduleFunc 增加任务
		t time.Duration 多长时间运行一次
		callBack 回调函数
		params 回调函数的参数，只支持interface{};
		addParams 可变参数  第一个值作为MaxRunTimes的值，默认为 -1，第二个值作为Priority 优先级的值
*/
//AddScheduleFunc add schedule
func (h *HeapTimer) AddScheduleFunc(t time.Duration, callBack CallBackFunc, params interface{}, addParams ...int) int64 {
	return h.AddCallBack(t, callBack, params, addParams...)
}

// AddCallBack add call back
func (h *HeapTimer) AddCallBack(t time.Duration, callBack CallBackFunc, params interface{}, addParams ...int) int64 {
	//panic recover
	defer handelPanic()

	//judge runningTime
	if t < MinRunTime {
		t = MinRunTime
	}
	//LOCK and UnLock
	h.Locker.Lock()
	defer h.Locker.Unlock()

	//定义MaxRunTimes的值
	runTimes := Unlimited       //代表无限制
	priority := DefaultPriority //优先级

	if len(addParams) > 0 {
		//第一个值为MaxRunTimes
		if addParams[0] > 0 {
			runTimes = int32(addParams[0])
		}
		//第二个值为priority
		if len(addParams) > 1 {
			priority = addParams[1]
		}
	}

	//get index
	index := getIndex()

	//Push
	heap.Push(h, &Schedule{
		Index:        index,
		Params:       params,
		RunningTime:  time.Now().Add(t),
		GrowthTime:   t,
		Priority:     priority,
		CallBack:     callBack,
		MaxRunTimes:  runTimes,
		ExecRunTimes: 0,
	})

	//push index
	IndexMap.Store(index, true)

	//return
	return index
}

// StarTimer 开始timer,调用scheduleLoop 取数据判断
func (h *HeapTimer) StarTimer() {
	go func() {
		for {
			select {
			case t := <-h.CloseTag:
				if t {
					runtime.Goexit()
				}
			default:
				h.ScheduleLoop()
				time.Sleep(MinRunTime)
			}
		}
	}()
}

// Cancel 关闭定时任务
func (h *HeapTimer) Cancel() {
	//设置关闭Tag 通道为true
	h.CloseTag <- true
}

func (h *HeapTimer) StopIdx(index int64) {
	IndexMap.Store(index, false)
}

// ScheduleLoop 循环从堆里面取数据并进行处理
func (h *HeapTimer) ScheduleLoop() {
	//panic recover
	defer handelPanic()

	//Locker
	h.Locker.Lock()
	defer h.Locker.Unlock()
	sTime := time.Now()
	for i := 0; i < h.Len(); i++ {
		if h.Len() <= 0 {
			break
		}

		//从最小堆里面取数据
		t := heap.Pop(h).(*Schedule)

		//判断是否已经被停止
		if idx, ok := IndexMap.Load(t.Index); ok {
			if idx.(bool) == false {
				IndexMap.Delete(t.Index)
				break
			}
		} else {
			//映射表里面没有这个值，存一份
			IndexMap.Store(t.Index, true)
		}

		if t.RunningTime.Before(sTime) {
			//将handle塞动HandleQueue
			HandleQueue <- t

			//如果有配置最大运行次数
			if t.MaxRunTimes > 0 {
				atomic.AddInt32(&t.ExecRunTimes, 1)
				if t.MaxRunTimes <= t.ExecRunTimes {
					//直接退出不再push到堆
					break
				}
			}

			//修改下一次执行时间
			t.RunningTime = sTime.Add(t.GrowthTime)
		}

		//重新push进去
		heap.Push(h, t)
	}
}

// ------------------Local func--------------------------//
func init() {
	//设置随机数种子
	rand.Seed(time.Now().Unix())
}

// handelPanic recover
func handelPanic() {
	err := recover()
	if err != nil {
		debug.PrintStack()
	}
}

// getIndex
func getIndex() (index int64) {
	i := 0
	for {
		index = rand.Int63n(99999999) + 10000000
		_, ok := IndexMap.Load(index)
		if !ok {
			return
		}

		//最大重试100次
		if i >= 100 {
			break
		}
	}
	//返回
	return
}
