package mllm

import (
	"context"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/mongovector"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Client struct {
	*ollama.LLM
	model string
	mongo *MongodbStore
	store *mongovector.Store
	emb   *embeddings.EmbedderImpl
}

func NewLLM(model, uri string, opts ...ollama.Option) (*Client, error) {
	opts = append(opts, ollama.WithModel(model), ollama.WithServerURL(uri))
	llm, err := ollama.New(opts...)
	if err != nil {
		return nil, err
	}

	return &Client{LLM: llm, model: model}, nil
}

func (c *Client) GetModel() string {
	return c.model
}

func (c *Client) SetMongodbStore(ctx context.Context, uri, dbname, collname, idx string, fields ...Field) (err error) {
	c.mongo, err = NewMongodbStore(uri, dbname, collname, idx)
	if err != nil {
		return err
	}

	// 创建集合
	if !c.mongo.SelectCollection(ctx) {
		if err = c.mongo.CreateCollection(ctx); err != nil {
			return
		}
	}

	// 创建索引
	if flag, _ := c.mongo.SelectIndex(ctx, idx); !flag {
		if len(fields) < 1 {
			fields = append(fields, Field{
				Type:          FieldTypeVector,
				Path:          "plot_embedding",
				NumDimensions: 2048,
				Similarity:    FieldSimilarityDotProduct,
			})
		}
		return c.mongo.CreateIndex(ctx, idx, fields)
	}
	return nil
}

func (c *Client) GetStore() (*mongovector.Store, error) {
	if c.store != nil {
		return c.store, nil
	}

	var err error
	c.emb, err = embeddings.NewEmbedder(c.LLM)
	if err != nil {
		return nil, err
	}

	// 获取到mongodbVector
	store := mongovector.New(c.mongo.coll, c.emb, mongovector.WithIndex(c.mongo.idx))
	c.store = &store
	return c.store, nil
}

func (c *Client) AddDocuments(ctx context.Context, filename string) ([]string, error) {
	docs, files, err := c.load(ctx, filename)
	if err != nil {
		return nil, err
	}

	// 获取表中所有数据
	list := make([]Vector, 0, len(files))
	cursor, err := c.mongo.coll.Find(ctx, bson.M{"metadata.filename": bson.M{"$in": files}})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	fileExistsMap := make(map[string]string)
	for _, doc := range list {
		k := doc.Metadata[FilenameKey].(string)
		fileExistsMap[k] = doc.Metadata[UpdatedTime].(string)
	}

	// 去除重复文档数据
	newdocs := make([]schema.Document, 0, len(docs))
	for _, doc := range docs {
		key := doc.Metadata[FilenameKey].(string)
		val := doc.Metadata[UpdatedTime].(string)
		if fileExistsMap[key] != val {
			newdocs = append(newdocs, doc)
		}
	}
	if len(newdocs) < 1 {
		return []string{}, nil
	}

	store, err := c.GetStore()
	if err != nil {
		return nil, err
	}
	return store.AddDocuments(ctx, newdocs)
}

func (c *Client) Close(ctx context.Context) (err error) {
	if c.mongo != nil {
		err = c.mongo.Close(ctx)
	}
	return err
}

func (c *Client) Chain(ctx context.Context, query string) (map[string]any, error) {
	store, err := c.GetStore()
	if err != nil {
		return nil, err
	}
	qa := chains.NewRetrievalQAFromLLM(c.LLM, vectorstores.ToRetriever(store, 10))
	return qa.Call(ctx, map[string]interface{}{"query": query})
}
