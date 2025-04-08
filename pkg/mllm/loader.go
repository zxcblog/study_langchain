package mllm

import (
	"context"
	"errors"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"os"
	"path/filepath"
	"time"
)

var (
	ops = []textsplitter.Option{
		textsplitter.WithChunkSize(512),    // 切割后的大小块
		textsplitter.WithChunkOverlap(128), // 相邻文本之间的重叠
		textsplitter.WithCodeBlocks(true),  // 包含代码块
	}

	FilenameKey = "filename"
	UpdatedTime = "updated_time"
)

// SetSplitter 设置文本分割器
func (c *Client) SetSplitter(opts ...textsplitter.Option) {
	ops = opts
}

// AddSplitter 添加文本分割器
func (c *Client) AddSplitter(opts ...textsplitter.Option) {
	ops = append(ops, opts...)
}

func (c *Client) load(ctx context.Context, filename string) ([]schema.Document, []string, error) {
	f, err := os.Lstat(filename)
	if err != nil {
		return nil, nil, err
	}

	// 加载单个文件
	if !f.IsDir() {
		docs, err := c.loadFile(ctx, filename)
		return docs, []string{filename}, err
	}

	// 加载文件夹下的文件
	dir, err := os.ReadDir(filename)
	if err != nil {
		return nil, nil, err
	}

	docs := make([]schema.Document, 0, 100)
	files := make([]string, 0, 100)
	for _, d := range dir {
		rd, fs, err := c.load(ctx, filepath.Join(filename, d.Name()))
		if err != nil {
			return nil, nil, err
		}
		docs = append(docs, rd...)
		files = append(files, fs...)
	}
	return docs, files, nil
}

func (c *Client) loadFile(ctx context.Context, filename string) ([]schema.Document, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var (
		loader  documentloaders.Loader
		spliter textsplitter.TextSplitter
	)

	finfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// 定义元数据，方便进行获取
	metadata := map[string]string{
		FilenameKey: filename,
		UpdatedTime: finfo.ModTime().Format(time.DateTime),
	}
	ext := filepath.Ext(filename)
	switch ext {
	case ".md":
		loader = documentloaders.NewText(file)
		spliter = textsplitter.NewMarkdownTextSplitter(ops...)
	case ".txt":
		loader = documentloaders.NewText(file)
		spliter = textsplitter.NewRecursiveCharacter(ops...)
	case ".pdf":
		loader = documentloaders.NewPDF(file, finfo.Size())
		spliter = textsplitter.NewRecursiveCharacter(ops...)
	default:
		return nil, errors.New("不支持的文档类型:" + ext)
	}

	// 加载并拆分文档
	docs, err := loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	texts := make([]string, len(docs))
	metadatas := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		texts[i] = doc.PageContent
		meta := doc.Metadata
		for k, v := range metadata {
			if _, ok := meta[k]; !ok {
				meta[k] = v
			}
		}
		metadatas[i] = meta
	}

	return textsplitter.CreateDocuments(spliter, texts, metadatas)
}
