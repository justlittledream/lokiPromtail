/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// type label struct{
// 	key string
// 	value string
// }

// type promtailStream struct {
// 	Labels string       `json:"stream"`
// 	Values [1][2]string `json:"values"`
// }

// type promtailMsg struct {
// 	Streams []promtailStream `json:"streams"`
// }

//var finishedRequest = make(chan time.Duration, 100000)
//var badRequest = make(chan time.Duration, 100000)

var finishedRequest = make(map[int]time.Duration)
var badRequest = make(map[int]time.Duration)
var lock sync.Mutex

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push logs to loki",
	Long:  `----------help----------------`,
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		key, _ := cmd.Flags().GetString("key")
		value, _ := cmd.Flags().GetString("value")
		task, _ := cmd.Flags().GetInt("task")
		//fre, _ := cmd.Flags().GetInt("frequency")
		max, _ := cmd.Flags().GetInt("max")
		fmt.Printf("\nPush %v logs to loki with max concurrent tasks = %v\n", task, max)
		//fmt.Printf("\nbegin to send logs\n")
		costTime := sendLogs(key, value, task, max)
		//fmt.Printf("\nresults is as followed\n")
		calculate(startTime, costTime)
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.PersistentFlags().StringP("key", "k", "app", "key of label")
	pushCmd.PersistentFlags().StringP("value", "v", "server", "value of label")
	pushCmd.PersistentFlags().IntP("task", "t", 1000, "number of logs")
	//pushCmd.PersistentFlags().IntP("frequency", "p", 1, "send a log every p seconds")
	pushCmd.PersistentFlags().IntP("max", "m", 10, "Maximum of concurrent task")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pushCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// pushCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

/*
input：
	key: key of label
	value: value of label
	task：number of logs;
	max: Maximum of concurrent task
output：
	endtime
*/
func sendLogs(key, value string, task, max int) time.Duration {
	//统一了推送日志流的时间
	postTime := time.Now().UnixNano() // 纳秒
	logTime := strconv.FormatInt(int64(postTime), 10)
	//记录执行发送第一条请求前的时间
	startTime := time.Now()

	channel := make(chan int, max) //控制协程最大数
	for i := 0; i < task; i++ {
		channel <- 1
		go send(key, value, i, channel, logTime)
	}
	for {
		if len(channel) == 0 {
			return time.Now().Sub(startTime)
		}
	}
}

/*
	key: key of label
	value: value of label
	i：index of logs;
	channel: chan of task
	t: current time
*/
func send(key string, value string, i int, channel chan int, logTime string) {
	// time := time.Now().UnixNano() // 纳秒
	// t := strconv.FormatInt(int64(time), 10)
	url := "http://localhost:31000/loki/api/v1/push"
	logs := "line " + strconv.FormatInt(int64(i), 10) + " : test for log"
	//请求内容符合loki http api要求
	test := "{\"streams\":[{\"stream\":{\"" + key + "\":\"" + value + "\"},\"values\":[[\"" + logTime + "\",\"" + logs + "\"]]}]}"
	request, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(test)))
	if err != nil {
		fmt.Printf(" request err: http.NewRequest%v", err)
		//return queryobj, err
	}
	request.Header.Set("Content-Type", "application/json")
	client := http.Client{}
	startTime := time.Now()
	resp, err := client.Do(request.WithContext(context.TODO())) //发送请求
	costTime := time.Now().Sub(startTime)
	//fmt.Println(costTime)
	// 如果请求成功，将耗费时间传入finishedRequest通道，否则传入badRequest通道
	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("client.Do%v", err)
		badRequest[i] = costTime
		//badRequest <- costTime
	} else {
		//finishedRequest <- costTime
		finishedRequest[i] = costTime
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ioutil.ReadAll%v", err)
	}
	fmt.Print(string(respBytes))
	<-channel //表示该协程已经结束
	// res, err := http.Post(url, header, bytes.NewBuffer([]byte(test)))
	// if err != nil {
	// 	fmt.Println("Fatal error ", err.Error())
	// }
	// content, err := ioutil.ReadAll(res.Body)
	// if err != nil {
	// 	fmt.Println("Fatal error ", err.Error())
	// }
	// fmt.Println(string(content))
	// str := (*string)(unsafe.Pointer(&content)) //转化为string,优化内存
	// fmt.Println(*str)

}

func calculate(startTime time.Time, costTime time.Duration) {
	var times = []time.Duration{}
	fmt.Printf("\nFinished requests:\t%v\nFailed requests:\t%v\n\n", len(finishedRequest), len(badRequest))
	//fmt.Println(len(finishedRequest))
	//close(finishedRequest)
	for i := range finishedRequest {
		times = append(times, finishedRequest[i])
	}
	//fmt.Println(len(times))
	sort.Slice(times, func(i, j int) bool {
		return times[i] < times[j]
	})
	var sumTime time.Duration
	for i := range times {
		//fmt.Println(times[i])
		sumTime += times[i]
	}
	//fmt.Printf("\ncostTime for all requests: %v\n", costTime)
	count := time.Duration(len(times)) * time.Nanosecond
	meanTimes := sumTime / count
	meanTimesCurrentTask := costTime / count
	fmt.Printf("Time per request:\t%v (mean)\n", meanTimes)
	fmt.Printf("Time per request:\t%v (mean,across all concurrent requests)\n\n", meanTimesCurrentTask)
	//fmt.Printf("Requests per second:\t%v", int64(time.Second / times[1]))
	//fmt.Print(times[0])
	fmt.Println("Percentage of the requests served within a certain time:")
	for i := 50; i < 100; i = i + 10 {
		j := i * len(times) / 100
		fmt.Printf("%v%%\t%v\n", i, times[j-1])
	}
	fmt.Printf("%v%%\t%v\n", 95, times[95*len(times)/100-1])
	fmt.Printf("%v%%\t%v\n", 99, times[99*len(times)/100-1])
	fmt.Printf("%v%%\t%v(longest request)\n", 100, times[len(times)-1])

	fmt.Printf("\ntotal time for test:\t%v\n\n", time.Now().Sub(startTime))
}

func addTime(request map[int]time.Duration, t time.Duration) {

}

// func pushLogs(key string, value string, fre int, i int) {
// 	for {
// 		//fmt.Printf("push %v\n", i)
// 		//send(key, value, i)
// 		fmt.Printf("client %v send a log-------\n", i)
// 		time.Sleep(time.Second * time.Duration(fre))
// 	}
// 	// wg.Done()
// }

// key: key of label
// value: value of label
// i： client i
