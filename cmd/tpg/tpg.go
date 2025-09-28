package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	mathRand "math/rand"
	"os"
	"strings"
	"time"

	"github.com/yinxulai/ait/internal/display"
)

var (
	// 使用全局随机数生成器
	rng = mathRand.New(mathRand.NewSource(time.Now().UnixNano()))

	sentences = []string{
		"The quick brown fox jumps over the lazy dog",
		"Pack my box with five dozen liquor jugs",
		"Generate a creative story about artificial intelligence",
		"Explain quantum computing in simple terms",
		"Write a poem about the changing seasons",
		"Analyze the impact of social media on modern society",
		"Describe your ideal futuristic cityscape",
		"Compare and contrast machine learning algorithms",
		"Provide a detailed recipe for chocolate chip cookies",
		"Discuss the philosophical implications of consciousness",
		"Create a business plan for a tech startup",
		"Outline the key elements of effective leadership",
		"Describe how blockchain technology works",
		"Write a short script for a science fiction movie",
		"Explain the theory of relativity using analogies",
		"Discuss the pros and cons of renewable energy",
		"Generate ideas for reducing carbon footprint",
		"Analyze the theme of love in classical literature",
		"Create a marketing strategy for a new product",
		"Explain the water cycle to a 5-year-old",
		"Describe the process of cellular respiration",
		"Write a persuasive essay about education reform",
		"Discuss the cultural significance of ancient mythology",
		"Propose a solution to traffic congestion in cities",
		"Explain how vaccines work to fight diseases",
		// 中文句子
		"请解释人工智能的基本原理和应用场景",
		"描述一下你理想中的智能城市是什么样子",
		"分析社交媒体对现代社会的积极和消极影响",
		"写一首关于四季变化的现代诗",
		"解释区块链技术的工作原理和优势",
		"设计一个可持续发展的商业模式",
		"讨论教育改革的必要性和实施策略",
		"分析古典文学中爱情主题的表现手法",
		"提出解决城市交通拥堵的创新方案",
		"解释疫苗如何帮助人体抵抗疾病",
		// 日文句子
		"人工知能の基本的な仕組みと応用について説明してください",
		"理想的な未来都市の姿を描写してください",
		"ソーシャルメディアが現代社会に与える影響を分析してください",
		"四季の移り変わりについて詩を書いてください",
		"ブロックチェーン技術の仕組みと利点を説明してください",
		"持続可能なビジネスモデルを設計してください",
		"教育改革の必要性と実施戦略について論じてください",
		"古典文学における愛のテーマの表現手法を分析してください",
		"都市の交通渋滞を解決する革新的な方案を提案してください",
		"ワクチンが病気と闘うメカニズムを説明してください",
		// 韩文句子
		"인공지능의 기본 원리와 응용 분야에 대해 설명해주세요",
		"당신이 생각하는 이상적인 미래 도시의 모습을 묘사해주세요",
		"소셜미디어가 현대사회에 미치는 긍정적, 부정적 영향을 분석해주세요",
		"계절의 변화에 대한 시를 써주세요",
		"블록체인 기술의 작동 원리와 장점을 설명해주세요",
		"지속가능한 비즈니스 모델을 설계해주세요",
		"교육 개혁의 필요성과 실행 전략에 대해 논의해주세요",
		"고전문학에서 사랑 주제의 표현 기법을 분석해주세요",
		"도시 교통 체증을 해결할 혁신적인 방안을 제안해주세요",
		"백신이 질병과 싸우는 메커니즘을 설명해주세요",
		// 法文句子
		"Expliquez les principes fondamentaux de l'intelligence artificielle",
		"Décrivez votre vision de la ville intelligente idéale",
		"Analysez l'impact des réseaux sociaux sur la société moderne",
		"Écrivez un poème sur les changements de saisons",
		"Expliquez le fonctionnement de la technologie blockchain",
		"Concevez un modèle commercial durable",
		"Discutez de la nécessité de réformer l'éducation",
		"Analysez le thème de l'amour dans la littérature classique",
		"Proposez des solutions innovantes aux embouteillages urbains",
		"Expliquez comment les vaccins combattent les maladies",
		// 德文句子
		"Erklären Sie die Grundprinzipien der künstlichen Intelligenz",
		"Beschreiben Sie Ihre Vision einer idealen Smart City",
		"Analysieren Sie die Auswirkungen sozialer Medien auf die Gesellschaft",
		"Schreiben Sie ein Gedicht über den Wechsel der Jahreszeiten",
		"Erklären Sie die Funktionsweise der Blockchain-Technologie",
		"Entwickeln Sie ein nachhaltiges Geschäftsmodell",
		"Diskutieren Sie die Notwendigkeit von Bildungsreformen",
		"Analysieren Sie das Liebesthema in der klassischen Literatur",
		"Schlagen Sie innovative Lösungen für städtische Verkehrsstaus vor",
		"Erklären Sie, wie Impfstoffe Krankheiten bekämpfen",
		// 西班牙文句子
		"Explique los principios básicos de la inteligencia artificial",
		"Describa su visión de la ciudad inteligente ideal",
		"Analice el impacto de las redes sociales en la sociedad moderna",
		"Escriba un poema sobre los cambios estacionales",
		"Explique cómo funciona la tecnología blockchain",
		"Diseñe un modelo de negocio sostenible",
		"Discuta la necesidad de reformas educativas",
		"Analice el tema del amor en la literatura clásica",
		"Proponga soluciones innovadoras para la congestión del tráfico urbano",
		"Explique cómo las vacunas combaten las enfermedades",
		// 俄文句子
		"Объясните основные принципы искусственного интеллекта",
		"Опишите вашу концепцию идеального умного города",
		"Проанализируйте влияние социальных сетей на современное общество",
		"Напишите стихотворение о смене времен года",
		"Объясните принципы работы технологии блокчейн",
		"Разработайте устойчивую бизнес-модель",
		"Обсудите необходимость реформы образования",
		"Проанализируйте тему любви в классической литературе",
		"Предложите инновационные решения городских пробок",
		"Объясните, как вакцины борются с болезнями",
		// 阿拉伯文句子
		"اشرح المبادئ الأساسية للذكاء الاصطناعي",
		"صف رؤيتك للمدينة الذكية المثالية",
		"حلل تأثير وسائل التواصل الاجتماعي على المجتمع الحديث",
		"اكتب قصيدة عن تغير الفصول",
		"اشرح كيفية عمل تقنية البلوك تشين",
		"صمم نموذج أعمال مستدام",
		"ناقش ضرورة إصلاح التعليم",
		"حلل موضوع الحب في الأدب الكلاسيكي",
		"اقترح حلول مبتكرة لازدحام المرور في المدن",
		"اشرح كيف تحارب اللقاحات الأمراض",
	}
)

// Template 模板结构
type Template struct {
	Content string
	Variables map[string]string
}

// applyTemplate 应用模板，替换占位符
func (t *Template) applyTemplate(content string, index int, timestamp time.Time) string {
	result := t.Content
	
	// 替换基本占位符
	result = strings.ReplaceAll(result, "{{content}}", content)
	result = strings.ReplaceAll(result, "{{index}}", fmt.Sprintf("%d", index))
	result = strings.ReplaceAll(result, "{{timestamp}}", timestamp.Format(time.RFC3339))
	
	// 替换自定义变量
	for key, value := range t.Variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	
	return result
}

// generateTaskID 生成任务ID（参考 ait.go）
func generateTaskID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)

	// 设置版本 (4) 和变体位
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

// validateParams 验证参数
func validateParams(count, length int, outputDir string) error {
	if count <= 0 {
		return fmt.Errorf("count 必须大于 0")
	}

	if length <= 0 {
		return fmt.Errorf("length 必须大于 0")
	}

	if outputDir == "" {
		return fmt.Errorf("输出目录不能为空")
	}

	return nil
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println("TPG - 测试 Prompt 生成器")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  tpg [选项]")
	fmt.Println("")
	fmt.Println("选项:")
	fmt.Println("  -count        生成的 prompt 数量 (默认: 10)")
	fmt.Println("  -length       每个 prompt 的近似长度 (默认: 50)")
	fmt.Println("  -output       输出目录 (默认: output)")
	fmt.Println("  -template     模板字符串，支持占位符: {{content}}, {{index}}, {{timestamp}}")
	fmt.Println("  -help         显示此帮助信息")
	fmt.Println("")
	fmt.Println("模板占位符说明:")
	fmt.Println("  {{content}}    - 生成的prompt内容")
	fmt.Println("  {{index}}      - prompt序号 (从1开始)")
	fmt.Println("  {{timestamp}}  - 当前时间戳 (RFC3339格式)")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  tpg -count=20 -length=100")
	fmt.Println("  tpg -output=./test-prompts")
	fmt.Println("  tpg -template=\"请回答以下问题: {{content}}\"")
}

// generateRandomText 生成指定长度的随机文本
func generateRandomText(desiredLength int) string {
	var result strings.Builder
	var selectedSentences []string
	
	// 选择句子直到累计长度达到或接近期望长度
	totalLength := 0
	for totalLength < desiredLength {
		sentence := sentences[rng.Intn(len(sentences))]
		// 如果加上这个句子会超出期望长度太多，且已经有句子了，就停止
		if totalLength > 0 && totalLength + len(sentence) + 1 > desiredLength * 2 {
			break
		}
		selectedSentences = append(selectedSentences, sentence)
		totalLength += len(sentence) + 1 // +1 for space
	}
	
	// 拼接句子
	for i, sentence := range selectedSentences {
		result.WriteString(sentence)
		if i < len(selectedSentences) - 1 {
			result.WriteString(" ")
		}
	}
	
	return result.String()
}

// writePromptFile 写入 prompt 文件
func writePromptFile(prompt, filename string, template *Template, index int) error {
	// 如果有模板，应用模板
	finalContent := prompt
	if template != nil {
		finalContent = template.applyTemplate(prompt, index, time.Now())
	}

	if err := os.WriteFile(filename, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("写入文件 %s 失败: %v", filename, err)
	}
	return nil
}

func main() {
	// 定义命令行参数
	count := flag.Int("count", 10, "生成的 prompt 数量")
	length := flag.Int("length", 50, "每个 prompt 的近似长度")
	outputDir := flag.String("output", "prompts", "输出目录")
	templateStr := flag.String("template", "", "模板字符串，支持占位符")
	help := flag.Bool("help", false, "显示帮助信息")

	flag.Parse()

	// 显示帮助信息
	if *help {
		printHelp()
		return
	}

	// 验证参数
	if err := validateParams(*count, *length, *outputDir); err != nil {
		fmt.Printf("%s错误: %s%s\n", display.ColorRed, err.Error(), display.ColorReset)
		fmt.Println("使用 -help 查看帮助信息")
		os.Exit(1)
	}
	
	// 处理模板
	var template *Template
	
	if *templateStr != "" {
		template = &Template{
			Content: *templateStr,
			Variables: make(map[string]string),
		}
	}

	// 创建输出目录
	if err := os.MkdirAll(*outputDir, os.ModePerm); err != nil {
		fmt.Printf("%s错误: 创建输出目录失败: %v%s\n", display.ColorRed, err, display.ColorReset)
		os.Exit(1)
	}

	// 生成任务ID（可用于日志等）
	taskID := generateTaskID()
	_ = taskID // 当前版本暂不使用，但保留接口

	// 显示欢迎信息和配置
	displayer := display.New()
	displayer.ShowWelcome()

	fmt.Printf("%s=== TPG 配置信息 ===%s\n", display.ColorBlue, display.ColorReset)
	fmt.Printf("数量: %d\n", *count)
	fmt.Printf("长度: %d 字符\n", *length)
	fmt.Printf("输出目录: %s\n", *outputDir)
	if template != nil {
		fmt.Printf("使用模板: 是\n")
	} else {
		fmt.Printf("使用模板: 否\n")
	}
	fmt.Println()

	// 生成 prompts
	successCount := 0
	var errors []string

	for i := 0; i < *count; i++ {
		prompt := generateRandomText(*length)
		filename := fmt.Sprintf("%s/prompt_%d.txt", *outputDir, i+1)

		if err := writePromptFile(prompt, filename, template, i+1); err != nil {
			errors = append(errors, err.Error())
			continue
		}

		successCount++
		fmt.Printf("%s✓%s 生成文件: %s\n", display.ColorGreen, display.ColorReset, filename)
	}

	// 显示结果统计
	fmt.Println()
	fmt.Printf("%s=== 生成结果 ===%s\n", display.ColorBlue, display.ColorReset)
	fmt.Printf("%s成功生成: %d/%d%s\n", display.ColorGreen, successCount, *count, display.ColorReset)

	if len(errors) > 0 {
		fmt.Printf("%s失败: %d%s\n", display.ColorRed, len(errors), display.ColorReset)
		fmt.Printf("%s错误详情:%s\n", display.ColorRed, display.ColorReset)
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}
}
