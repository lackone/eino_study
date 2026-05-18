package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../.env")

	fmt.Println(os.Getwd())
	fmt.Println(os.Getenv("OPENAI_BASE_URL"))

	ctx := context.Background()
	model, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
		Model:   os.Getenv("EMBEDDER"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		panic(err)
	}
	embeddings, err := model.EmbedStrings(ctx, []string{"你好"})
	if err != nil {
		panic(err)
	}

	fmt.Println(embeddings)
}
