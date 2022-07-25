package daymap

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func ListTasks(creds User, t chan map[string][]Task, e chan error) {
	tasksUrl := "https://gihs.daymap.net/daymap/student/assignments.aspx?View=0"
	client := &http.Client{}

	req, err := http.NewRequest("GET", tasksUrl, nil)
	if err != nil {
		t <- nil
		e <- err
		return
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)
	if err != nil {
		t <- nil
		e <- err
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t <- nil
		e <- err
		return
	}

	// TODO: What exactly does "tForm" mean?
	tForm := url.Values{}
	b := string(respBody)
	i := strings.Index(b, "<input ")

	for i != -1 {
		var value string
		b = b[i:]
		i = strings.Index(b, ">")

		if i == -1 {
			panic("1")
			t <- nil
			e <- errors.New("DayMap: invalid HTML response")
			return
		}

		inputTag := b[:i]
		b = b[i:]
		i = strings.Index(inputTag, `type="`)

		if i == -1 {
			panic("2")
			t <- nil
			e <- errors.New("DayMap: invalid HTML response")
			return
		}

		i += len(`type="`)
		inputType := inputTag[i:]
		i = strings.Index(inputType, `"`)

		if i == -1 {
			panic("3")
			t <- nil
			e <- errors.New("DayMap: invalid HTML response")
			return
		}

		inputType = inputType[:i]

		if inputType != "hidden" {
			i = strings.Index(b, "<input ")
			continue
		}

		i = strings.Index(inputTag, `name="`)

		if i == -1 {
			panic("4")
			t <- nil
			e <- errors.New("DayMap: invalid HTML response")
			return
		}

		i += len(`name="`)
		name := inputTag[i:]
		i = strings.Index(name, `"`)

		if i == -1 {
			panic("5")
			t <- nil
			e <- errors.New("DayMap: invalid HTML response")
			return
		}

		name = name[:i]
		i = strings.Index(inputTag, `value="`)

		if i != -1 {
			i += len(`value="`)
			value = inputTag[i:]
			i = strings.Index(value, `"`)

			if i == -1 {
				panic("7")
				t <- nil
				e <- errors.New("DayMap: invalid HTML response")
				return
			}

			value = value[:i]
		}

		tForm.Set(name, value)
		i = strings.Index(b, "<input ")
	}

	tForm.Set(`ctl00_ctl00_cp_cp_grdAssignments_ctl00_ctl03_ctl01_PageSizeComboBox_ClientState`, `{"logEntries":[],"value":"50","text":"50","enabled":true,"checkedIndices":[],"checkedItemsTextOverflows":false}`)
	tForm.Set(`ctl00$ctl00$cp$cp$grdAssignments$ctl00$ctl03$ctl01$PageSizeComboBox`, `1000000000`)
	tForm.Set(`__EVENTTARGET`, `ctl00$ctl00$cp$cp$grdAssignments`)
	tForm.Set(`__EVENTARGUMENT`, `FireCommand:ctl00$ctl00$cp$cp$grdAssignments$ctl00;PageSize;1000000000`)
	tForm.Set(`ctl00_ctl00_cp_cp_ScriptManager_TSM`, `;;System.Web.Extensions, Version=4.0.0.0, Culture=neutral, PublicKeyToken=31bf3856ad364e35:en-AU:9ddf364d-d65d-4f01-a69e-8b015049e026:ea597d4b:b25378d2;Telerik.Web.UI, Version=2020.1.219.45, Culture=neutral, PublicKeyToken=121fae78165ba3d4:en-AU:bb184598-9004-47ca-9e82-5def416be84b:16e4e7cd:33715776:58366029:f7645509:24ee1bba:f46195d3:2003d0b8:c128760b:88144a7a:1e771326:aa288e2d:258f1c72`)

	tdata := strings.NewReader(tForm.Encode())
	fullReq, err := http.NewRequest("POST", tasksUrl, tdata)

	if err != nil {
		t <- nil
		e <- err
		return
	}

	fullReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	fullReq.Header.Set("Cookie", creds.Token)
	full, err := client.Do(fullReq)

	if err != nil {
		t <- nil
		e <- err
		return
	}

	fullBody, err := ioutil.ReadAll(full.Body)

	if err != nil {
		t <- nil
		e <- err
		return
	}

	b = string(fullBody)
	os.Stderr.WriteString(fmt.Sprintln(b))

	// TODO: Parse full list of tasks, sort through list and return.

	/*
		task2 := Task{
			Name: "TaskCollect",
			Class: "Digital Technologies",
			Link: "https://openbsd.org",
			Due: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			Submitted: false,
			Platform: "foo",
			Id: "34573453-4353453",
		}

		tasks := map[string][]Task{
			"tasks": {task2, task2},
			"notdue": {task2, task2},
			"overdue": {task2, task2},
			"submitted": {task2, task2},
		}

		t <- tasks
	*/

	t <- nil
	e <- nil
}
