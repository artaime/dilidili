package story

import (
	"math/rand"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	diversityRecentLimit = 10
	diversityRecentDays  = 7
	diversityMaxAvoid    = 16
)

// DiversitySeed 开放/原创生成时注入的多样性约束。
type DiversitySeed struct {
	Genre           string   // 强制题材（空表示不强制）
	SubjectHint     string   // 题材切入点
	ProtagonistHint string   // 主角名建议
	AvoidNames      []string // 近期人物名，禁止复用
	AvoidGenres     []string // 近期题材
	AvoidThemes     []string // 近期主题
}

// NeedsGenreDiversity 用户未点名具体故事/类型时，需要系统指定题材轮换。
func NeedsGenreDiversity(params StoryParams) bool {
	p := params
	NormalizeStoryParams(&p)
	if ShouldTellCanonical(p) {
		return false
	}
	theme := NormalizeThemeKey(p.Theme)
	open := p.UserSaidCasual || theme == "" || isGenericThemeKey(theme)
	if !open {
		return false
	}
	// 已点名神话/寓言/童话等类型时题材已定，只做人物名多样性
	switch p.RequestType {
	case StoryModeMyth, StoryModeFable, StoryModeFairy, StoryModeClassic:
		return false
	default:
		return true
	}
}

// NeedsNameDiversity 原创故事均需人物名新鲜度约束。
func NeedsNameDiversity(params StoryParams) bool {
	p := params
	NormalizeStoryParams(&p)
	return !ShouldTellCanonical(p)
}

// CollectRecentAvoidance 从近期记录收集题材/主题/人物名回避信息。
func CollectRecentAvoidance(recent []StoryRecord) (genres, themes, names []string) {
	genreSet := map[string]struct{}{}
	themeSet := map[string]struct{}{}
	nameSet := map[string]struct{}{}

	addUnique := func(set map[string]struct{}, list *[]string, v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := set[v]; ok {
			return
		}
		if len(*list) >= diversityMaxAvoid {
			return
		}
		set[v] = struct{}{}
		*list = append(*list, v)
	}

	for _, rec := range recent {
		if rec.ParamsSnapshot != nil {
			if g, ok := rec.ParamsSnapshot["genre"].(string); ok {
				addUnique(genreSet, &genres, NormalizeGenre(g))
			}
			if t, ok := rec.ParamsSnapshot["theme"].(string); ok {
				addUnique(themeSet, &themes, NormalizeThemeKey(t))
			}
			if t, ok := rec.ParamsSnapshot["story_title"].(string); ok {
				addUnique(themeSet, &themes, NormalizeThemeKey(t))
			}
			switch c := rec.ParamsSnapshot["characters"].(type) {
			case []string:
				for _, n := range c {
					addUnique(nameSet, &names, n)
				}
			case []any:
				for _, item := range c {
					if s, ok := item.(string); ok {
						addUnique(nameSet, &names, s)
					}
				}
			case string:
				for _, n := range strings.Split(c, ",") {
					addUnique(nameSet, &names, n)
				}
			}
		}
		if title := strings.TrimSpace(rec.Title); title != "" {
			addUnique(themeSet, &themes, NormalizeThemeKey(title))
		}
		for _, n := range ExtractCharacterNames(rec.FullText, 8) {
			addUnique(nameSet, &names, n)
		}
	}
	return genres, themes, names
}

var (
	// 叫XX / 名叫XX（仅汉字，随后在 clean 中截断「的」等）
	nameCallPattern = regexp.MustCompile(`(?:叫|名叫|名字叫)\s*([\p{Han}]{1,4})`)
	// 小X（常见儿童故事昵称）
	xiaoNamePattern = regexp.MustCompile(`小[\p{Han}]{1,2}`)
)

// overusedDefaultNames LLM 高频默认名，始终提示回避。
var overusedDefaultNames = []string{
	"小明", "小红", "小丽", "小强", "小华", "小刚", "小美",
	"小兔子", "小熊", "小狐狸", "豆豆", "乐乐", "圆圆", "明明",
}

// genreSubjectHints 各题材切入点池（开放生成随机选用）。
var genreSubjectHints = map[string][]string{
	"童话": {
		"会发光的蘑菇灯", "迷路的风筝", "会说话的雨靴", "月亮上的缝纫机",
		"把星星缝回天空的裁缝", "不肯开花的盆栽", "夜里出门的图书馆",
	},
	"历史": {
		"古镇上发现旧地图的少年", "学做瓷器的小小工匠", "跟着商队学写字的孩子",
		"运河边送信的小姑娘", "书院里偷偷画画的书童",
	},
	"神话": {
		"向风神借一把蒲扇", "守护井口的小龙", "追着彩虹找露珠的少年",
		"月亮里修桥的石匠", "把雷声藏进鼓里的孩子",
	},
	"寓言": {
		"两只争镜子的喜鹊", "不愿分享影子的猫", "只会模仿的鹦鹉学不会问为什么",
		"把诚实装进口袋的狐狸", "比谁跑得慢的蜗牛比赛",
	},
	"冒险": {
		"地下图书馆的迷路灯笼", "云层里的迷你火车站", "潮汐退去后的秘密沙滩",
		"地图上多出来的小岛", "跟着萤火虫穿越竹林",
	},
	"侦探": {
		"教室里失踪的粉笔", "谁偷走了午后的阳光", "花园脚印之谜",
		"会自己移动的棋子", "图书馆书架上的暗号书签",
	},
	"科幻": {
		"会收集梦的小小机器人", "雨天失灵的天气预报员", "月球邮局的实习送信员",
		"把时间调慢十分钟的手表", "会翻译鸟语的耳机",
	},
	"生活": {
		"第一次独自坐公交", "和爷爷学包饺子", "班级义卖上的旧玩具",
		"搬家那天认识的新邻居", "雨天把伞借给同学",
	},
}

// protagonistNamePool 主角名种子池（覆盖人/动物/物件拟人，避免刻板重复）。
var protagonistNamePool = []string{
	"阿澄", "林夏", "江禾", "宋晚晚", "顾安安", "沈星河", "叶知秋", "陆小小",
	"青禾", "南风", "白露", "桃桃", "杏儿", "米粒", "云朵", "石子",
	"阿暖", "小橙", "阿屿", "北北", "桐桐", "苏苏", "言言", "乔乔",
	"点点灯", "蒲公英", "回形针", "蓝气球", "木马", "风铃", "糖纸", "邮票",
	"灰狸", "雪貂", "青雀", "斑鸠", "刺猬阿刺", "螃蟹横横", "海豚波波", "猫头鹰夜夜",
}

// PickDiversitySeed 根据参数与近期记录生成多样性种子；rng 为空则用时间播种。
func PickDiversitySeed(params StoryParams, recent []StoryRecord, rng *rand.Rand) *DiversitySeed {
	if !NeedsNameDiversity(params) && !NeedsGenreDiversity(params) {
		return nil
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	avoidGenres, avoidThemes, avoidNames := CollectRecentAvoidance(recent)
	for _, n := range overusedDefaultNames {
		if !containsFold(avoidNames, n) && len(avoidNames) < diversityMaxAvoid {
			avoidNames = append(avoidNames, n)
		}
	}

	seed := &DiversitySeed{
		AvoidNames:  avoidNames,
		AvoidGenres: avoidGenres,
		AvoidThemes: avoidThemes,
	}

	if NeedsGenreDiversity(params) {
		genre := pickGenre(params, avoidGenres, rng)
		seed.Genre = genre
		seed.SubjectHint = pickSubjectHint(genre, avoidThemes, rng)
	}

	if NeedsNameDiversity(params) {
		seed.ProtagonistHint = pickProtagonistName(avoidNames, rng)
	}
	return seed
}

func pickGenre(params StoryParams, avoid []string, rng *rand.Rand) string {
	// 用户 style 已点名题材时优先尊重（如「侦探」「科幻」）
	if style := strings.TrimSpace(params.Style); style != "" {
		if g := NormalizeGenre(style); g != "" && containsString(StoryGenres, g) {
			return g
		}
	}

	candidates := append([]string(nil), StoryGenres...)
	bedtime := params.IsBedtime != nil && *params.IsBedtime
	if bedtime {
		candidates = []string{"生活", "童话", "寓言"}
	}

	var preferred []string
	for _, g := range candidates {
		if !containsString(avoid, g) {
			preferred = append(preferred, g)
		}
	}
	if len(preferred) == 0 {
		preferred = candidates
	}
	return preferred[rng.Intn(len(preferred))]
}

func pickSubjectHint(genre string, avoidThemes []string, rng *rand.Rand) string {
	pool := genreSubjectHints[genre]
	if len(pool) == 0 {
		pool = genreSubjectHints["生活"]
	}
	var preferred []string
	for _, s := range pool {
		key := NormalizeThemeKey(s)
		if containsFold(avoidThemes, key) {
			continue
		}
		preferred = append(preferred, s)
	}
	if len(preferred) == 0 {
		preferred = pool
	}
	if len(preferred) == 0 {
		return ""
	}
	return preferred[rng.Intn(len(preferred))]
}

func pickProtagonistName(avoid []string, rng *rand.Rand) string {
	var preferred []string
	for _, n := range protagonistNamePool {
		if containsFold(avoid, n) {
			continue
		}
		preferred = append(preferred, n)
	}
	if len(preferred) == 0 {
		preferred = protagonistNamePool
	}
	return preferred[rng.Intn(len(preferred))]
}

// ExtractCharacterNames 从正文启发式抽取可能的人物称呼。
func ExtractCharacterNames(text string, limit int) []string {
	text = strings.TrimSpace(text)
	if text == "" || limit <= 0 {
		return nil
	}
	// 只扫前段，降低噪声
	runes := []rune(text)
	if len(runes) > 400 {
		text = string(runes[:400])
	}

	seen := map[string]struct{}{}
	var out []string
	add := func(n string) {
		n = cleanExtractedName(n)
		if !isPlausibleName(n) {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}

	for _, m := range nameCallPattern.FindAllStringSubmatch(text, -1) {
		if len(m) >= 2 {
			add(m[1])
			if len(out) >= limit {
				return out
			}
		}
	}
	for _, m := range xiaoNamePattern.FindAllString(text, -1) {
		add(m)
		if len(out) >= limit {
			return out
		}
	}
	return out
}

func cleanExtractedName(n string) string {
	n = strings.TrimSpace(n)
	n = strings.Trim(n, "「」『』\"'“”")
	if i := strings.Index(n, "的"); i > 0 {
		n = n[:i]
	}
	for _, suf := range []string{"了", "着", "过", "呢", "吧", "啊", "也", "和", "与", "在", "就", "还", "都", "来", "去"} {
		if strings.HasSuffix(n, suf) && len([]rune(n)) > len([]rune(suf))+1 {
			n = strings.TrimSuffix(n, suf)
		}
	}
	return strings.TrimSpace(n)
}

func isPlausibleName(n string) bool {
	n = strings.TrimSpace(n)
	if n == "" {
		return false
	}
	r := []rune(n)
	if len(r) < 1 || len(r) > 4 {
		return false
	}
	// 过滤明显非人名片段
	ban := []string{"什么", "他们", "她们", "我们", "自己", "时候", "地方", "故事", "孩子", "朋友"}
	for _, b := range ban {
		if n == b {
			return false
		}
	}
	han := 0
	for _, c := range r {
		if unicode.Is(unicode.Han, c) {
			han++
		}
	}
	return han >= 1
}

func containsString(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func containsFold(list []string, v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	for _, s := range list {
		if strings.EqualFold(strings.TrimSpace(s), v) {
			return true
		}
	}
	return false
}

// FormatDiversityPromptLines 生成写入 user prompt 的约束行。
func FormatDiversityPromptLines(seed *DiversitySeed) []string {
	if seed == nil {
		return nil
	}
	var parts []string
	if seed.Genre != "" {
		parts = append(parts, "本次指定题材："+seed.Genre+"（meta.genre 必须与此一致）")
	}
	if seed.SubjectHint != "" {
		parts = append(parts, "题材切入点："+seed.SubjectHint+"（围绕此切入点原创情节，标题可重新拟定）")
	}
	if seed.ProtagonistHint != "" {
		parts = append(parts, "主角名必须使用：「"+seed.ProtagonistHint+"」（勿改成常见默认名）")
	}
	if len(seed.AvoidNames) > 0 {
		parts = append(parts, "禁止使用的人物名（含近期故事）："+strings.Join(seed.AvoidNames, "、"))
	}
	if len(seed.AvoidThemes) > 0 {
		parts = append(parts, "禁止复用近期主题/标题："+strings.Join(clipList(seed.AvoidThemes, 8), "、"))
	}
	if len(seed.AvoidGenres) > 0 && seed.Genre == "" {
		parts = append(parts, "近期已讲过题材尽量避开："+strings.Join(seed.AvoidGenres, "、"))
	}
	return parts
}

func clipList(list []string, n int) []string {
	if n <= 0 || len(list) <= n {
		return list
	}
	return list[:n]
}
