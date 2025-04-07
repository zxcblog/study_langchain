package main

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// 单次通话
//func main() {
//	llm, err := ollama.New(ollama.WithModel("qwen2.5:3b"), ollama.WithServerURL("http://127.0.0.1:11434"))
//	if err != nil {
//		fmt.Printf("连接ollama错误:%s\n", err.Error())
//		return
//	}
//
//	res, err := llms.GenerateFromSinglePrompt(context.Background(), llm, "你是谁")
//	if err != nil {
//		fmt.Printf("请求失败：%s\n", err.Error())
//		return
//	}
//	fmt.Printf("返回信息%s\n", res)
//}

func main() {
	llm, err := ollama.New(ollama.WithModel("qwen2.5:3b"), ollama.WithServerURL("http://127.0.0.1:11434"))
	if err != nil {
		fmt.Printf("连接ollama错误:%s\n", err.Error())
		return
	}

	msg := []llms.MessageContent{
		{Role: llms.ChatMessageTypeSystem, Parts: []llms.ContentPart{llms.TextContent{Text: "你是一个熟读三国演义的专家，请尽可能的帮我回答与三国相关的问题。"}}},
		llms.TextParts(llms.ChatMessageTypeHuman, "请问刘备是谁"),
	}
	res, err := llm.GenerateContent(context.Background(), msg)
	if err != nil {
		fmt.Printf("请求失败：%s\n", err.Error())
		return
	}
	choices := res.Choices
	if len(choices) < 1 {
		fmt.Printf("模型返回空消息")
		return
	}

	fmt.Printf("返回信息%s\n", choices[0].Content)

	// 携带历史数据进行访问
	msg = append(msg, llms.TextParts(llms.ChatMessageTypeAI, choices[0].Content), llms.TextParts(llms.ChatMessageTypeHuman, "曹操呢"))
	res, err = llm.GenerateContent(context.Background(), msg)
	if err != nil {
		fmt.Printf("请求失败：%s\n", err.Error())
		return
	}
	choices = res.Choices
	if len(choices) < 1 {
		fmt.Printf("模型返回空消息")
		return
	}

	fmt.Printf("返回信息%s\n", choices[0].Content)
}
