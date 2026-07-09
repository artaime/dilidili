package story

type AgeBandSpec struct {
	ID          string
	MinAge      int
	MaxAge      int
	MinWords    int
	MaxWords    int
	Description string
}

var ageBandSpecs = []AgeBandSpec{
	{ID: "preschool", MinAge: 3, MaxAge: 6, MinWords: 300, MaxWords: 800, Description: "学龄前：句子短、节奏重复、人物≤3"},
	{ID: "primary_low", MinAge: 7, MaxAge: 9, MinWords: 600, MaxWords: 1500, Description: "小学低年级：完整故事、简单悬念与幽默"},
	{ID: "primary_high", MinAge: 10, MaxAge: 12, MinWords: 1200, MaxWords: 3000, Description: "小学高年级：多角色、成长与推理"},
	{ID: "junior_high", MinAge: 13, MaxAge: 15, MinWords: 1500, MaxWords: 5000, Description: "初中：可有多线剧情与心理成长，禁止成人恋爱与黑暗文学"},
}

func AgeBandFromYears(years int) string {
	for _, spec := range ageBandSpecs {
		if years >= spec.MinAge && years <= spec.MaxAge {
			return spec.ID
		}
	}
	if years < 3 {
		return "preschool"
	}
	if years > 15 {
		return "junior_high"
	}
	return "primary_low"
}

func GetAgeBandSpec(band string) AgeBandSpec {
	for _, spec := range ageBandSpecs {
		if spec.ID == band {
			return spec
		}
	}
	return ageBandSpecs[1]
}

func WordRangeForBand(band string) (int, int) {
	spec := GetAgeBandSpec(band)
	return spec.MinWords, spec.MaxWords
}
