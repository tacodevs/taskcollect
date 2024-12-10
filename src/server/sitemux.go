package server

import (
	"sort"
	"time"

	"git.sr.ht/~kvo/go-std/errors"

	"main/logger"
	"main/site"
)

// TODO: delete after v2
func getTasks(user site.User) map[string][]site.Task {
	filtered := map[string][]site.Task{
		"active":    {},
		"notDue":    {},
		"overdue":   {},
		"submitted": {},
	}
	school, ok := schools[user.School]
	if !ok {
		logger.Debug(errors.New("unsupported platform", nil))
		return filtered
	}
	classes, err := school.Classes(user)
	if err != nil {
		logger.Debug(errors.New("cannot fetch class list", err))
		return filtered
	}
	tasks, err := school.Tasks(user, classes...)
	if err != nil {
		logger.Debug(errors.New("cannot fetch tasks list", err))
		return filtered
	}
	for _, task := range tasks {
		if task.Graded {
			continue
		} else if task.Submitted {
			filtered["submitted"] = append(filtered["submitted"], task)
		} else if task.Due.IsZero() {
			filtered["notDue"] = append(filtered["notDue"], task)
		} else if task.Due.Before(time.Now()) {
			filtered["overdue"] = append(filtered["overdue"], task)
		} else {
			filtered["active"] = append(filtered["active"], task)
		}
	}
	sort.SliceStable(filtered["submitted"], func(i, j int) bool {
		return filtered["submitted"][i].Posted.Unix() > filtered["submitted"][j].Posted.Unix()
	})
	sort.SliceStable(filtered["notDue"], func(i, j int) bool {
		return filtered["notDue"][i].Posted.Unix() > filtered["notDue"][j].Posted.Unix()
	})
	sort.SliceStable(filtered["overdue"], func(i, j int) bool {
		return filtered["overdue"][i].Due.Unix() > filtered["overdue"][j].Due.Unix()
	})
	sort.SliceStable(filtered["active"], func(i, j int) bool {
		return filtered["active"][i].Due.Unix() < filtered["active"][j].Due.Unix()
	})
	return filtered
}

// TODO: refactor
func getResources(user site.User) ([]string, map[string][]site.Resource) {
	var classList []string
	resMap := make(map[string][]site.Resource)
	school, ok := schools[user.School]
	if !ok {
		logger.Debug(errors.New("unsupported platform", nil))
		return classList, resMap
	}
	classes, err := school.Classes(user)
	if err != nil {
		logger.Debug(errors.New("cannot fetch class list", err))
		return classList, resMap
	}
	resources, err := school.Resources(user, classes...)
	if err != nil {
		logger.Debug(errors.New("cannot fetch resources list", err))
		return classList, resMap
	}
	for _, resource := range resources {
		resMap[resource.Class] = append(resMap[resource.Class], resource)
	}
	for class := range resMap {
		classList = append(classList, class)
	}
	sort.Strings(classList)
	for class := range resMap {
		sort.SliceStable(resMap[class], func(i, j int) bool {
			return resMap[class][i].Posted.Unix() > resMap[class][j].Posted.Unix()
		})
	}
	return classList, resMap
}
