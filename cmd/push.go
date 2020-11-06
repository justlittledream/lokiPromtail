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
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var finishedRequest = make([]int, 60001)
var badRequest = make([]int, 60001)
var requestCount = 0

var waitgroup sync.WaitGroup
var lock sync.Mutex

// const (
// 	MaxIdleConns        int = 30
// 	MaxIdleConnsPerHost int = 30
// 	IdleConnTimeout     int = 30
// )

type ClientConfig struct {
	PushURL string
	Stream  *bytes.Buffer
}

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push logs to loki",
	Long: `----------help----------------
	example: ./main push -u http://192.168.88.171:31000/loki/api/v1/push -t 1000 -m 1 -k app -v nginx-deployment
	`,
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		url, _ := cmd.Flags().GetString("url")
		key, _ := cmd.Flags().GetString("key")
		value, _ := cmd.Flags().GetString("value")
		task, _ := cmd.Flags().GetInt("task")
		max, _ := cmd.Flags().GetInt("max")
		fmt.Printf("\nPush %v logs to loki with max concurrent tasks = %v\n", task, max)
		costTime := sendLogs(url, key, value, task, max)
		calculate(costTime)
		fmt.Printf("\ntotal time for test:\t%v\n\n", time.Now().Sub(startTime))
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.PersistentFlags().StringP("url", "u", "http://localhost:3100/loki/api/v1/push", "url of loki http api")
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
func sendLogs(url, key, value string, task, max int) time.Duration {
	postTime := time.Now().UnixNano()                 // 纳秒
	logTime := strconv.FormatInt(int64(postTime), 10) //统一了推送日志流的时间
	startTime := time.Now()                           //记录执行发送第一条请求前的时间
	channel := make(chan http.Client, max)            //控制协程最大数
	defer func() {
		close(channel)
	}()
	var clientCount = 0
	waitgroup.Add(task)
	for i := 0; i < task; i++ {
		//符合loki http api要求的请求流，label为key=value，时间统一=logTime，日志=logs，只有i不同
		logs := "line " + strconv.FormatInt(int64(i), 10) + " : test for log"
		text := "{\"streams\":[{\"stream\":{\"" + key + "\":\"" + value + "\"},\"values\":[[\"" + logTime + "\",\"" + logs + "\"]]}]}"
		conf := ClientConfig{
			PushURL: url,
			Stream:  bytes.NewBuffer([]byte(text)),
		}

		//无空余连接，如果总量小于最大值，创建新的连接
		if len(channel) == 0 && clientCount < max {
			clientCount++
			//fmt.Println(clientCount)
			client := createHTTPClient()
			channel <- client
		}
		go send(conf, channel)
	}
	waitgroup.Wait()
	return time.Now().Sub(startTime)
}

/*
	key: key of label
	value: value of label
	i：index of logs;
	channel: chan of task
	t: current time
*/
func send(conf ClientConfig, channel chan http.Client) {
	client := <-channel
	defer func() {
		channel <- client
		waitgroup.Done()
	}()
	//url := "http://localhost:31000/loki/api/v1/push"
	request, err := http.NewRequest("POST", conf.PushURL, conf.Stream)
	if err != nil {
		fmt.Printf(" request err: http.NewRequest%v", err)

	}
	request.Header.Set("Content-Type", "application/json")
	startTime := time.Now()
	resp, err := client.Do(request) //发送请求
	costTime := time.Now().Sub(startTime)
	index := int(costTime/time.Millisecond) + 1
	//fmt.Printf("costTime: %v   Index: %v\n", costTime, index)
	if err != nil {
		fmt.Printf("client.Do%v", err)

		//badRequest <- costTime
	}

	if resp.StatusCode != 204 {
		//finishedRequest <- costTime
		addTime(badRequest, index)
	} else {
		addTime(finishedRequest, index)
	}

	defer resp.Body.Close()
	// respBytes, err := ioutil.ReadAll(resp.Body)

	// if err != nil {
	// 	fmt.Printf("ioutil.ReadAll%v", err)
	// }
	// if string(respBytes) != "" {
	// 	fmt.Println(string(respBytes))
	// }
	//return resp
}

func calculate(costTime time.Duration) {
	sumTime := 0
	count := 0 //总共的request数
	for i, v := range finishedRequest {
		if v != 0 {
			//fmt.Printf("index: %v   count: %v\n", i, v)
			sumTime += i * v
			count += v
		}

	}
	badCount := 0
	for _, v := range badRequest {
		//sumTime += i * v
		badCount += v
	}
	fmt.Printf("\nFinished requests:\t%v\nFailed requests:\t%v\n\n", count, badCount)

	meanTimes := sumTime/count + 1
	//fmt.Println(costTime)
	meanTimesCurrentTask := (float64(costTime/time.Millisecond) + 1) / float64(count)
	fmt.Printf("Time per request:\t%vms (mean)\n", meanTimes)
	fmt.Printf("Time per request:\t%vms (mean,across all concurrent requests)\n", meanTimesCurrentTask)
	qps, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", 1000/meanTimesCurrentTask), 64)
	fmt.Printf("Requests per second:\t%v\n\n", qps)
	//fmt.Printf("Requests per second:\t%v", int64(time.Second / times[1]))
	//fmt.Print(times[0])
	var index = 0
	// var percentage
	var output = make(map[int]int)
	for i, v := range finishedRequest {
		index += v
		//fmt.Println(i, v)
		if v != 0 {
			var per = index * 100 / count
			//fmt.Println(i, v, per)
			switch {
			case per == 100:
				if _, ok := output[100]; !ok {
					output[100] = i
				}
				fallthrough
				//fmt.Printf("%v%%\t%v(longest request)\n", 100, i)
			case per >= 99:
				if _, ok := output[99]; !ok {
					output[99] = i
				}
				fallthrough
				//fmt.Printf("%v%%\t%v\n", 99, i)
			case per >= 95:
				if _, ok := output[95]; !ok {
					output[95] = i
				}
				fallthrough
				//fmt.Printf("%v%%\t%v\n", 95, i)
			case per >= 90:
				if _, ok := output[90]; !ok {
					output[90] = i
				}
				fallthrough
				//fmt.Printf("%v%%\t%v\n", 90, i)
			case per >= 80:
				if _, ok := output[80]; !ok {
					output[80] = i
				}
				fallthrough
				//fmt.Printf("%v%%\t%v\n", 80, i)
			case per >= 70:
				if _, ok := output[70]; !ok {
					output[70] = i
				}
				fallthrough
				//fmt.Printf("%v%%\t%v\n", 70, i)
			case per >= 60:
				if _, ok := output[60]; !ok {
					output[60] = i
				}
				fallthrough
				//fmt.Printf("%v%%\t%v\n", 60, i)
			case per >= 50:
				if _, ok := output[50]; !ok {
					output[50] = i
				}
				//fmt.Printf("%v%%\t%v\n", 50, i)
			}
		}
	}
	printOut(output)
}

func addTime(request []int, i int) {
	lock.Lock()
	request[i]++
	lock.Unlock()
}

func printOut(output map[int]int) {
	fmt.Println("Percentage of the requests served within a certain time(ms):")
	fmt.Printf("%v%%\t%v\n", 50, output[50])
	fmt.Printf("%v%%\t%v\n", 60, output[60])
	fmt.Printf("%v%%\t%v\n", 70, output[70])
	fmt.Printf("%v%%\t%v\n", 80, output[80])
	fmt.Printf("%v%%\t%v\n", 90, output[90])
	fmt.Printf("%v%%\t%v\n", 95, output[95])
	fmt.Printf("%v%%\t%v\n", 99, output[99])
	fmt.Printf("%v%%\t%v\n", 100, output[100])
}

func createHTTPClient() http.Client {
	client := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			// MaxIdleConns:        MaxIdleConns,
			// MaxIdleConnsPerHost: MaxIdleConnsPerHost,
			// IdleConnTimeout:     time.Duration(IdleConnTimeout) * time.Second,
		},
		Timeout: 20 * time.Second,
	}
	return client
}
