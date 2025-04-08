package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"log"
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
)

func main() {
	docs, err := LoadDoc(context.Background(), "./docs")
	if err != nil {
		log.Fatalf("文件加载失败：%s", err.Error())
	}

	for _, doc := range docs {
		fmt.Printf("%s\n", doc)
	}
}

// LoadDoc 对目录或单个文件进行加载
func LoadDoc(ctx context.Context, fileName string) ([]schema.Document, error) {
	f, err := os.Lstat(fileName)
	if err != nil {
		return nil, err
	}

	// 加载单个文件
	if !f.IsDir() {
		return LoadFile(ctx, fileName)
	}

	// 加载文件夹下的文件
	dir, err := os.ReadDir(fileName)
	if err != nil {
		return nil, err
	}

	docs := make([]schema.Document, 0, 100)
	for _, d := range dir {
		rd, err := LoadDoc(ctx, filepath.Join(fileName, d.Name()))
		if err != nil {
			return nil, err
		}
		docs = append(docs, rd...)
	}
	return docs, nil
}

// LoadFile 读取文件内容信息
func LoadFile(ctx context.Context, fileName string) ([]schema.Document, error) {
	file, err := os.Open(fileName)
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
	metadata := map[string]any{
		"filename":     fileName,
		"updated_time": finfo.ModTime().Format(time.DateTime),
	}
	ext := filepath.Ext(fileName)
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

	//// 不添加元数据，直接返回切割后的数据信息
	//return loader.LoadAndSplit(ctx, spliter)

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
