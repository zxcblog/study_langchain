package main

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/mongovector"
	"log"
	"study_langchain/pkg/store"
)

func main() {
	const (
		indexDP2048 = "vector_index_dotProduct_2048"
	)

	llm, err := ollama.New(ollama.WithModel("qwen2.5:3b"), ollama.WithServerURL("http://127.0.0.1:11434"))
	if err != nil {
		log.Fatalf("llm初始化失败：%s", err.Error())
	}

	uri := "mongodb://root:123456@127.0.0.1:27018/?directConnection=true"
	mongoClient, err := store.NewMongoStore(uri, "study", "vector")
	if err != nil {
		log.Fatalf("mongodb初始化失败：%s", err.Error())
	}
	defer mongoClient.Close(context.Background())

	if err = mongoClient.CreateIndex(context.Background(), indexDP2048); err != nil {
		log.Fatalf("创建索引失败：%s", err.Error())
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatal("failed to create an embedder: %v", err)
	}

	// 获取到mongodbVector
	vstore := mongovector.New(mongoClient.Coll(), embedder, mongovector.WithIndex(indexDP2048))
	_, err = vstore.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "euclidean — 测量向量两端之间的距离。 该值允许您根据不同的维度来衡量相似性。 要了解更多信息，请参阅 欧几里得。"},
		{PageContent: "cosine — 根据向量之间的角度衡量相似度。 通过该值，您可以衡量不按幅度缩放的相似度。 不能将零幅度向量与 cosine 一起使用。 要衡量余弦相似度，我们建议您对向量进行归一化并改用 dotProduct。"},
		{PageContent: "dotProduct - 衡量相似度，类似于 cosine，但还会考虑向量的长度。如果对幅度进行归一化，则 cosine 和 dotProduct 在衡量相似性方面几乎相同。"},
		{PageContent: "要使用 dotProduct，您必须在索引时和查询时将向量标准化为单位长度。"},
		{PageContent: "为获得最佳性能，请检查您的嵌入模型，以确定哪个相似度函数适合嵌入模型的培训进程。 如果没有任何指导，请从 dotProduct 开始。"},
		{PageContent: "将 fields.similarity 设置为 dotProduct 值可让您根据角度和幅度有效地衡量相似性。 dotProduct 比 cosine 消耗的计算资源更少，并且在向量为单位长度时非常高效。"},
		{PageContent: " 但是，如果您的向量未标准化，请评估 euclidean 距离和 cosine 相似度的示例查询结果中的相似度分数，以确定哪一个对应于合理的结果。"},
	})
	if err != nil {
		log.Fatal("文档添加失败: %s", err.Error())
	}

	// Search for similar documents.
	docs, err := vstore.SimilaritySearch(context.Background(), "dotProduct相似度", 4)
	for i, v := range docs {
		fmt.Println("dotProduct", i, v.PageContent)
	}

	// Search for similar documents using score threshold.
	docs, err = vstore.SimilaritySearch(context.Background(), "测量向量两端之间的距离。 该值允许您根据不同的维度来衡量相似性。 要了解更多信息，请参阅 欧几里得。", 1,
		vectorstores.WithScoreThreshold(0.7))
	for i, v := range docs {
		fmt.Println("测量向量两端之间的距离", i, v.PageContent)
	}
}
