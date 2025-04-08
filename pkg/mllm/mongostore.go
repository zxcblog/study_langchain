package mllm

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"time"
)

type MongodbStore struct {
	dbname   string
	collname string
	idx      string
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

	VectorSearchType = "vectorSearch"
)

type Vector struct {
	PageContent string         `bson:"page_content"`
	Metadata    map[string]any `bson:"metadata"`
}

func NewMongodbStore(uri, dbname, collname, idx string) (*MongodbStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	coll := client.Database(dbname).Collection(collname)
	return &MongodbStore{dbname: dbname, collname: collname, client: client, coll: coll, idx: idx}, nil
}

func (m *MongodbStore) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

func (m *MongodbStore) CreateCollection(ctx context.Context) error {
	return m.client.Database(m.dbname).CreateCollection(ctx, m.collname)
}

func (m *MongodbStore) SelectIndex(ctx context.Context, idx string) (bool, error) {
	indexs := m.coll.SearchIndexes()

	siOpts := options.SearchIndexes().SetName(idx).SetType(VectorSearchType)
	cursor, err := indexs.List(ctx, siOpts)
	if err != nil {
		return false, err
	}

	if cursor == nil || (cursor.Current == nil && !cursor.Next(ctx)) {
		return false, nil
	}

	name := cursor.Current.Lookup("name").StringValue()
	queryable := cursor.Current.Lookup("queryable").Boolean()

	return name == idx && queryable, nil
}

func (m *MongodbStore) CreateIndex(ctx context.Context, idx string, fields []Field) error {
	if len(fields) < 1 {
		return errors.New("索引字段不能为空")
	}

	indexs := m.coll.SearchIndexes()
	// 设置创建的索引类型为 vectorSearch
	siOpts := options.SearchIndexes().SetName(idx).SetType(VectorSearchType)

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

func (m *MongodbStore) SelectCollection(ctx context.Context) bool {
	colls, _ := m.client.Database(m.dbname).ListCollectionNames(ctx, bson.M{"name": m.collname})
	return len(colls) > 0
}
