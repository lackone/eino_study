package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func FString() []*schema.Message {
	ctx := context.Background()
	messages := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个{role},请用{language}回答用户"),
		schema.UserMessage("{question}"),
	)
	result, err := messages.Format(ctx, map[string]any{
		"role":     "高中美少女",
		"language": "中文",
		"question": "你好",
	})
	if err != nil {
		panic(err)
	}
	return result
}

func GoTemplate() []*schema.Message {
	ctx := context.Background()
	messages := prompt.FromMessages(schema.GoTemplate,
		schema.SystemMessage("{{if .isExpert}}你是一个专家级{{.domain}}顾问。{{else}}你是一个初级{{.domain}}助手。{{end}}\\n{{if .isFormal}}请使用正式的语言风格。{{else}}请使用友好的语言风格。{{end}}\\n你的任务是{{.task}}。"),
		schema.UserMessage("{{.question}}"),
	)
	result, err := messages.Format(ctx, map[string]any{
		"isExpert": false,
		"domain":   "编程",
		"isFormal": false,
		"task":     "帮助初学者理解编程概念",
		"question": "什么是变量？",
	})
	for _, v := range result {
		println(v.Content)
	}
	if err != nil {
		panic(err)
	}
	return result
}

func Jinja2() []*schema.Message {
	ctx := context.Background()
	messages := prompt.FromMessages(schema.Jinja2,
		schema.SystemMessage("{% if level == 'expert' %}你是一个专家级顾问。{% else %}你是一个初级助手。{% endif %}\n{% if domain %}你专长于{{ domain }}领域。{% endif %}\n请用{% if formal %}正式{% else %}友好{% endif %}的语气回答问题。"),
		schema.UserMessage("{{question}}"),
	)
	result, err := messages.Format(ctx, map[string]any{
		"level":    "expert",
		"domain":   "人工智能",
		"formal":   true,
		"question": "请解释Transformer模型的工作原理。",
	})
	for _, v := range result {
		println(v.Content)
	}
	if err != nil {
		panic(err)
	}
	return result
}

func history() []*schema.Message {
	ctx := context.Background()
	messages := prompt.FromMessages(
		schema.GoTemplate,
		schema.SystemMessage(`你是一个{{.role}}，你的任务是{{.task}}。请参考之前的对话历史来回答当前的问题`),
		schema.MessagesPlaceholder("history", false), // 消息占位符，用于插入历史消息
		schema.UserMessage("{{.question}}"),
	)
	result, err := messages.Format(ctx, map[string]any{
		"role": "记忆器",
		"task": "根据用户提供的信息，给出准确的回答，如果历史中有答案，采用用户的答案",
		"history": []*schema.Message{
			//用户问题
			schema.UserMessage("你好，我想了解一下Go语言的并发机制"),
			//AI回答
			schema.AssistantMessage("Go语言提供了goroutines和channels来支持并发编程。Goroutines是轻量级线程，channels用于goroutines之间的通信。", nil),
			//用户问题
			schema.UserMessage("你错了，Go语言只提供了go关键字来支持并发"),
		},
		"question": "Go语言的并发机制",
	})
	for _, v := range result {
		println(v.Content)
	}
	if err != nil {
		panic(err)
	}
	return result
}

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

	//result := FString()

	//result := GoTemplate()

	//result := Jinja2()

	result := history()

	stream, err := model.Stream(ctx, result)
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
