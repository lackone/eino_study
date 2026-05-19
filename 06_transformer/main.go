package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/html"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/semantic"
	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

var htmlContent = `<!DOCTYPE html>
<html>
<body>
    <div>
        <h1>H1</h1>
        <p>H1 content1</p>
        <div>
            <h2>H2.1</h2>
            <p>H2.1 content</p>
            <h3>H3.1</h3>
            <p>H3.1 content</p>
            <h3>H3.2</h3>
            <p>H3.2 content</p>
            <h2>H2.2</h2>
            <p>H2.2 content</p>
        </div>
        <div>
            <h2>H2.3</h2>
            <p>H2.3 content</p>
        </div>
		<div>
			<p>H1 content2</p>
		</div>
        <br>
        <p>H1 content3</p>
    </div>
	<div>
		<h2>H2.4</h2>
		<p>H2.4 content</p>
	</div>
	<div>
		<p>content</p>
	</div>
</body>
</html>`

func htmlSplitter() {
	ctx := context.Background()
	splitter, err := html.NewHeaderSplitter(ctx, &html.HeaderConfig{
		Headers: map[string]string{
			"h1": "h1",
			"h2": "h2",
			"h3": "h3",
			"h4": "h4",
			"h5": "h5",
			"h6": "h6",
		},
	})
	if err != nil {
		panic(err)
	}
	docs, err := splitter.Transform(ctx, []*schema.Document{
		{
			ID:      "1",
			Content: htmlContent,
		},
	})
	if err != nil {
		panic(err)
	}
	for _, doc := range docs {
		fmt.Println(doc.Content)
		fmt.Println("==================")
	}
}

func mdSplitter() {
	ctx := context.Background()

	splitter, err := markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{
		Headers: map[string]string{
			"#":      "h1",
			"##":     "h2",
			"###":    "h3",
			"####":   "h4",
			"#####":  "h5",
			"######": "h6",
		},
		TrimHeaders: true,
	})
	if err != nil {
		panic(err)
	}
	docs, err := splitter.Transform(ctx, []*schema.Document{
		{
			ID: "1",
			Content: `
# 这是一级标题
这是一级标题内容
## 这是二级标题
这是二级标题内容
### 这是三级标题
这是三级标题内容
`,
		},
	})
	if err != nil {
		panic(err)
	}
	for _, doc := range docs {
		fmt.Println(doc.Content)
		fmt.Println("==================")
	}
}

func recursiveSplitter() {
	ctx := context.Background()
	splitter, err := recursive.NewSplitter(ctx, &recursive.Config{
		ChunkSize:   10,                                 // 必需：目标片段大小
		OverlapSize: 2,                                  // 可选：片段重叠大小
		Separators:  []string{"\n", ".", "?", "！", "!"}, // 可选：分隔符列表
		LenFunc:     nil,                                // 可选：自定义长度计算函数
		KeepType:    recursive.KeepTypeNone,             // 可选：分隔符保留策略
	})
	if err != nil {
		panic(err)
	}
	docs, err := splitter.Transform(ctx, []*schema.Document{
		{
			ID: "1",
			Content: `
			这是第一个段落，包含了一些内容。
            
            这是第二个段落。这个段落有多个句子！这些句子通过标点符号分隔。
            
            这是第三个段落。这里有更多的内容。`,
		},
	})
	if err != nil {
		panic(err)
	}
	for _, doc := range docs {
		println(doc.String())
		println("=========================")
	}
}

func semanticSplitter() {
	godotenv.Load("../.env")
	ctx := context.Background()
	embedder, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
		Model:   os.Getenv("EMBEDDER"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		panic(err)
	}
	splitter, err := semantic.NewSplitter(ctx, &semantic.Config{
		Embedding:    embedder,                      // 必需：用于生成文本向量的嵌入器
		BufferSize:   2,                             // 可选：上下文缓冲区大小
		MinChunkSize: 100,                           // 可选：最小片段大小
		Separators:   []string{"\n", ".", "?", "！"}, // 可选：分隔符列表
		Percentile:   0.9,                           // 可选：分割阈值百分位数
		LenFunc:      nil,                           // 可选：自定义长度计算函数
	})
	if err != nil {
		panic(err)
	}
	docs, err := splitter.Transform(ctx, []*schema.Document{
		{
			ID: "1",
			Content: `这是第一段内容，包含了一些重要信息。
            这是第二段内容，与第一段语义相关。
            这是第三段内容，主题已经改变。
            这是第四段内容，继续讨论新主题。`,
		},
	})
	if err != nil {
		panic(err)
	}
	for _, doc := range docs {
		println(doc.String())
		println("=========================")
	}
}

func main() {
	//htmlSplitter()

	//mdSplitter()

	//recursiveSplitter()

	semanticSplitter()
}
