package server

import (
	"main/site"
	"main/site/daymap"
	"main/site/example"
	"main/site/myadelaide"
	"main/site/saml"
)

func enrol(institutes ...string) {
	for _, institute := range institutes {
		switch institute {
		case "gihs":
			schools["gihs"] = site.NewMux()
			schools["gihs"].AddAuth(saml.Auth)
			schools["gihs"].AddAuth(daymap.Auth)
			//schools["gihs"].AddAuth(classroom.Auth)
			schools["gihs"].AddClasses(daymap.Classes)
			//schools["gihs"].AddClasses(classroom.Classes)
			//schools["gihs"].AddDueTasks(daymap.DueTasks)
			//schools["gihs"].AddDueTasks(classroom.DueTasks)
			//schools["gihs"].AddEvents(outlook.Events)
			schools["gihs"].AddGraded(daymap.Graded)
			//schools["gihs"].AddGraded(classroom.Graded)
			schools["gihs"].SetLessons(daymap.Lessons)
			//schools["gihs"].AddMessages(daymap.Messages)
			//schools["gihs"].SetReports(learnprof.Reports)
			//schools["gihs"].AddResources("daymap", daymap.Resources)
			//schools["gihs"].AddResources("classroom", classroom.Resources)
			schools["gihs"].AddTask("daymap", daymap.Task)
			schools["gihs"].AddTasks("daymap", daymap.Tasks)
			//schools["gihs"].AddTasks("classroom", classroom.Tasks)
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
			schools["example"].AddTasks("example", example.Tasks)
		}
	}
}
