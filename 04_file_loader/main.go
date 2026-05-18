package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino-ext/components/document/loader/url"
	"github.com/cloudwego/eino-ext/components/document/parser/docx"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/document/parser"
)

func loadFile() {
	ctx := context.Background()
	loader, err := file.NewFileLoader(ctx, &file.FileLoaderConfig{
		UseNameAsID: true,
	})
	if err != nil {
		panic(err)
	}
	docs, err := loader.Load(ctx, document.Source{
		URI: "test.md",
	})
	if err != nil {
		panic(err)
	}

	for _, doc := range docs {
		println(doc.ID)
		println(doc.Content)
	}
}

func loadDocx() {
	ctx := context.Background()
	docxParser, err := docx.NewDocxParser(ctx, &docx.Config{})
	if err != nil {
		panic(err)
	}
	extParser, err := parser.NewExtParser(ctx, &parser.ExtParserConfig{
		FallbackParser: parser.TextParser{}, // 最后一个解析器，用于处理所有其他文件类型
		Parsers: map[string]parser.Parser{
			".docx": docxParser,
		},
	})
	if err != nil {
		panic(err)
	}
	loader, err := file.NewFileLoader(ctx, &file.FileLoaderConfig{
		UseNameAsID: true,
		Parser:      extParser,
	})
	if err != nil {
		panic(err)
	}
	docs, err := loader.Load(ctx, document.Source{
		URI: "test.docx",
	})

	if err != nil {
		panic(err)
	}

	for _, doc := range docs {
		println(doc.ID)
		println("===================")
		println(doc.Content)

		for k, v := range doc.MetaData {
			fmt.Printf("%v %v\n", k, v)
		}
	}
}

func loadUrl() {
	ctx := context.Background()
	urlLoader, err := url.NewLoader(ctx, &url.LoaderConfig{})
	if err != nil {
		panic(err)
	}

	docs, err := urlLoader.Load(ctx, document.Source{
		URI: "https://www.baidu.com/",
	})
	if err != nil {
		panic(err)
	}

	for _, doc := range docs {
		println(doc.ID)
		println(doc.Content)
	}
}

func main() {
	//loadFile()

	//loadDocx()

	loadUrl()
}
