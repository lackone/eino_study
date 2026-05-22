package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func graph1(model *openai.ChatModel) {
	ctx := context.Background()

	msg := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个技术文档助手，请根据用户的问题给出清晰的解答。"),
		schema.UserMessage("{input}"),
	)

	g := compose.NewGraph[map[string]any, *schema.Message]()

	g.AddChatTemplateNode("msg", msg)
	g.AddChatModelNode("model", model)
	g.AddLambdaNode("format", compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (message *schema.Message, err error) {
		msg.Content = "格式化：" + msg.Content
		return msg, nil
	}))

	g.AddEdge(compose.START, "msg")

	g.AddEdge("msg", "model")

	g.AddEdge("model", "format")

	g.AddEdge("format", compose.END)

	compile, err := g.Compile(ctx)
	if err != nil {
		panic(err)
	}

	invoke, err := compile.Invoke(ctx, map[string]any{
		"input": "什么是 goroutine 泄漏？怎么排查？",
	})

	if err != nil {
		panic(err)
	}

	fmt.Println(invoke.Content)
}

func graph2(model *openai.ChatModel) {
	ctx := context.Background()
	// 分类器：用 Lambda 做简单的关键词分类
	classifier := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		question := input["question"].(string)
		if strings.Contains(question, "代码") || strings.Contains(question, "编程") || strings.Contains(question, "bug") {
			input["category"] = "code"
		} else if strings.Contains(question, "部署") || strings.Contains(question, "运维") || strings.Contains(question, "服务器") {
			input["category"] = "ops"
		} else {
			input["category"] = "general"
		}
		return input, nil
	})

	// 三个不同角色的 Prompt 模板
	codeTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个资深Go语言开发者，擅长代码审查和问题排查，请回答用户的编程问题。"),
		schema.UserMessage("{question}"),
	)

	opsTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个运维专家，精通Linux、Docker和K8s，请回答用户的运维问题。"),
		schema.UserMessage("{question}"),
	)

	generalTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个友好的技术助手，请简洁地回答用户的问题。"),
		schema.UserMessage("{question}"),
	)

	// 构建 Graph
	graph := compose.NewGraph[map[string]any, *schema.Message]()

	// 添加节点
	_ = graph.AddLambdaNode("classifier", classifier)
	_ = graph.AddChatTemplateNode("code_tpl", codeTpl)
	_ = graph.AddChatTemplateNode("ops_tpl", opsTpl)
	_ = graph.AddChatTemplateNode("general_tpl", generalTpl)
	_ = graph.AddChatModelNode("model", model)

	// 定义条件路由
	_ = graph.AddEdge(compose.START, "classifier")
	_ = graph.AddBranch("classifier", compose.NewGraphBranch(
		// 条件函数：根据分类结果决定走哪个分支
		func(ctx context.Context, input map[string]any) (string, error) {
			category := input["category"].(string)
			return category + "_tpl", nil
		},
		// 分支映射：声明所有可能的下游节点
		map[string]bool{
			"code_tpl":    true,
			"ops_tpl":     true,
			"general_tpl": true,
		},
	))

	// 三条分支最终都汇聚到同一个模型节点
	_ = graph.AddEdge("code_tpl", "model")
	_ = graph.AddEdge("ops_tpl", "model")
	_ = graph.AddEdge("general_tpl", "model")
	_ = graph.AddEdge("model", compose.END)

	// 编译并运行
	runner, err := graph.Compile(ctx)
	if err != nil {
		log.Fatal("编译失败:", err)
	}

	// 测试不同类型的问题
	questions := []string{
		"Go代码里怎么避免goroutine泄漏？",
		"Docker容器部署时端口映射不生效怎么办？",
		"推荐几本学习分布式系统的书？",
	}

	for _, q := range questions {
		result, err := runner.Invoke(ctx, map[string]any{"question": q})
		if err != nil {
			log.Printf("问题: %s, 错误: %v\n", q, err)
			continue
		}
		fmt.Printf("问题: %s\n回答: %s\n\n", q, result.Content)
	}
}

func graph3(model *openai.ChatModel) {
	ctx := context.Background()
	// 两个不同视角的模板
	techTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个技术架构师，请从技术可行性角度分析这个需求，用一两句话概括。"),
		schema.UserMessage("{requirement}"),
	)
	productTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个产品经理，请从用户价值角度分析这个需求，用一两句话概括。"),
		schema.UserMessage("{requirement}"),
	)

	// 构建两条子链
	techChain := compose.NewChain[map[string]any, *schema.Message]()
	techChain.AppendChatTemplate(techTpl).AppendChatModel(model)

	productChain := compose.NewChain[map[string]any, *schema.Message]()
	productChain.AppendChatTemplate(productTpl).AppendChatModel(model)

	// 构建并行节点
	parallel := compose.NewParallel()
	parallel.AddGraph("tech", techChain)
	parallel.AddGraph("product", productChain)

	// 主链：并行执行 → 合并结果
	chain := compose.NewChain[map[string]any, string]()
	chain.
		AppendParallel(parallel).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, results map[string]any) (string, error) {
			techResult := results["tech"].(*schema.Message)
			productResult := results["product"].(*schema.Message)
			return fmt.Sprintf("【技术视角】%s\n\n【产品视角】%s",
				techResult.Content, productResult.Content), nil
		}))

	runner, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal("编译失败:", err)
	}

	result, err := runner.Invoke(ctx, map[string]any{
		"requirement": "为电商App添加AI智能客服功能",
	})
	if err != nil {
		log.Fatal("运行失败:", err)
	}

	fmt.Println(result)
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

	//graph1(chatModel)

	//graph2(chatModel)

	graph3(chatModel)
}
