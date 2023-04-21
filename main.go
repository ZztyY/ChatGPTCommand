package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	OpenaiApiKey string `json:"openai_api_key"`
	OpenaiProxy string `json:"openai_proxy"`
}

func main() {
	// 读取 Config JSON 文件
	filePath := "config.json"
	configFile, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}
	var config Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		fmt.Println("配置文件JSON序列化失败", err)
	}
	if config.OpenaiProxy == "" || config.OpenaiApiKey == "" {
		fmt.Println("请配置 config.json, 填入你的 openai api key 和代理地址")
		time.Sleep(time.Second * 5)
		os.Exit(0)
	}
	fmt.Println("欢迎使用命令行程序，按下 Ctrl + C 可以退出。")

	// 创建一个信号通道
	signalChan := make(chan os.Signal, 1)
	// 监听 Interrupt 信号（Ctrl + C）
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		// 循环监听用户输入
		for {
			fmt.Print("提问：")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			command := scanner.Text()

			// 调用处理输入的函数
			result := processCommand(command, config)

			// 输出回复
			fmt.Println("回复：", result)
		}
	}()

	// 阻塞等待信号
	<-signalChan

	fmt.Println("接收到退出信号，程序即将退出...")
	// 在这里可以执行一些清理操作

	// 正常退出程序
	os.Exit(0)
}

type Message struct {
	Role string `json:"role"`
	Content string `json:"content"`
}

type GPT3Request struct {
	Model string `json:"model"`
	Messages []Message `json:"messages"`
}

type GPT3Response struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
		Index        int64  `json:"index"`
	} `json:"choices"`
}

// 处理输入的函数
func processCommand(command string, config Config) string {
	// 定义请求的URL和请求体
	url := config.OpenaiProxy

	// 创建一个请求体
	requestBody := GPT3Request{
		Model: "gpt-3.5-turbo",
		Messages: []Message {
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: command},
		},
	}

	// 将请求体序列化为JSON格式
	payload, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println(err)
		return "JSON序列化失败"
	}

	// 创建一个请求体
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Print(err)
		return "创建请求失败"
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer " + config.OpenaiApiKey)

	// 创建HTTP客户端
	client := &http.Client{}

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "发送请求失败"
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	// 检查响应状态码
	if resp.StatusCode == http.StatusOK {
		// 读取响应体
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return "读取响应体失败"
		}
		// 解析JSON响应
		var response GPT3Response
		err = json.Unmarshal(respBody, &response)
		if err != nil {
			fmt.Println(err)
			return "解析JSON响应失败"
		}
		generatedText := response.Choices[0].Message.Content
		return generatedText
	} else {
		fmt.Println(resp.StatusCode)
		return "POST请求失败"
	}
}
