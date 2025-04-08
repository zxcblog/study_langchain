package main

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/mongovector"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"log"
	"time"
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
	mongoClient, err := NewMongoStore(uri, "study", "vector")
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

// 使用 mongodb-atlas 来存储向量
type MongoStore struct {
	uri      string // mongodb地址
	dbName   string // 数据库名称
	collName string // 集合名称
	index    string // 索引名称
	client   *mongo.Client
	coll     *mongo.Collection
}

// Field 添加索引是需要使用的字段
// https://www.mongodb.com/zh-cn/docs/atlas/atlas-vector-search/vector-search-type/
type Field struct {
	Type          FieldType       `bson:"type,omitempty"`
	Path          string          `bson:"path,omitempty"`
	NumDimensions int             `bson:"numDimensions,omitempty"`
	Similarity    FieldSimilarity `bson:"similarity,omitempty"`
}

type FieldType string
type FieldSimilarity string

const (
	FieldTypeVector FieldType = "vector" // 包含向量嵌入的字段
	FieldTypeFilter FieldType = "filter" // 适用于包含布尔值、日期、objectId、数字、字符串或 UUID 值的字段

	FieldSimilarityEuclidean  FieldSimilarity = "euclidean"  // 测量向量两端之间的距离
	FieldSimilarityCosine     FieldSimilarity = "cosine"     // 根据向量之间的角度衡量相似度
	FieldSimilarityDotProduct FieldSimilarity = "dotProduct" // 与 cosine 类似地衡量相似度，但会考虑向量的大小

	QwenAIEmbeddingDim = 2048 // 千问模型转换emb后的数值

	VectorSearchType = "vectorSearch"
)

// NewMongoStore 获取到mongodb连接实例
func NewMongoStore(uri, dbName, collName string) (*MongoStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	coll := client.Database(dbName).Collection(collName)
	return &MongoStore{uri: uri, dbName: dbName, collName: collName, client: client, coll: coll}, nil
}

func (m *MongoStore) GetQwenField() Field {
	return Field{
		Type:          FieldTypeVector,
		Path:          "plot_embedding", // mongodb vector默认字段
		NumDimensions: QwenAIEmbeddingDim,
		Similarity:    FieldSimilarityDotProduct,
	}

}

// CreateIndex 创建 vectorSearch 索引， 如果没有传递字段，则使用默认字段创建索引
// 使用mongodb-atlas做矢量库时， 需要对添加索引，否则查询时会查不到结果
func (m *MongoStore) CreateIndex(ctx context.Context, idx string, fields ...Field) error {
	ok, err := m.SelectIndex(ctx, idx)
	if ok || err != nil {
		return err
	}

	// 创建集合，
	if err = m.client.Database(m.dbName).CreateCollection(ctx, m.collName); err != nil {
		return err
	}

	indexs := m.coll.SearchIndexes()
	// 设置创建的索引类型为 vectorSearch
	siOpts := options.SearchIndexes().SetName(idx).SetType(VectorSearchType)
	if len(fields) < 1 {
		fields = []Field{m.GetQwenField()}
	}

	// 创建索引
	searchName, err := indexs.CreateOne(ctx, mongo.SearchIndexModel{Definition: bson.M{"fields": fields}, Options: siOpts})
	if err != nil {
		return err
	}

	// 判断索引有没有创建好， 没有就等待5秒
	var doc bson.Raw
	for doc == nil {
		cursor, err := indexs.List(ctx, options.SearchIndexes().SetName(searchName))
		if err != nil {
			return err
		}

		if !cursor.Next(ctx) {
			break
		}

		name := cursor.Current.Lookup("name").StringValue()
		queryable := cursor.Current.Lookup("queryable").Boolean()
		if name == searchName && queryable {
			doc = cursor.Current
		} else {
			time.Sleep(5 * time.Second)
		}
	}

	return nil
}

// SelectIndex 查询索引是否存在
func (m *MongoStore) SelectIndex(ctx context.Context, idx string) (bool, error) {
	indexs := m.coll.SearchIndexes()

	siOpts := options.SearchIndexes().SetName(idx).SetType(VectorSearchType)
	cursor, err := indexs.List(ctx, siOpts)
	if err != nil {
		return false, err
	}

	if cursor == nil {
		return false, nil
	}

	if cursor.Current == nil {
		if ok := cursor.Next(ctx); !ok {
			return false, nil
		}
	}

	name := cursor.Current.Lookup("name").StringValue()
	queryable := cursor.Current.Lookup("queryable").Boolean()

	return name == idx && queryable, nil
}

// Coll 获取coll集合
func (m *MongoStore) Coll() *mongo.Collection {
	return m.coll
}

// Close 关闭mongodb连接
func (m *MongoStore) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}
