package server

import (
	"fmt"
	"html"
	"html/template"
	"image/color"
	"strings"
	"time"

	"main/errors"
	"main/plat"
)

var gradeColors = []color.RGBA{
	{0xc9, 0x16, 0x14, 0xff}, // Red, #c91614
	{0xd9, 0x6b, 0x0a, 0xff}, // Amber/Orange, #d96b0a
	{0xf6, 0xde, 0x0a, 0xff}, // Yellow, #f6de0a
	{0x03, 0x6e, 0x05, 0xff}, // Green, #036e05
}

func genDueStr(due time.Time, creds User) string {
	var dueDate string
	now := time.Now().In(creds.Timezone)
	localDueDate := due.In(creds.Timezone)

	todayStart := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		creds.Timezone,
	)

	todayEnd := todayStart.AddDate(0, 0, 1)
	tmrEnd := todayStart.AddDate(0, 0, 2)
	yesterday := todayStart.AddDate(0, 0, -1)
	lastWeek := todayStart.AddDate(0, 0, -7)
	weekEnd := todayStart.AddDate(0, 0, 7)

	if localDueDate.After(weekEnd) || localDueDate.Before(lastWeek) {
		dueDate = localDueDate.Format("2 Jan 2006")
	} else if localDueDate.Before(weekEnd) && localDueDate.After(tmrEnd) {
		dueDate = localDueDate.Weekday().String()
	} else if localDueDate.After(lastWeek) && localDueDate.Before(yesterday) {
		dueDate = "last " + localDueDate.Weekday().String()
	} else if localDueDate.After(yesterday) && localDueDate.Before(todayStart) {
		dueDate = "yesterday"
	} else if localDueDate.Before(todayEnd) {
		dueDate = "today"
	} else if localDueDate.Before(tmrEnd) {
		dueDate = "tomorrow"
	}

	if localDueDate.Hour() != 0 || localDueDate.Minute() != 0 {
		dueDate += localDueDate.Format(", 15:04")
	}

	return dueDate
}

func genPostStr(posted time.Time, creds User) string {
	var postDate string
	now := time.Now().In(creds.Timezone)
	localPostDate := posted.In(creds.Timezone)

	todayStart := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		creds.Timezone,
	)

	todayEnd := todayStart.AddDate(0, 0, 1)
	tmrEnd := todayStart.AddDate(0, 0, 2)
	yesterday := todayStart.AddDate(0, 0, -1)
	lastWeek := todayStart.AddDate(0, 0, -7)
	weekEnd := todayStart.AddDate(0, 0, 7)

	if localPostDate.After(weekEnd) || localPostDate.Before(lastWeek) {
		postDate = localPostDate.Format("2 Jan 2006")
	} else if localPostDate.Before(weekEnd) && localPostDate.After(tmrEnd) {
		postDate = "for " + localPostDate.Weekday().String()
	} else if localPostDate.After(lastWeek) && localPostDate.Before(yesterday) {
		postDate = localPostDate.Weekday().String()
	} else if localPostDate.After(yesterday) && localPostDate.Before(todayStart) {
		postDate = "yesterday"
	} else if localPostDate.Before(todayEnd) {
		postDate = "today"
	} else if localPostDate.Before(tmrEnd) {
		postDate = "tomorrow"
	}

	if localPostDate.Hour() != 0 || localPostDate.Minute() != 0 {
		postDate += localPostDate.Format(", 15:04")
	}

	return postDate
}

// Generate a single task and format it in HTML (for the list of tasks)
func genTask(assignment plat.Task, noteType string, creds User) taskItem {
	task := taskItem{
		Id:       assignment.Id,
		Name:     assignment.Name,
		Platform: assignment.Platform,
		Class:    assignment.Class,
		URL:      assignment.Link,
	}

	switch noteType {
	case "dueDate":
		task.DueDate = genDueStr(assignment.Due, creds)
	case "posted":
		task.Posted = genPostStr(assignment.Posted, creds)
	case "grade":
		task.Grade = assignment.Result.Grade
		if task.Grade != "" && assignment.Result.Mark != 0.0 {
			task.Grade += fmt.Sprintf(" (%.f%%)", assignment.Result.Mark)
		} else if assignment.Result.Mark != 0.0 {
			task.Grade = fmt.Sprintf("%.f%%", assignment.Result.Mark)
		} else {
			task.Grade = "N/A"
		}
	}

	return task
}

// Generate the HTML page for viewing a single task
func genTaskPage(assignment plat.Task, creds User) pageData {
	data := pageData{
		PageType: "task",
		Head: headData{
			Title: assignment.Name,
		},
		Body: bodyData{
			TaskData: taskData{
				Id:          assignment.Id,
				Name:        assignment.Name,
				Platform:    assignment.Platform,
				Class:       assignment.Class,
				URL:         assignment.Link,
				IsDue:       false,
				Desc:        "",
				ResLinks:    nil,
				WorkLinks:   nil,
				HasResLinks: false,
			},
		},
		User: userData{
			Name: creds.DispName,
		},
	}

	if !assignment.Due.IsZero() {
		data.Body.TaskData.IsDue = true
		data.Body.TaskData.DueDate = genDueStr(assignment.Due, creds)
	}

	if !assignment.Submitted {
		data.Body.TaskData.Submitted = false
	}

	if assignment.Desc != "" {
		taskDesc := assignment.Desc
		// Escape strings since it will be converted to safe HTML after
		taskDesc = html.EscapeString(taskDesc)
		taskDesc = strings.ReplaceAll(taskDesc, "\n", "<br>")
		data.Body.TaskData.Desc = template.HTML(taskDesc)
	}

	if assignment.ResLinks != nil {
		data.Body.TaskData.HasResLinks = true

		data.Body.TaskData.ResLinks = make(map[string]string)
		for i := 0; i < len(assignment.ResLinks); i++ {
			url := assignment.ResLinks[i][0]
			name := assignment.ResLinks[i][1]
			data.Body.TaskData.ResLinks[name] = url
		}
	}

	//logger.Info("%+v\n", data.Body.TaskData.ResLinks)

	if assignment.Upload {
		data.Body.TaskData.HasUpload = true

		data.Body.TaskData.WorkLinks = make(map[string]string)
		for i := 0; i < len(assignment.WorkLinks); i++ {
			url := assignment.WorkLinks[i][0]
			name := assignment.WorkLinks[i][1]
			data.Body.TaskData.WorkLinks[name] = url
		}
	}

	if assignment.Result.Grade != "" {
		data.Body.TaskData.TaskGrade.Grade = assignment.Result.Grade
	} else {
		data.Body.TaskData.TaskGrade.Grade = "N/A"
	}

	bgColor := color.RGBA{0x00, 0x00, 0x00, 0x00}
	data.Body.TaskData.TaskGrade.Mark = fmt.Sprintf("%.f%%", assignment.Result.Mark)

	if assignment.Result.Mark != 0.0 {
		if assignment.Result.Mark < 50 {
			bgColor = gradeColors[0] // Red
		} else if (50 <= assignment.Result.Mark) && (assignment.Result.Mark < 70) {
			bgColor = gradeColors[1] // Amber/Orange
		} else if (70 <= assignment.Result.Mark) && (assignment.Result.Mark < 85) {
			bgColor = gradeColors[2] // Yellow
		} else if assignment.Result.Mark >= 85 {
			bgColor = gradeColors[3] // Green
		}
		textColor := "#ffffff"
		luminance := (0.299*float32(bgColor.R) + 0.587*float32(bgColor.G) + 0.114*float32(bgColor.B)) / 255
		if luminance > 0.5 {
			textColor = "#000000"
		}
		data.Body.TaskData.TaskGrade.Color = textColor
	} else {
		data.Body.TaskData.TaskGrade.Color = "" // Blank string so it will default to the correct color
	}

	data.Body.TaskData.TaskGrade.BGColor = fmt.Sprintf("#%02x%02x%02x%02x", bgColor.R, bgColor.G, bgColor.B, bgColor.A)

	if assignment.Comment != "" {
		taskCmt := assignment.Comment
		// Escape strings since it will be converted to safe HTML after
		taskCmt = html.EscapeString(taskCmt)
		taskCmt = strings.ReplaceAll(taskCmt, "\n", "<br>")
		data.Body.TaskData.Comment = template.HTML(taskCmt)
	}

	return data
}

// Generate the HTML page for viewing a single resource
func genResPage(res plat.Resource, creds User) pageData {
	data := pageData{
		PageType: "resource",
		Head: headData{
			Title: res.Name,
		},
		Body: bodyData{
			ResourceData: resourceData{
				Id:          res.Id,
				Name:        res.Name,
				Platform:    res.Platform,
				Class:       res.Class,
				URL:         res.Link,
				Desc:        "",
				Posted:      genPostStr(res.Posted, creds),
				ResLinks:    nil,
				HasResLinks: false,
			},
		},
		User: userData{
			Name: creds.DispName,
		},
	}

	if res.Desc != "" {
		resDesc := res.Desc
		// Escape strings since it will be converted to safe HTML after
		resDesc = html.EscapeString(resDesc)
		resDesc = strings.ReplaceAll(resDesc, "\n", "<br>")
		data.Body.ResourceData.Desc = template.HTML(resDesc)
	}

	if res.ResLinks != nil {
		data.Body.ResourceData.HasResLinks = true

		data.Body.ResourceData.ResLinks = make(map[string]string)
		for i := 0; i < len(res.ResLinks); i++ {
			url := res.ResLinks[i][0]
			name := res.ResLinks[i][1]
			data.Body.ResourceData.ResLinks[name] = url
		}
	}

	return data
}

// Generate a resource link
func genHtmlResLink(className string, res []plat.Resource, creds User) resClass {
	class := resClass{
		Name: className,
	}

	for _, r := range res {
		class.ResItems = append(class.ResItems, resItem{
			Id:       r.Id,
			Name:     r.Name,
			Posted:   genPostStr(r.Posted, creds),
			Platform: r.Platform,
			URL:      r.Link,
		})
	}

	return class
}

// Generate resources and components for the webpage
func genRes(resPath string, resURL string, creds User) (pageData, error) {
	var data pageData
	data.User.Name = creds.DispName

	if resURL == "/timetable" {
		data.PageType = "timetable"
		data.Head.Title = "Timetable"

		timetable, err := genTimetable(creds)
		if err != nil {
			return data, errors.NewError("server.genRes", "failed to generate timetable", err)
		}

		data.Body.TimetableData = timetable

	} else if resURL == "/tasks" {
		data.PageType = "tasks"
		data.Head.Title = "Tasks"
		data.Body.TasksData.Heading = "Tasks"

		tasks := getTasks(creds)
		activeTasks := taskType{
			Name:     "Active tasks",
			NoteType: "dueDate",
		}
		for i := 0; i < len(tasks["active"]); i++ {
			activeTasks.Tasks = append(activeTasks.Tasks, genTask(
				tasks["active"][i],
				"dueDate",
				creds,
			))
		}
		data.Body.TasksData.TaskTypes = append(data.Body.TasksData.TaskTypes, activeTasks)

		notDueTasks := taskType{
			Name:     "No due date",
			NoteType: "posted",
		}
		for i := 0; i < len(tasks["notDue"]); i++ {
			notDueTasks.Tasks = append(notDueTasks.Tasks, genTask(
				tasks["notDue"][i],
				"posted",
				creds,
			))
		}
		data.Body.TasksData.TaskTypes = append(data.Body.TasksData.TaskTypes, notDueTasks)

		overdueTasks := taskType{
			Name:     "Overdue tasks",
			NoteType: "dueDate",
		}
		for i := 0; i < len(tasks["overdue"]); i++ {
			overdueTasks.Tasks = append(overdueTasks.Tasks, genTask(
				tasks["overdue"][i],
				"dueDate",
				creds,
			))
		}
		data.Body.TasksData.TaskTypes = append(data.Body.TasksData.TaskTypes, overdueTasks)

		submittedTasks := taskType{
			Name:     "Submitted tasks",
			NoteType: "posted",
		}
		for i := 0; i < len(tasks["submitted"]); i++ {
			submittedTasks.Tasks = append(submittedTasks.Tasks, genTask(
				tasks["submitted"][i],
				"posted",
				creds,
			))
		}
		data.Body.TasksData.TaskTypes = append(data.Body.TasksData.TaskTypes, submittedTasks)

	} else if resURL == "/res" {
		data.PageType = "resources"
		data.Head.Title = "Resources"
		data.Body.ResData.Heading = "Resources"

		classes, resources := getResources(creds)
		for _, class := range classes {
			data.Body.ResData.Classes = append(data.Body.ResData.Classes, genHtmlResLink(
				class,
				resources[class],
				creds,
			))
		}

	} else if resURL == "/grades" {
		data.PageType = "grades"
		data.Head.Title = "Grades"

		tasks := gradedTasks(creds)
		for _, task := range tasks {
			data.Body.GradesData.Tasks = append(
				data.Body.GradesData.Tasks,
				genTask(task, "grade", creds),
			)
		}

	} else {
		return data, errors.ErrNotFound
	}

	return data, nil
}
