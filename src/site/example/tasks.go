package example

import (
	"main/site"
	"time"
)

var bio = []site.Task{
	{
		Name:      "Genetic inheritance worksheet",
		Class:     "Biology",
		Link:      "https://example.com",
		Desc:      "",
		Due:       time.Time{},
		Posted:    time.Date(2021, 2, 6, 10, 21, 49, 0, time.UTC),
		ResLinks:  [][2]string{{"https://example.com/1", "Yr 10 Genetic Inheritance.docx"}},
		Upload:    true,
		WorkLinks: [][2]string{},
		Submitted: false,
		Graded:    false,
		Grade:     "",
		Score:     0.0,
		Comment:   "",
		Platform:  "example",
		Id:        "783663248",
	},
	{
		Name:      "Cell structure: in-class questions",
		Class:     "Biology",
		Link:      "https://example.com",
		Desc:      "",
		Due:       time.Time{},
		Posted:    time.Date(2021, 1, 29, 17, 20, 1, 0, time.UTC),
		ResLinks:  [][2]string{},
		Upload:    true,
		WorkLinks: [][2]string{},
		Submitted: true,
		Graded:    false,
		Grade:     "",
		Score:     0.0,
		Comment:   "",
		Platform:  "example",
		Id:        "873468673",
	},
}

var chem = []site.Task{
	{
		Name:      "Organic chemistry practice questions",
		Class:     "Chemistry",
		Link:      "https://example.com",
		Desc:      "",
		Due:       time.Date(2022, 3, 1, 23, 59, 59, 999999999, time.UTC),
		Posted:    time.Date(2022, 1, 29, 17, 20, 1, 0, time.UTC),
		ResLinks:  [][2]string{{"https://example.com/1", "yr 11 Organic chemistry practice -1.docx"}},
		Upload:    true,
		WorkLinks: [][2]string{},
		Submitted: false,
		Graded:    false,
		Grade:     "",
		Score:     0.0,
		Comment:   "",
		Platform:  "example",
		Id:        "725987605",
	},
	{
		Name:      "SHE task topic proposals",
		Class:     "Chemistry",
		Link:      "https://example.com",
		Desc:      "Please submit your SHE topic proposals",
		Due:       time.Date(2022, 2, 12, 23, 59, 59, 999999999, time.UTC),
		Posted:    time.Date(2022, 1, 27, 21, 23, 55, 0, time.UTC),
		ResLinks:  [][2]string{{"https://example.com/1", "Year 11 Chemistry - SHE task sheet"}, {"https://example.com/2", "SHE task past topics"}},
		Upload:    true,
		WorkLinks: [][2]string{{"https://example.com/3", "John SMITH - SHE task proposal"}},
		Submitted: true,
		Graded:    true,
		Grade:     "",
		Score:     0.0,
		Comment:   "",
		Platform:  "example",
		Id:        "576252975",
	},
}

var english = []site.Task{
	{
		Name:      "The Visit: passage analysis",
		Class:     "English",
		Link:      "https://example.com",
		Desc:      "In this task you are looking at a passage from The Visit (pg. 84-85). Your goal is to unpack the role of Ill's family as secondary characters in this passage, and discuss the implications of this on other characters (especially Ill!)\n\nRemember: DEVICE drives MEANING!\n\nYou must first identify the literary DEVICES used by Dürrenmatt, then explain HOW Dürrenmatt uses these devices to SHOW something (while providing EVIDENCE in your explanations!)",
		Due:       time.Time{},
		Posted:    time.Date(2023, 1, 29, 16, 43, 23, 0, time.UTC),
		ResLinks:  [][2]string{},
		Upload:    true,
		WorkLinks: [][2]string{},
		Submitted: true,
		Graded:    true,
		Grade:     "5",
		Score:     73.3333,
		Comment:   "(Formative: in class on paper)\n\nThe main ideas behind the points you make are good and are supported by evidence, but the delivery of your analysis was not as coherent and clear as it could have been. The logical flow is occasionally disrupted by slight deviations, which you should try to avoid in literary analysis.",
		Platform:  "example",
		Id:        "756438139",
	},
}

var history = []site.Task{
	{
		Name:      "Article analysis: examining different perspectives",
		Class:     "History",
		Link:      "https://example.com",
		Desc:      "",
		Due:       time.Time{},
		Posted:    time.Date(2020, 2, 13, 14, 10, 15, 0, time.UTC),
		ResLinks:  [][2]string{{"https://example.com/1", "Article Analysis Task"}, {"https://example.com/2", "Article 1.docx"}, {"https://example.com/3", "Article 2.pdf"}},
		Upload:    true,
		WorkLinks: [][2]string{{"https://example.com/4", "John SMITH - Article Analysis Task"}},
		Submitted: true,
		Graded:    false,
		Grade:     "",
		Score:     0.0,
		Comment:   "",
		Platform:  "example",
		Id:        "723671061",
	},
	{
		Name:      "Primary and secondary sources",
		Class:     "History",
		Link:      "https://example.com",
		Desc:      "",
		Due:       time.Time{},
		Posted:    time.Date(2020, 2, 13, 14, 8, 36, 0, time.UTC),
		ResLinks:  [][2]string{{"https://example.com/1", "How to use primary and secondary sources.docx"}},
		Upload:    true,
		WorkLinks: [][2]string{},
		Submitted: false,
		Graded:    false,
		Grade:     "",
		Score:     0.0,
		Comment:   "",
		Platform:  "example",
		Id:        "547394651",
	},
}

var maths = []site.Task{
	{
		Name:      "Investigation draft",
		Class:     "Mathematics",
		Link:      "https://example.com",
		Desc:      "",
		Due:       time.Date(time.Now().Year()+1, 2, 9, 17, 0, 0, 0, time.UTC),
		Posted:    time.Date(time.Now().Year(), 2, 9, 21, 12, 34, 0, time.UTC),
		ResLinks:  [][2]string{{"https://example.com/1", "HL Maths Mini-IA"}},
		Upload:    true,
		WorkLinks: [][2]string{},
		Submitted: false,
		Graded:    false,
		Grade:     "",
		Score:     0.0,
		Comment:   "",
		Platform:  "example",
		Id:        "125726502",
	},
	{
		Name:      "Integration by parts exercise",
		Class:     "Mathematics",
		Link:      "https://example.com",
		Desc:      "",
		Due:       time.Time{},
		Posted:    time.Date(time.Now().Year(), 1, 29, 16, 57, 23, 0, time.UTC),
		ResLinks:  [][2]string{{"https://example.com/1", "Integration by parts questions.pdf"}},
		Upload:    true,
		WorkLinks: [][2]string{},
		Submitted: false,
		Graded:    false,
		Grade:     "",
		Score:     0.0,
		Comment:   "",
		Platform:  "example",
		Id:        "196728422",
	},
}

func Tasks(user site.User, c chan site.Pair[[]site.Task, error], classes []site.Class) {
	var result site.Pair[[]site.Task, error]
	var tasks []site.Task
	for _, class := range classes {
		switch class.Name {
		case "Biology":
			tasks = append(tasks, bio...)
		case "Chemistry":
			tasks = append(tasks, chem...)
		case "English":
			tasks = append(tasks, english...)
		case "History":
			tasks = append(tasks, history...)
		case "Mathematics":
			tasks = append(tasks, maths...)
		}
	}
	// If an error occurs, set result.Second to err instead.
	result.First = tasks
	c <- result
}
