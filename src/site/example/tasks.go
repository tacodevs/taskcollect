package example

import (
	"main/site"
	"time"
)

// TODO: populate subject variables with example tasks
func Tasks(user site.User, c chan site.Pair[[]site.Task, error], classes []site.Class) {
	var result site.Pair[[]site.Task, error]
	var tasks []site.Task
	bio := []site.Task{}
	chem := []site.Task{
		{
			Name:      "SHE task topic proposals",
			Class:     "Chemistry",
			Link:      "https://example.com",
			Desc:      "Please submit your SHE topic proposals",
			Due:       time.Date(2021, 2, 12, 23, 59, 59, 999999999, user.Timezone),
			Posted:    time.Date(2021, 1, 27, 21, 23, 55, 0, user.Timezone),
			ResLinks:  [][2]string{{"https://example.com", "Year 10 Chemistry - SHE task sheet"}, {"https://example.com", "SHE task past topics"}},
			Upload:    true,
			WorkLinks: [][2]string{{"https://example.com", "John SMITH - SHE task proposal"}},
			Submitted: true,
			Graded:    true,
			Grade:     "",
			Score:     0.0,
			Comment:   "",
			Platform:  "example",
			Id:        "576252975",
		},
	}
	english := []site.Task{
		{
			Name:      "The Visit: passage analysis",
			Class:     "English",
			Link:      "https://example.com",
			Desc:      "In this task you are looking at a passage from The Visit (pg. 84-85). Your goal is to unpack the role of Ill's family as secondary characters in this passage, and discuss the implications of this on other characters (especially Ill!)\n\nRemember: DEVICE drives MEANING!\n\nYou must first identify the literary DEVICES used by Dürrenmatt, then explain HOW Dürrenmatt uses these devices to SHOW something (while providing EVIDENCE in your explanations!)",
			Due:       time.Time{},
			Posted:    time.Date(2023, 1, 29, 16, 43, 23, 0, user.Timezone),
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
	history := []site.Task{}
	maths := []site.Task{}
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
