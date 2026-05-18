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
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

type WeatherTool struct {
}

func ConvertSlice[S any, T any](input []S) []T {
	result := make([]T, 0, len(input))
	for _, v := range input {
		if val, ok := any(v).(T); ok {
			result = append(result, val)
		}
	}
	return result
}

func (w *WeatherTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析输入参数
	var params map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	fmt.Printf("params: %#v\n", params)

	city, ok := params["city"].(string)
	if !ok || city == "" {
		return "", fmt.Errorf("city is required")
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
			// 是JSON字符串，需要先解析
			var temp []string
			if err := json.Unmarshal([]byte(v), &temp); err != nil {
				return "", fmt.Errorf("failed to parse exts JSON: %w", err)
			}
			exts = temp
		default:
			return "", fmt.Errorf("unsupported exts type: %T", e)
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
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

func (w *WeatherTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "weather",
		Desc: "获取城市的天气信息",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"city": {
				Type:     schema.String,
				Required: true,
				Desc:     "城市名称",
			},
			"exts": {
				Type: schema.Array,
				Desc: "扩展信息：extended(扩展气象字段) forecast(多天预报) hourly(逐小时预报) minutely(分钟级降水预报) indices(18项生活指数)",
				Enum: []string{"extended", "forecast", "hourly", "minutely", "indices"},
			},
		}),
	}, nil
}

func NewWeatherTool() *WeatherTool {
	return &WeatherTool{}
}

func testTool() {
	ctx := context.Background()
	wtool := NewWeatherTool()
	node, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{wtool},
	})
	if err != nil {
		panic(err)
	}
	input := schema.AssistantMessage("你好", []schema.ToolCall{
		{
			Function: schema.FunctionCall{
				Name:      "weather",
				Arguments: `{"city": "武汉","exts": ["extended", "forecast", "hourly", "minutely", "indices"]}`,
			},
		},
	})
	invoke, err := node.Invoke(ctx, input)
	if err != nil {
		panic(err)
	}
	for _, v := range invoke {
		fmt.Printf("%v\n", v.Content)
	}
}

func useTool(model *openai.ChatModel) {
	ctx := context.Background()
	wtool := NewWeatherTool()
	node, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{wtool},
	})
	if err != nil {
		panic(err)
	}

	wtoolInfo, _ := wtool.Info(ctx)
	toolCallingModel, err := model.WithTools([]*schema.ToolInfo{wtoolInfo})
	if err != nil {
		panic(err)
	}

	result, err := toolCallingModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是一个天气助手"),
		schema.UserMessage("你好，我想了解一下武汉的天气，加点生活指标"),
	})
	if err != nil {
		panic(err)
	}

	for _, v := range result.ToolCalls {
		fmt.Println(v.Function.Name)
		fmt.Println(v.Function.Arguments)
	}

	fmt.Printf(result.Content)
	fmt.Println("===================")

	invoke, err := node.Invoke(ctx, result)
	if err != nil {
		panic(err)
	}

	for _, v := range invoke {
		fmt.Println(v.Content)
	}
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

	useTool(model)
}
