package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	mcpTool "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func ConvertSlice[S any, T any](input []S) []T {
	result := make([]T, 0, len(input))
	for _, v := range input {
		if val, ok := any(v).(T); ok {
			result = append(result, val)
		}
	}
	return result
}

func WeatherTool() mcp.Tool {
	tool := mcp.NewTool("get_weather",
		mcp.WithDescription("获取城市的天气信息"),
		mcp.WithString("city", mcp.Required(), mcp.Description("城市名称")),
		mcp.WithArray("exts",
			mcp.Required(),
			mcp.Enum("extended", "forecast", "forecast", "hourly", "minutely", "indices"),
			mcp.Description("扩展信息：extended(扩展气象字段) forecast(多天预报) hourly(逐小时预报) minutely(分钟级降水预报) indices(18项生活指数)"),
			mcp.DefaultArray([]string{"extended", "forecast"}),
		),
	)
	return tool
}

func mcpServer() {
	newMCPServer := server.NewMCPServer("weather", mcp.LATEST_PROTOCOL_VERSION)
	tool := WeatherTool()
	newMCPServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := request.GetArguments()

		fmt.Printf("params: %#v\n", params)

		city, ok := params["city"].(string)
		if !ok || city == "" {
			return nil, fmt.Errorf("city is required")
		}

		baseURL := "https://uapis.cn/api/v1/misc/weather"
		queryParams := url.Values{}
		queryParams.Set("city", city)

		fmt.Printf("exts : %#v\n", params["exts"])

		if e, ok := params["exts"]; ok {
			var exts []string
			// 处理两种情况：数组 或 JSON字符串
			switch v := e.(type) {
			case []any:
				// 已经是数组，直接转换
				exts = ConvertSlice[any, string](v)
			case string:
				// 尝试解析为JSON数组
				var temp []string
				if err := json.Unmarshal([]byte(v), &temp); err != nil {
					// JSON解析失败，说明是单个字符串值
					exts = []string{v}
				} else {
					// JSON解析成功
					exts = temp
				}
			default:
				return nil, fmt.Errorf("unsupported exts type: %T", e)
			}
			for _, ext := range exts {
				queryParams.Set(ext, "true")
			}
		}

		fullURL := fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())

		fmt.Println(fullURL)

		// 发送HTTP请求
		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		// 读取响应
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		// 检查HTTP状态码
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		fmt.Println(string(body))

		return mcp.NewToolResultText(string(body)), nil
	})

	err := server.NewSSEServer(newMCPServer).Start(":8080")
	if err != nil {
		panic(err)
	}
}

func mcpClient() ([]tool.BaseTool, error) {
	ctx := context.Background()
	ssemcpClient, err := client.NewSSEMCPClient("http://localhost:8080/sse")
	if err != nil {
		panic(err)
	}
	err = ssemcpClient.Start(ctx)
	if err != nil {
		panic(err)
	}
	req := mcp.InitializeRequest{}
	req.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	req.Params.ClientInfo = mcp.Implementation{
		Name:    "weather-tool",
		Version: "1.0.0",
	}
	_, err = ssemcpClient.Initialize(ctx, req)
	if err != nil {
		panic(err)
	}
	tools, err := mcpTool.GetTools(ctx, &mcpTool.Config{
		Cli: ssemcpClient,
	})
	if err != nil {
		panic(err)
	}
	return tools, err
}

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

	tools, err := mcpClient()
	if err != nil {
		panic(err)
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
	})
	if err != nil {
		panic(err)
	}

	msg := []*schema.Message{
		schema.SystemMessage("请根据提供的天气查询工具，查询天气情况"),
		schema.UserMessage("查询武汉今天的天气，多天预报"),
	}

	result, err := agent.Generate(ctx, msg)
	if err != nil {
		panic(err)
	}

	for _, v := range result.ToolCalls {
		fmt.Println(v.Function.Name)
		fmt.Println(v.Function.Arguments)
	}

	println(result.Content)
}
