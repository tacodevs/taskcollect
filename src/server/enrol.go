package server

import (
	"main/site"
	"main/site/daymap"
	"main/site/example"
	"main/site/myadelaide"
	"main/site/saml"
)

func Enrol(institutes []string) {
	for _, institute := range institutes {
		switch institute {
		case "gihs":
			schools["gihs"] = site.NewMux()
			schools["gihs"].AddAuth(saml.Auth)
			schools["gihs"].AddAuth(daymap.Auth)
			schools["gihs"].AddClasses(daymap.Classes)
			schools["gihs"].AddGraded(daymap.Graded)
			schools["gihs"].SetLessons(daymap.Lessons)
			schools["gihs"].AddRemoveWork("daymap", daymap.RemoveWork)
			schools["gihs"].AddResource("daymap", daymap.Resource)
			schools["gihs"].AddResources("daymap", daymap.Resources)
			schools["gihs"].AddSubmit("daymap", daymap.Submit)
			schools["gihs"].AddTask("daymap", daymap.Task)
			schools["gihs"].AddTasks("daymap", daymap.Tasks)
			schools["gihs"].AddUploadWork("daymap", daymap.UploadWork)
		case "uofa":
			schools["uofa"] = site.NewMux()
			schools["uofa"].AddAuth(myadelaide.Auth)
			schools["uofa"].SetLessons(myadelaide.Lessons)
		case "example":
			schools["example"] = site.NewMux()
			schools["example"].AddAuth(example.Auth)
			schools["example"].AddClasses(example.Classes)
			schools["example"].AddGraded(example.Graded)
			schools["example"].SetLessons(example.Lessons)
			schools["example"].AddRemoveWork("example", example.RemoveWork)
			schools["example"].AddResource("example", example.Resource)
			schools["example"].AddResources("example", example.Resources)
			schools["example"].AddSubmit("example", example.Submit)
			schools["example"].AddTask("example", example.Task)
			schools["example"].AddTasks("example", example.Tasks)
			schools["example"].AddUploadWork("example", example.UploadWork)
		}
	}
}
