// Package timer /*
package timer

/*
* Copyright (c) 2023 JiaZhi.xue <xuejiazhi@gmail.com>
*
* This program is free software: you can use, redistribute, and/or modify
* it under the terms of the MIT license as published by the Free Software
* Foundation.
*
* This program is distributed in the hope that it will be useful, but WITHOUT
* ANY WARRANTY;    without even the implied warranty of MERCHANTABILITY or
* FITNESS FOR A PARTICULAR PURPOSE.
 */
import (
	"container/heap"
	"yftimer/sdk/go-cache-master"
	"math/rand"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MinRunTime              = 10 * time.Millisecond
	QueueSize               = 10000 //队列长长度
	MaxScheduleLength       = 10000 //每个对像最大任务个数
	Unlimited         int32 = -1
	DefaultPriority   int32 = 1
)

// CallBackFunc Define the callback function
type CallBackFunc func(interface{})
type TimerS []*Schedule

// HeapTimer Minimum heap timer structure definition

type HeapTimer struct {
	Locker            *sync.Mutex
	CloseTag          chan bool
	SwapTimerSchedule TimerS
	TimerSchedule     TimerS
}

type Schedule struct {
	Index        int           `json:"index"`          //index
	RunningTime  time.Time     `json:"running_time"`   //Execution time
	GrowthTime   time.Duration `json:"growth_time"`    //Each increment of time
	Priority     int32         `json:"priority"`       //Priority
	Params       interface{}   `json:"params"`         //Parameter of the callback
	CallBack     CallBackFunc  `json:"call_back"`      //Callback function
	ExecRunTimes int32         `json:"exec_run_times"` //Number of runs
	MaxRunTimes  int32         `json:"max_run_times"`  //Maximum run times
}

var (
	HandleQueue chan *Schedule
	//IndexMap    sync.Map
	IndexCache *cache.Cache
)

// Len heap get TimerS length
func (h TimerS) Len() int {
	return len(h)
}

// Less heap judge Priority
func (h TimerS) Less(i, j int) bool {
	p1 := h[i].Priority
	p2 := h[j].Priority
	return p1 < p2
}

func (h TimerS) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push  heap push value
func (h *TimerS) Push(v interface{}) {
	*h = append(*h, v.(*Schedule))
}

// Pop heap pop data
func (h *TimerS) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	//return x
	return x
}

type HeapInterface interface {
	WorkChan()
	StarTimer()
	ScheduleLoop()
	Cancel()
	StopIdx(int)
	ForceStopIdx(int)
	GetRunScLength() int
	GetRunScIndexList() *[]string
	AddScheduleFunc(time.Duration, CallBackFunc, interface{}, ...int) int
	AddCallBack(time.Duration, CallBackFunc, interface{}, ...int) int
}

// GetInstance begin get instance
func GetInstance(queueSize int) HeapInterface {
	//define cache
	IndexCache = cache.New(5*time.Minute, 10*time.Minute)
	//define queueSize
	if queueSize <= 0 {
		queueSize = QueueSize
	}

	//Defines the processing queue length
	HandleQueue = make(chan *Schedule, queueSize)

	//init
	var t TimerS
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

// WorkChan Running tasks, traversing queues, function callbacks
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

// AddScheduleFunc /*
// AddScheduleFunc add schedule
func (h *HeapTimer) AddScheduleFunc(t time.Duration, callBack CallBackFunc, params interface{}, addParams ...int) int {
	/*
	 @ AddScheduleFunc Add a task
	 t : time.Duration How often does it run
	 callBack : Callback function
	 params : Parameter of callback function. Only interface{} is supported;
	 addParams ：The first value of the variable argument is the value of MaxRunTimes,
	             which defaults to -1, and the second value is the value of Priority
	*/
	return h.AddCallBack(t, callBack, params, addParams...)
}

// AddCallBack add call back
func (h *HeapTimer) AddCallBack(t time.Duration, callBack CallBackFunc, params interface{}, addParams ...int) int {
	//panic recover
	defer handelPanic()

	//judge runningTime
	if t < MinRunTime {
		t = MinRunTime
	}

	//If the number of tasks is greater than or
	//equal to the maximum number, the task is discarded
	// return zero
	if h.TimerSchedule.Len() >= MaxScheduleLength {
		return 0
	}

	//LOCK and UnLock
	h.Locker.Lock()
	defer h.Locker.Unlock()

	//Define the value of MaxRunTimes
	runTimes := Unlimited       //Unrestricted representation
	priority := DefaultPriority //priority

	//set runtimes and priority
	if len(addParams) > 0 {
		//The first value is MaxRunTimes
		if addParams[0] > 0 {
			runTimes = int32(addParams[0])
		}
		//The second value is priority
		if len(addParams) > 1 {
			priority = int32(addParams[1])
		}
	}

	//get index
	index := getIndex()

	//Push
	heap.Push(&h.TimerSchedule, &Schedule{
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
	//IndexMap.Store(index, true)
	IndexCache.Set("t_"+strconv.Itoa(index), 1, -1)

	//return
	return index
}

// StarTimer begin timer,call scheduleLoop Fetch data judgment
func (h *HeapTimer) StarTimer() {
	//run schedule
	go func() {
		for {
			select {
			case t, ok := <-h.CloseTag:
				if ok && t {
					runtime.Goexit()
				}
			default:
				time.Sleep(MinRunTime)
				h.ScheduleLoop()
			}
		}
	}()
}

// Cancel
// Disabling a scheduled task
func (h *HeapTimer) Cancel() {
	//Set closing the Tag channel to true
	h.CloseTag <- true
}

// StopIdx Stop the task based on the value of Index
func (h *HeapTimer) StopIdx(index int) {
	//set sync
	//IndexMap.Store(index, false)
	IndexCache.Delete("t_" + strconv.Itoa(index))
}

// ForceStopIdx Force the task to stop based on the value of Index
func (h *HeapTimer) ForceStopIdx(index int) {
	//LOCKER
	h.Locker.Lock()

	defer h.Locker.Unlock()
	//search index remove
	for i, k := range h.TimerSchedule {
		if k.Index == index {
			//remove heap
			heap.Remove(&h.TimerSchedule, i)
			//remove sync map
			//IndexMap.Delete(index)
			IndexCache.Delete("t_" + strconv.Itoa(index))
		}
	}
}

// GetRunScLength  get run schedule length
func (h *HeapTimer) GetRunScLength() int {
	return h.TimerSchedule.Len()
}

func (h *HeapTimer) GetRunScIndexList() *[]string {
	//define value list
	var indexList []string
	//range sync map
	for key, _ := range IndexCache.Items() {
		indexList = append(indexList, key)
	}
	//IndexMap.Range(func(key, value any) bool {
	//	indexList = append(indexList, key.(int))
	//	return true
	//})
	return &indexList
}

// ScheduleLoop
//
//	Loops fetch data from the heap and process it
func (h *HeapTimer) ScheduleLoop() {
	//panic recover
	defer handelPanic()

	//define
	nowTime := time.Now()
	timerLen := h.TimerSchedule.Len()

	//judge len
	if h.TimerSchedule.Len() <= 0 {
		return
	}

	//Locker
	h.Locker.Lock()
	defer h.Locker.Unlock()

	for i := 0; i < timerLen; i++ {
		//Fetch data from the minimum heap
		t := heap.Pop(&h.TimerSchedule).(*Schedule)

		////Determine whether it has been stopped
		//if idx, ok := IndexMap.Load(t.Index); ok {
		//	if !idx.(bool) {
		//		IndexMap.Delete(t.Index)
		//		break
		//	}
		//} else {
		//	//It's not in the map. Save a copy
		//	IndexMap.Store(t.Index, true)
		//}

		if _, ok := IndexCache.Get("t_" + strconv.Itoa(t.Index)); !ok {
			break
		}

		if t.RunningTime.Before(nowTime) {
			//Plug the handle into the HandleQueue
			HandleQueue <- t

			//Maximum number of runs if configured
			if t.MaxRunTimes > 0 {
				atomic.AddInt32(&t.ExecRunTimes, 1)
				if t.MaxRunTimes <= t.ExecRunTimes {
					//Direct exit no longer pushes to the heap
					break
				}
			}

			//Example Change the next execution time
			t.RunningTime = nowTime.Add(t.GrowthTime)
		}

		//judge Priority
		if t.Priority > 100 {
			atomic.StoreInt32(&t.Priority, DefaultPriority)
		} else {
			atomic.AddInt32(&t.Priority, 1)
		}

		//append to swap
		h.SwapTimerSchedule = append(h.SwapTimerSchedule, t)
	}

	//judge and operate
	if h.TimerSchedule.Len() == 0 {
		//copy swap to schedule
		h.TimerSchedule = h.SwapTimerSchedule
		//clean slice
		h.SwapTimerSchedule = nil
	}
}

// ------------------Local func--------------------------//
func init() {
	//Set the random number seed
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
func getIndex() (index int) {
	i := 0
	for {
		index = rand.Intn(9999999) + 1000000
		//_, ok := IndexMap.Load(index)
		_, ok := IndexCache.Get("t_" + strconv.Itoa(index))
		if !ok {
			return
		}

		//The maximum number of retries is 100
		if i >= 100 {
			break
		}
		i++
	}

	//return
	return
}
