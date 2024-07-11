package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan int, 1)
	go getfromchan(ch, 1)
	go getfromchan(ch, 2)
	go getfromchan(ch, 3)
	ch <- 1
	time.Sleep(time.Second * 2)
	fmt.Println("close channel")
	close(ch)
	time.Sleep(time.Hour)
	for {
		select {}
	}
}

func getfromchan(ch chan int, id int) {
	for d := range ch {
		fmt.Printf("get data:%d from id:%d", d, id)
	}
	fmt.Printf("end,id:%d\n", id)
}

//只有 close 才能唤醒所有等待者，
//cond 条件变量，可以同步，唤醒其他线程的效果，可以唤醒一个或者所有等待者。
//channel 只有close 才能唤醒所有等待者， 往channe 写一个数据，只能唤醒一个等待者。
