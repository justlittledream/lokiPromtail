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

var wg sync.WaitGroup

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push logs to loki",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		key, _ := cmd.Flags().GetString("key")
		value, _ := cmd.Flags().GetString("value")
		//size, _ := cmd.Flags().GetInt("size")
		fre, _ := cmd.Flags().GetInt("frequency")
		number, _ := cmd.Flags().GetInt("number")
		// fmt.Println(number)
		// fmt.Printf("%v = %v", key, value)
		// fmt.Println(fre)
		wg.Add(number)
		for i := 1; i <= number; i++ {
			//fmt.Println(i)
			//time.Sleep(time.Second)
			go pushLogs(key, value, fre, i)
		}
		//time.Sleep(time.Duration(1) * time.Hour)
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.PersistentFlags().StringP("key", "k", "app", "key of label")
	pushCmd.PersistentFlags().StringP("value", "v", "server", "value of label")
	pushCmd.PersistentFlags().IntP("size", "s", 0, "size of logs")
	pushCmd.PersistentFlags().IntP("frequency", "p", 1, "send a log every p seconds")
	pushCmd.PersistentFlags().IntP("number", "n", 2, "the number of clients sending logs")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pushCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// pushCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// key: key of label
// value: value of label
// fre: send logs every fre seconds every client
// i： client i

func pushLogs(key string, value string, fre int, i int) {
	for {
		//fmt.Printf("push %v\n", i)
		send(key, value, i)
		fmt.Printf("client %v send a log-------\n", i)

		time.Sleep(time.Second * time.Duration(fre))
	}
	wg.Done()
}

// key: key of label
// value: value of label
// i： client i
func send(key string, value string, i int) {
	//json
	// logs := "test for push"
	// var values [1][2]string = [1][2]string{
	// 	{t, logs},
	// }
	// var label = "{\"app\" :\"server\"}"
	// //fmt.Println(label)
	// var streams []promtailStream
	// streams = append(streams, promtailStream{
	// 	Labels: label,
	// 	Values: values,
	// })
	// msg := promtailMsg{
	// 	Streams: streams,
	// }
	// jsonString, _ := json.Marshal(msg)
	// //fmt.Println(jsonString)
	// fmt.Println(string(jsonString))
	time := time.Now().Unix()
	t := strconv.FormatInt(int64(time), 10) + "000000000"
	url := "http://localhost:31001/loki/api/v1/push"
	logs := "client " + strconv.FormatInt(int64(i), 10) + " : test for log"
	test := "{\"streams\":[{\"stream\":{\"" + key + "\":\"" + value + "\"},\"values\":[[\"" + t + "\",\"" + logs + "\"]]}]}"
	//fmt.Println(test)
	// var jsonstr = []byte(test)
	// buffer := bytes.NewBuffer(jsonstr)
	request, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(test)))
	if err != nil {
		fmt.Printf(" request err: http.NewRequest%v", err)
		//return queryobj, err
	}
	request.Header.Set("Content-Type", "application/json")
	client := http.Client{}
	resp, err := client.Do(request.WithContext(context.TODO())) //发送请求
	if err != nil {
		fmt.Printf("client.Do%v", err)
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ioutil.ReadAll%v", err)
	}
	fmt.Print(string(respBytes))

	// res, err := http.Post(url, header, bytes.NewBuffer([]byte(test)))
	// if err != nil {
	// 	fmt.Println("Fatal error ", err.Error())
	// }
	// defer res.Body.Close()
	// content, err := ioutil.ReadAll(res.Body)
	// if err != nil {
	// 	fmt.Println("Fatal error ", err.Error())
	// }
	// fmt.Println(string(content))
	// str := (*string)(unsafe.Pointer(&content)) //转化为string,优化内存
	// fmt.Println(*str)

}
