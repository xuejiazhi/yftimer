// Package timer /*
package timer

/*
* Copyright (c) 2023 jiazhi.xue <xuejiazhi@gmail.com>
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
	DefaultPriority int32 = 1
)

// CallBackFunc Define the callback function
type CallBackFunc func(interface{})

// HeapTimer Minimum heap timer structure definition

type HeapTimer struct {
	Locker            *sync.Mutex
	CloseTag          chan bool
	SwapTimerSchedule []*Schedule
	TimerSchedule     []*Schedule
}

type Schedule struct {
	Index        int64         `json:"index"`          //index
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

// GetInstance begin get instance
func GetInstance(queueSize int) HeapInterface {
	//define queueSize
	if queueSize <= 0 {
		queueSize = QueueSize
	}

	//Defines the processing queue length
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

// Len heap get TimerSchedule length
func (h *HeapTimer) Len() int {
	return len(h.TimerSchedule)
}

// Less heap judge Priority
func (h *HeapTimer) Less(i, j int) bool {
	p1 := h.TimerSchedule[i].Priority
	p2 := h.TimerSchedule[j].Priority
	return p1 < p2
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
 @ AddScheduleFunc Add a task
 t : time.Duration How often does it run
 callBack : Callback function
 params : Parameter of callback function. Only interface{} is supported;
 addParams ：The first value of the variable argument is the value of MaxRunTimes,
             which defaults to -1, and the second value is the value of Priority
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

	//Define the value of MaxRunTimes
	runTimes := Unlimited       //Unrestricted representation
	priority := DefaultPriority //priority

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

// StarTimer begin timer,call scheduleLoop Fetch data judgment
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

// Cancel
// Disabling a scheduled task
func (h *HeapTimer) Cancel() {
	//Set closing the Tag channel to true
	h.CloseTag <- true
}

func (h *HeapTimer) StopIdx(index int64) {
	IndexMap.Store(index, false)
}

// ScheduleLoop
//
//	Loops fetch data from the heap and process it
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

		//Fetch data from the minimum heap
		t := heap.Pop(h).(*Schedule)

		//Determine whether it has been stopped
		if idx, ok := IndexMap.Load(t.Index); ok {
			if idx.(bool) == false {
				IndexMap.Delete(t.Index)
				break
			}
		} else {
			//It's not in the map. Save a copy
			IndexMap.Store(t.Index, true)
		}

		if t.RunningTime.Before(sTime) {
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
			t.RunningTime = sTime.Add(t.GrowthTime)
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
	if h.Len() == 0 {
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
func getIndex() (index int64) {
	i := 0
	for {
		index = rand.Int63n(99999999) + 10000000
		_, ok := IndexMap.Load(index)
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
