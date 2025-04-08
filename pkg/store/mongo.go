package store

import (
	"context"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"time"
)

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
