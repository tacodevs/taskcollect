package server

import (
	"sort"
	"time"

	"git.sr.ht/~kvo/go-std/errors"

	"main/logger"
	"main/site"
	"main/site/daymap"
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
		logger.Debug(errors.New("cannot fetch task lists", err))
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
		return filtered["overdue"][i].Posted.Unix() > filtered["overdue"][j].Posted.Unix()
	})
	sort.SliceStable(filtered["active"], func(i, j int) bool {
		return filtered["active"][i].Due.Unix() < filtered["active"][j].Due.Unix()
	})
	return filtered
}

func getResources(user site.User) ([]string, map[string][]site.Resource) {
	dmResChan := make(chan []site.Resource)
	dmErrChan := make(chan []error)

	dmCreds := daymap.User{
		Timezone: user.Timezone,
		Token:    user.SiteTokens["daymap"],
	}

	go daymap.ListRes(dmCreds, dmResChan, dmErrChan)

	unordered := map[string][]site.Resource{}

	dmResLinks, errs := <-dmResChan, <-dmErrChan
	for _, err := range errs {
		if err != nil {
			logger.Debug(errors.New("failed to get list of resources from daymap", err))
		}
	}

	for _, r := range dmResLinks {
		unordered[r.Class] = append(unordered[r.Class], site.Resource(r))
	}

	resources := map[string][]site.Resource{}
	classes := []string{}

	for c := range unordered {
		classes = append(classes, c)
	}

	sort.Strings(classes)

	for c, resList := range unordered {
		times := map[int]int{}
		resIndexes := []int{}

		for i, r := range resList {
			posted := int(r.Posted.UTC().Unix())
			times[i] = posted
			resIndexes = append(resIndexes, i)
		}

		sort.SliceStable(resIndexes, func(i, j int) bool {
			return times[resIndexes[i]] > times[resIndexes[j]]
		})

		for _, x := range resIndexes {
			resources[c] = append(resources[c], resList[x])
		}
	}

	return classes, resources
}

// Get a resource from the given platform.
func getResource(platform, resId string, user site.User) (site.Resource, error) {
	res := site.Resource{}
	err := errors.Raise(site.ErrNoPlatform)

	switch platform {
	case "daymap":
		dmCreds := daymap.User{
			Timezone: user.Timezone,
			Token:    user.SiteTokens["daymap"],
		}
		dmRes, dmErr := daymap.GetResource(dmCreds, resId)
		res = site.Resource(dmRes)
		err = dmErr
	}

	return res, err
}
