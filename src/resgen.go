package main

import (
	"html"
	"html/template"
	"strconv"
	"strings"
	"time"

	"main/errors"
)

func genDueStr(due time.Time, creds tcUser) string {
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
	weekEnd := todayStart.AddDate(0, 0, 7)

	if localDueDate.Before(todayStart) || !localDueDate.Before(weekEnd) {
		dueDate = strconv.Itoa(localDueDate.Day())
		dueDate += " " + localDueDate.Month().String()[:3]
		if localDueDate.Year() != now.Year() {
			dueDate += " " + strconv.Itoa(localDueDate.Year())
		}
	} else if localDueDate.Before(todayEnd) {
		dueDate = "Today"
	} else if localDueDate.Before(tmrEnd) {
		dueDate = "Tomorrow"
	} else if localDueDate.Before(weekEnd) {
		dueDate = localDueDate.Weekday().String()
	}

	if localDueDate.Hour() != 0 || localDueDate.Minute() != 0 {
		strHour := strconv.Itoa(localDueDate.Hour())
		if len(strHour) == 1 {
			strHour = "0" + strHour
		}

		strMinute := strconv.Itoa(localDueDate.Minute())
		if len(strMinute) == 1 {
			strMinute = "0" + strMinute
		}

		dueDate += ", " + strHour + ":" + strMinute
	}

	return dueDate
}

// Generate a single task and format it in HTML (for the list of tasks)
func genTask(assignment task, noteType string, creds tcUser) taskItem {
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
		task.Posted = genDueStr(assignment.Posted, creds)
	case "grade":
		task.Grade = assignment.Grade
	}

	return task
}

// Generate the HTML page for viewing a single task
func genTaskPage(assignment task, creds tcUser) pageData {
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

	if assignment.Grade != "" {
		data.Body.TaskData.Grade = assignment.Grade
	}

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
func genResPage(res resource, creds tcUser) pageData {
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
				Posted:      genDueStr(res.Posted, creds),
				ResLinks:    nil,
				HasResLinks: false,
			},
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
func genHtmlResLink(className string, res []resource, creds tcUser) resClass {
	class := resClass{
		Name: className,
	}

	for _, r := range res {
		class.ResItems = append(class.ResItems, resItem{
			Id:       r.Id,
			Name:     r.Name,
			Posted:   genDueStr(r.Posted, creds),
			Platform: r.Platform,
			URL:      r.Link,
		})
	}

	return class
}

// Generate resources and components for the webpage
func genRes(resPath string, resURL string, creds tcUser) (pageData, error) {
	var data pageData

	if resURL == "/timetable" {
		data.PageType = "timetable"
		data.Head.Title = "Timetable"

		timetable, err := genTimetable(creds)
		if err != nil {
			newErr := errors.NewError("main.genRes", "failed to generate timetable", err)
			return data, newErr
		}

		data.Body.TimetableData = timetable

	} else if resURL == "/tasks" {
		data.PageType = "tasks"
		data.Head.Title = "Tasks"
		data.Body.TasksData.Heading = "Tasks"

		tasks, err := getTasks(creds)
		if err != nil {
			newErr := errors.NewError("main.genRes", "failed to get tasks", err)
			return data, newErr
		}

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

		classes, resources, err := getResources(creds)
		if err != nil {
			newErr := errors.NewError("main.genRes", "failed to get resources", err)
			return data, newErr
		}

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

		tasks, err := gradedTasks(creds)
		if err != nil {
			newErr := errors.NewError("main.genRes", "failed to get graded tasks", err)
			return data, newErr
		}

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
