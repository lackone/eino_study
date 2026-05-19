package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

// 创建Host Agent
func newHost(ctx context.Context) (*host.Host, error) {
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		Model:   os.Getenv("OPENAI_MODEL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		return nil, err
	}
	return &host.Host{
		ToolCallingModel: chatModel,
		SystemPrompt:     "你是一个日记助手，可以帮助用户写日记、读日记。调用提供的",
	}, nil
}

// 创建写日记专家
func newWriteJournalSpecialist(ctx context.Context) (*host.Specialist, error) {
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		Model:   os.Getenv("OPENAI_MODEL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		return nil, err
	}
	return &host.Specialist{
		ChatModel:    chatModel,
		SystemPrompt: "请将用户输入的内容写入日记。请勿返回任何内容。",
		AgentMeta: host.AgentMeta{
			Name:        "write_journal",
			IntendedUse: "将用户输入的内容写入日记",
		},
		Invokable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
			return &schema.Message{
				Role:    schema.Assistant,
				Content: "日记已保存",
			}, nil
		},
		Streamable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.StreamReader[*schema.Message], error) {
			return &schema.StreamReader[*schema.Message]{}, nil
		},
	}, nil
}

// 创建读日记专家
func newReadJournalSpecialist(ctx context.Context) (*host.Specialist, error) {
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		Model:   os.Getenv("OPENAI_MODEL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		return nil, err
	}
	return &host.Specialist{
		ChatModel:    chatModel,
		SystemPrompt: "请将日记内容返回给用户。请勿返回任何内容。",
		AgentMeta: host.AgentMeta{
			Name:        "view_journal_content",
			IntendedUse: "读取并显示日记内容",
		},
		Invokable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
			return &schema.Message{
				Role:    schema.Assistant,
				Content: "今天天气很好\n学习了Eino框架\n创建了一个Multi-Agent系统\n",
			}, nil
		},
		Streamable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.StreamReader[*schema.Message], error) {
			journal, err := readJournal()
			if err != nil {
				return nil, err
			}
			reader, writer := schema.Pipe[*schema.Message](0)
			go func() {
				scanner := bufio.NewScanner(journal)
				scanner.Split(bufio.ScanLines)

				for scanner.Scan() {
					line := scanner.Text()
					message := &schema.Message{
						Role:    schema.Assistant,
						Content: line + "\n",
					}
					writer.Send(message, nil)
				}

				if err := scanner.Err(); err != nil {
					writer.Send(nil, err)
				}

				writer.Close()
			}()

			return reader, nil
		},
	}, nil
}

// 模拟读取日记的函数
func readJournal() (io.Reader, error) {
	// 实际应用中这里会从文件或数据库读取内容
	content := "今天天气很好\n学习了Eino框架\n创建了一个Multi-Agent系统\n"
	return strings.NewReader(content), nil
}

func main() {
	godotenv.Load("../.env")

	ctx := context.Background()

	// 创建Host和专家Agents
	h, err := newHost(ctx)
	if err != nil {
		panic(err)
	}

	writer, err := newWriteJournalSpecialist(ctx)
	if err != nil {
		panic(err)
	}

	reader, err := newReadJournalSpecialist(ctx)
	if err != nil {
		panic(err)
	}

	// 创建Multi-Agent系统
	hostMA, err := host.NewMultiAgent(ctx, &host.MultiAgentConfig{
		Host: *h,
		Specialists: []*host.Specialist{
			writer,
			reader,
		},
	})
	if err != nil {
		panic(err)
	}

	// 交互式使用示例
	fmt.Println("=== 日记助手 ===")

	msg := &schema.Message{
		Role:    schema.User,
		Content: "读日志，今天学习了eino框架的multiagent",
	}

	// 流式获取结果
	out, err := hostMA.Generate(ctx, []*schema.Message{msg})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	}

	fmt.Print("助手: ")
	fmt.Println(out.Content)
}
