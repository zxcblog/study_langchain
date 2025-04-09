package main

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	"log"
	"study_langchain/pkg/mllm"
)

func main() {
	client, err := mllm.NewLLM("qwen2.5:3b", "http://127.0.0.1:11434")
	if err != nil {
		log.Fatalf("llm初始化失败：%s", err.Error())
	}

	// 使用模板进行对话
	template := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate("你是一个翻译人员，只翻译文本，不进行解释", nil),
		prompts.NewHumanMessagePromptTemplate("将此文本从{{.inputLang}}转换为{{.outputLang}}:\n{{.input}}", []string{"inputLang", "outputLang", "input"}),
	})

	value, err := template.FormatPrompt(map[string]any{
		"inputLang":  "English",
		"outputLang": "Chinese",
		"input":      "I love programming",
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	msg := make([]llms.MessageContent, 0, len(value.Messages()))
	for _, v := range value.Messages() {
		msg = append(msg, llms.MessageContent{Role: v.GetType(), Parts: []llms.ContentPart{llms.TextPart(v.GetContent())}})
	}

	res, err := client.LLM.GenerateContent(context.Background(), msg)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(res.Choices[0].Content)
}
