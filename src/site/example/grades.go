package example

import (
	"time"

	"main/site"
)

func Graded(user site.User, c chan site.Pair[[]site.Task, error]) {
	var result site.Pair[[]site.Task, error]
	// The grades tab does not use the following fields:
	//   - Desc
	//   - Due
	//   - ResLinks
	//   - Upload
	//   - WorkLinks
	//   - Submitted
	//   - Comment
	// Needless to say, you should not do extra work to retrieve them.
	graded := []site.Task{
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
	// If an error occurs, set result.Second to err instead.
	result.First = graded
	c <- result
}
