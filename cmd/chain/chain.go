package main

import (
	"context"
	"fmt"
	"log"
	"study_langchain/pkg/mllm"
)

func main() {
	ctx := context.Background()
	uri := "mongodb://root:123456@127.0.0.1:27018/?directConnection=true"
	idx := "vector_index_dotProduct_2048"

	client, err := mllm.NewLLM("qwen2.5:3b", "http://127.0.0.1:11434")
	if err != nil {
		log.Fatalf("llm初始化失败：%s", err.Error())
	}
	defer client.Close(ctx)

	if err = client.SetMongodbStore(ctx, uri, "study", "vector", idx); err != nil {
		log.Fatalf("初始化mongodb store失败：%s", err.Error())
	}

	_, err = client.AddDocuments(ctx, "./docs")
	if err != nil {
		log.Fatalf("保存向量数据失败：%s", err.Error())
	}

	res, err := client.Chain(ctx, "常见的分块策略包括")
	if err != nil {
		log.Fatalf("保存向量数据失败：%s", err.Error())
	}
	fmt.Println(res)
}
