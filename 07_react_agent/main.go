package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

type UserRequest struct {
	Name  string `json:name`
	Email string `json:email`
}

type UserResponse struct {
	Name     string `json:name`
	Email    string `json:email`
	Company  string `json:company`
	Position string `json:position`
	Salary   string `json:salary`
}

func main() {
	ctx := context.Background()

	godotenv.Load("../.env")

	fmt.Println(os.Getwd())
	fmt.Println(os.Getenv("OPENAI_BASE_URL"))

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   os.Getenv("OPENAI_MODEL"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		panic(err)
	}

	tt := utils.NewTool(&schema.ToolInfo{
		Name: "get_user_info",
		Desc: "Get user info by name and email",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"name": {
				Type: schema.String,
				Desc: "用户姓名",
			},
			"email": {
				Type: schema.String,
				Desc: "用户邮箱",
			},
		}),
	}, func(ctx context.Context, req *UserRequest) (*UserResponse, error) {
		return &UserResponse{
			Name:     req.Name,
			Email:    req.Email,
			Company:  "阿里",
			Position: "前端开发",
			Salary:   "50000",
		}, nil
	})

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{tt},
		},
		MessageRewriter: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(`
			你是一名专业的房产经纪人。你的任务是根据用户的职位和薪酬信息，为其推荐最合适的房产。
			请严格遵循以下步骤：
			1. 首先，调用 get_user_info 工具获取用户的详细信息（公司、职位、薪酬）。
			2. 然后，根据下方提供的“房产信息”和“购房建议规则”，为用户生成一份个性化的购房建议。
			3. 推荐时要明确说明为什么这个房产适合他，比如预算匹配度、通勤便利性、生活品质等。
			
			--- 房产信息 ---
			
			### A. 楼盘列表
			
			**1. 瀚海星辰 (ID: A-01)**
			- **区域**: 海淀区-中关村
			- **特点**: 顶级学区房, 毗邻多所名校, 周围遍布知名科技公司（如字节跳动、腾讯等）。
			- **户型**: 120平米三居室
			- **总价**: 约1500万人民币
			- **适合人群**: 科技公司高管、重视子女教育的家庭。
			
			**2. 国贸天际 (ID: B-02)**
			- **区域**: 朝阳区-国贸CBD
			- **特点**: 城市核心地标, 270度落地窗俯瞰CBD夜景, 奢华精装修，顶级商业配套。
			- **户型**: 280平米大平层
			- **总价**: 约3500万人民币
			- **适合人群**: 企业家、公司创始人(CEO/C-level)、金融精英、追求顶级生活品质人士。
			
			**3. 未来之城 (ID: C-03)**
			- **区域**: 通州区-城市副中心
			- **特点**: 新兴规划区域, 潜力巨大, 环境优美, 配套设施完善, 性价比高。
			- **户型**: 140平米四居室
			- **总价**: 约800万人民币
			- **适合人群**: 在国贸或副中心工作的白领、首次改善型购房家庭。
			
			**4. 文艺 loft (ID: D-04)**
			- **区域**: 朝阳区-798艺术区
			- **特点**: 设计师风格, 挑高5米, 充满艺术气息, 交通便利。
			- **户型**: 60平米复式Loft
			- **总价**: 约450万人民币
			- **适合人群**: 年轻单身贵族、设计师、创意工作者。
			
			### B. 购房建议规则
			
			1.  **预算评估**:
				- 房屋总价建议不超过家庭年收入的10倍。
				- 月供（按30年商业贷款，利率4%估算）不应超过家庭月收入的50%。
			2.  **职住平衡**: 推荐的房产区域应与用户公司所在地有较好的通勤关系。例如，在字节跳动工作的高管，优先推荐海淀区的“瀚海星辰”。
			3.  **身份匹配**: 房产的“适合人群”标签应与用户的职位和身份高度匹配。例如，CEO身份的用户应优先考虑“国贸天际”这类彰显身份的豪宅。
			`))
			res = append(res, input...)
			return res
		},
	})

	if err != nil {
		panic(err)
	}

	result, err := agent.Generate(ctx, []*schema.Message{
		schema.UserMessage("我叫 zhangsan, 邮箱是 zhangsan@bytedance.com, 帮我推荐一处房产，用中文回答问题"),
	})

	if err != nil {
		panic(err)
	}

	fmt.Println("Agent回答:", result.Content)
}
