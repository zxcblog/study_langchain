package main

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/outputparser"
	"github.com/tmc/langchaingo/prompts"
	"log"
	"study_langchain/pkg/mllm"
)

type Trans struct {
	Text string `json:"text" describe:"翻译后文本"`
}

func main() {
	examplePrompt := prompts.NewPromptTemplate("例：\n将此文本从{{.inputLang}}转换为{{.outputLang}}:\n{{.input}}\n```json\n {\"text\":\"{{.output}}\"} \n```", []string{"inputLang", "outputLang", "input", "output"})
	examples := []map[string]string{{"inputLang": "English", "outputLang": "Chinese", "input": "I love programming", "output": "我爱编程"}}

	p, err := prompts.NewFewShotPrompt(examplePrompt, examples, nil,
		"你是一个翻译人员，只翻译文本，不对文本进行解释。", "请开始你的回答: 将此文本从{{.inputLang}}转换为{{.outputLang}}: {{.question}}",
		[]string{"question", "inputLang", "outputLang"}, map[string]interface{}{"type": func() string { return "json" }},
		"\n", prompts.TemplateFormatGoTemplate, true)
	if err != nil {
		log.Fatal(err)
	}

	msg, err := p.Format(map[string]any{"inputLang": "English", "outputLang": "Chinese", "question": "What a nice day today"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(msg)

	client, err := mllm.NewLLM("qwen2.5:3b", "http://127.0.0.1:11434")
	if err != nil {
		log.Fatalf("llm初始化失败：%s", err.Error())
	}

	res, err := client.LLM.Call(context.Background(), msg)
	if err != nil {
		log.Fatal(err.Error())
	}

	// 对返回结果进行解析
	output, err := outputparser.NewDefined(Trans{})
	if err != nil {
		log.Fatal(err.Error())
	}
	trans, err := output.Parse(res)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(trans)
}
