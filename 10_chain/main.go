package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../.env")

	ctx := context.Background()
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		Model:   os.Getenv("OPENAI_MODEL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		panic(err)
	}

	msg := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个{role},请用简洁专业的语言回答问题。"),
		schema.UserMessage("{input}"),
	)

	chain := compose.NewChain[map[string]any, string]()

	chain.AppendChatTemplate(msg)
	chain.AppendChatModel(chatModel)
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
		msg.Content = "我是Lambda：" + msg.Content
		return msg, nil
	}))
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
		return "最终结果：" + msg.Content, nil
	}))

	compile, err := chain.Compile(ctx)
	if err != nil {
		panic(err)
	}

	invoke, err := compile.Invoke(ctx, map[string]any{
		"role":  "天气助手",
		"input": "你好，我想了解一下武汉的天气，加点生活指标",
	})
	if err != nil {
		panic(err)
	}

	//fmt.Println(invoke.Content)
	fmt.Println(invoke)
}
