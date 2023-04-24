package timer

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

/*
*
Tests add task functions
=== RUN   Test_Func
[2023-04-24 16:40:36] hytimer:hello world
[2023-04-24 16:40:39] hytimer:hello world
[2023-04-24 16:40:42] hytimer:hello world
[2023-04-24 16:40:45] hytimer:hello world
[2023-04-24 16:40:48] hytimer:hello world
[2023-04-24 16:40:51] hytimer:hello world
........
*/
func Test_ScheduleFunc(t *testing.T) {
	//Initializes and sets the queue size
	inst := GetInstance(10000)
	//Adding a scheduled task
	//Once every 3 seconds
	//Callback function:FuncTest1
	//Callback function input param: "hello world"
	inst.AddScheduleFunc(3*time.Second, FuncTest1, "hello world")
	//Scheduled task start
	inst.StarTimer()
	select {}
}

/*
*

	Set the number of runs. After 3 runs, it will stop

=== RUN   Test_RunTimes
[2023-04-24 16:47:28] hytimer:hello world
[2023-04-24 16:47:31] hytimer:hello world
[2023-04-24 16:47:34] hytimer:hello world
*/
func Test_RunTimes(t *testing.T) {
	//Initializes and sets the queue size
	inst := GetInstance(10000)
	//Adding a scheduled task
	//Once every 3 seconds
	//Callback function:FuncTest1
	//Callback function input param: "hello world"
	//MaxRunTimes:3  It will stop after 3 runs
	inst.AddScheduleFunc(3*time.Second, FuncTest1, "hello world", 3)
	//Scheduled task start
	inst.StarTimer()
	select {}
}

/*
*TestHeapTimer_Cancel

	Tests how to use Cancel

=== RUN   Test_Cancel
[2023-04-24 16:55:06] hytimer:hello world
[2023-04-24 16:55:09] hytimer:hello world
[2023-04-24 16:55:12] hytimer:hello world
10 seconds is up,Timer Over
*/
func Test_Cancel(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	//Initializes and sets the queue size
	inst := GetInstance(10000)
	go func() {
		//Adding a scheduled task
		//Once every 3 seconds
		//Callback function:FuncTest1
		//Callback function input param: "hello world"
		//MaxRunTimes:3  It will stop after 3 runs
		inst.AddScheduleFunc(3*time.Second, FuncTest1, "hello world")
		//Scheduled task start
		inst.StarTimer()
	}()

	time.Sleep(10 * time.Second)
	inst.Cancel()
	fmt.Println("10 seconds is up,Timer Over")
	<-sigs
}

/*
*Test_StopByIndex
=== RUN   Test_StopByIndex
[2023-04-24 17:02:35] hytimer:hello world
[2023-04-24 17:02:38] hytimer:hello world
[2023-04-24 17:02:41] hytimer:hello world
My schedule index is:37456253 5 seconds will be close
[2023-04-24 17:02:44] hytimer:hello world
schedule index [37456253] close success
--- PASS: Test_StopByIndex (15.01s)
PASS
ok  	clog/hytimer	15.201s
*/
func Test_StopByIndex(t *testing.T) {
	//Initializes and sets the queue size
	inst := GetInstance(10000)
	//Adding a scheduled task
	//Once every 3 seconds
	//Callback function:FuncTest1
	//Callback function input param: "hello world"
	//MaxRunTimes:3  It will stop after 3 runs
	sid := inst.AddScheduleFunc(3*time.Second, FuncTest1, "hello world")
	//Scheduled task start
	inst.StarTimer()
	//sleep
	time.Sleep(10 * time.Second)
	fmt.Println(fmt.Sprintf("My schedule index is:%d 5 seconds will be close", sid))
	time.Sleep(5 * time.Second)
	inst.StopIdx(sid)
	fmt.Println(fmt.Sprintf("schedule index [%d] close success", sid))
}

// test function
func FuncTest1(v interface{}) {
	fmt.Println(fmt.Sprintf("[%s] hytimer:%s", time.Now().Format("2006-01-02 15:04:05"), v.(string)))
}
