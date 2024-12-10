package example

import (
	"main/site"
	"time"
)

var core = []site.Resource{
	{
		Name:     "Knowledge and the knower",
		Class:    "Core",
		Link:     "https://example.com",
		Desc:     "",
		Posted:   time.Date(2022, 1, 30, 18, 37, 56, 0, time.UTC),
		ResLinks: [][2]string{{"https://example.com/1", "Why do we need knowledge?"}, {"https://example.com/2", "Knowledge as justified true belief"}},
		Platform: "example",
		Id:       "673096487",
	},
	{
		Name:     "Mini-exhibition examples",
		Class:    "Core",
		Link:     "https://example.com",
		Desc:     "Some examples to help guide you with your exhibition ideas.",
		Posted:   time.Date(2022, 3, 27, 8, 56, 13, 0, time.UTC),
		ResLinks: [][2]string{{"https://example.com/1", "Commentary 1"}, {"https://example.com/2", "Commentary 2"}, {"https://example.com/3", "Commentary 3"}},
		Platform: "example",
		Id:       "985712403",
	},
}

var french = []site.Resource{
	{
		Name:     "Reflexive verbs",
		Class:    "French",
		Link:     "https://example.com",
		Desc:     "",
		Posted:   time.Date(2021, 11, 3, 11, 0, 33, 0, time.UTC),
		ResLinks: [][2]string{{"https://example.com/1", "Reflexive verbs.docx"}},
		Platform: "example",
		Id:       "882467930",
	},
	{
		Name:     "When to use the subjunctive",
		Class:    "French",
		Link:     "https://example.com",
		Desc:     "",
		Posted:   time.Date(2021, 8, 1, 12, 33, 13, 0, time.UTC),
		ResLinks: [][2]string{{"https://example.com/1", "When to use the subjunctive.docx"}},
		Platform: "example",
		Id:       "976249195",
	},
}

var physics = []site.Resource{
	{
		Name:     "Hooke's law practical",
		Class:    "Physics",
		Link:     "https://example.com",
		Desc:     "Instructions for how to conduct Hooke's law practical with retort stand and 50 gram weights.",
		Posted:   time.Date(2021, 2, 1, 13, 25, 43, 0, time.UTC),
		ResLinks: [][2]string{{"https://example.com/1", "Hookes law practical.docx"}},
		Platform: "example",
		Id:       "781977340",
	},
	{
		Name:     "Linear motion",
		Class:    "Physics",
		Link:     "https://example.com",
		Desc:     "",
		Posted:   time.Date(2021, 2, 18, 14, 25, 19, 0, time.UTC),
		ResLinks: [][2]string{{"https://example.com/1", "Linear motion.pptx"}},
		Platform: "example",
		Id:       "876286709",
	},
}

func Resources(user site.User, c chan site.Pair[[]site.Resource, error], classes []site.Class) {
	var result site.Pair[[]site.Resource, error]
	var resources []site.Resource
	for _, class := range classes {
		switch class.Name {
		case "Core":
			resources = append(resources, core...)
		case "French":
			resources = append(resources, french...)
		case "Physics":
			resources = append(resources, physics...)
		}
	}
	// If an error occurs, set result.Second to err instead.
	result.First = resources
	c <- result
}
