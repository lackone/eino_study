package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../.env")

	fmt.Println(os.Getwd())
	fmt.Println(os.Getenv("OPENAI_BASE_URL"))

	ctx := context.Background()
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   os.Getenv("OPENAI_MODEL"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		panic(err)
	}

	msg := []*schema.Message{
		schema.SystemMessage("你是一个可爱的高中美少女"),
		schema.UserMessage("你好"),
	}

	stream, err := model.Stream(ctx, msg)
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println()
}
