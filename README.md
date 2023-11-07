# yftimer
**yftimer**  is a timer implemented using the minimum heap "container/heap". It is very lightweight and uses very few resources. It is very efficient..
yftimer是一个使用最小堆“container/heap”实现的定时器，它非常轻量级并且使用很少的资源，非常的高效。
## Installation
> $ go get -u github.com/xuejiazhi/yftimer

## Usage
### 1、init
```
===Coding
//Initializes and sets the queue size
inst := GetInstance(10000) 
```
### 2、Add Schedule 
#### Func
```
/**
 Adding a scheduled task
 Once every 3 seconds
 Callback function:FuncTest1
 Callback function input param: "hello world" 
**/
inst.AddScheduleFunc(3*time.Second, FuncTest1, "hello world")

//Scheduled task start
inst.StarTimer()
select {}
```

#### OutPut
```
=== RUN   Test_Func
[2023-04-24 16:40:36] yftimer:hello world
[2023-04-24 16:40:39] yftimer:hello world
[2023-04-24 16:40:42] yftimer:hello world
[2023-04-24 16:40:45] yftimer:hello world
[2023-04-24 16:40:48] yftimer:hello world
[2023-04-24 16:40:51] yftimer:hello world
........
```
### 3、Add Schedule Runtimes
#### Func
```
/**
 Adding a scheduled task
 Once every 3 seconds
 Callback function:FuncTest1
 Callback function input param: "hello world" 
 MaxRunTimes:3  It will stop after 3 runs 
**/
inst.AddScheduleFunc(3*time.Second, FuncTest1, "hello world",3)

//Scheduled task start
inst.StarTimer()
select {}
```

#### OutPut
```
=== RUN   Test_Func
[2023-04-24 16:47:28] yftimer:hello world
[2023-04-24 16:47:31] yftimer:hello world
[2023-04-24 16:47:34] yftimer:hello world
Run Over!
```
### 4、cancel Schedule
#### Func
```
//Initializes and sets the queue size
inst := GetInstance(10000)

go func() {
  /**
    Adding a scheduled task
    Once every 3 seconds
    Callback function:FuncTest1
    Callback function input param: "hello world"
  **/
  inst.AddScheduleFunc(3*time.Second, FuncTest1, "hello world")
  
  //Scheduled task start
  inst.StarTimer()
}()

//sleep 10 seconds
time.Sleep(10 * time.Second)
//cancel schedule
inst.Cancel()

fmt.Println("10 seconds is up,Timer Over")
select {}
```

#### OutPut
```
=== RUN   Test_Cancel
[2023-04-24 16:55:06] yftimer:hello world
[2023-04-24 16:55:09] yftimer:hello world
[2023-04-24 16:55:12] yftimer:hello world
10 seconds is up,Timer Over
```
### 5、Stop By Index
#### Func
```
//Initializes and sets the queue size
inst := GetInstance(10000)

/*
  Adding a scheduled task
  Once every 3 seconds
  Callback function:FuncTest1
  Callback function input param: "hello world"
*/
sid := inst.AddScheduleFunc(3*time.Second, FuncTest1, "hello world")

//Scheduled task start
inst.StarTimer()
	
//sleep
time.Sleep(10 * time.Second)
fmt.Println(fmt.Sprintf("My schedule index is:%d 5 seconds will be close", sid))

//stop by index
time.Sleep(5 * time.Second)
inst.StopIdx(sid)

//print
fmt.Println(fmt.Sprintf("schedule index [%d] close success", sid))
```
#### OutPut
```
=== RUN   Test_StopByIndex
[2023-04-24 17:02:35] yftimer:hello world
[2023-04-24 17:02:38] yftimer:hello world
[2023-04-24 17:02:41] yftimer:hello world
My schedule index is:37456253 5 seconds will be close
[2023-04-24 17:02:44] yftimer:hello world
schedule index [37456253] close success
--- PASS: Test_StopByIndex (15.01s)
PASS
ok  	clog/yftimer	15.201s
```
