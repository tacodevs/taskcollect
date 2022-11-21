package daymap

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"main/errors"
	"main/logger"
)

type Task struct {
	Name      string
	Class     string
	Link      string
	Desc      string
	Due       time.Time
	Posted    time.Time
	ResLinks  [][2]string
	Upload    bool
	WorkLinks [][2]string
	Submitted bool
	Grade     string
	Comment   string
	Platform  string
	Id        string
}

/*func (u fileUploader) Read(p []byte) (int, error) {
	// TODO: Make sure the errors have contextual information

	mimeDiv := strings.NewReader(u.MimeDivider)
	mimeHead := strings.NewReader(u.MimeHeader)
	mimeend := strings.NewReader(u.MimeDivider + "--")
	n := 0

	for n < len(p) {
		b, err := mimeDiv.ReadByte()

		if err != nil && err != io.EOF {
			return n, err
		} else if err == nil {
			continue
		}

		b, err = mimeHead.ReadByte()

		if err != nil && err != io.EOF {
			return n, err
		} else if err == nil {
			continue
		}

		b, err = u.FileReader.ReadByte()

		if err != nil && err != io.EOF {
			return n, err
		} else if err == nil {
			continue
		}

		b, err = mimeend.ReadByte()

		if err != nil && err != io.EOF {
			return n, err
		} else if err == nil {
			continue
		}

		p[n] = b
		n++
	}

	return n, nil
}*/

// TODO: Could use the function from utils.go -> Make a local utils package
func contains(a []string, s string) bool {
	for _, c := range a {
		if s == c {
			return true
		}
	}
	return false
}

// Public function to retrieve information about a DayMap task by its ID.
func GetTask(creds User, id string) (Task, error) {
	taskUrl := "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + id

	task := Task{
		Link:     taskUrl,
		Platform: "daymap",
		Id:       id,
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", taskUrl, nil)
	if err != nil {
		newErr := errors.NewError("daymap: GetTask", "GET request failed", err)
		return Task{}, newErr
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		newErr := errors.NewError("daymap: GetTask", "failed to get resp", err)
		return Task{}, newErr
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := errors.NewError("daymap: GetTask", "failed to read resp.Body", err)
		return Task{}, newErr
	}

	b := string(respBody)

	if strings.Contains(b, "My&nbsp;Work") || strings.Contains(b, "My Work</div>") {
		task.Upload = true
	}

	i := strings.Index(b, "ctl00_ctl00_cp_cp_divResults")

	if i == -1 {
		return Task{}, errInvalidTaskResp
	}

	b = b[i:]
	i = strings.Index(b, "SectionHeader")

	if i == -1 {
		return Task{}, errInvalidTaskResp
	}

	b = b[i:]
	i = len("SectionHeader") + 2
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return Task{}, errInvalidTaskResp
	}

	task.Name = b[:i]
	b = b[i:]
	i = strings.Index(b, "<div style='padding:6px'>")

	if i == -1 {
		return Task{}, errInvalidTaskResp
	}

	b = b[i:]
	i = len("<div style='padding:6px'>")
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return Task{}, errInvalidTaskResp
	}

	task.Class = b[:i]
	b = b[i:]
	i = strings.Index(b, "<div style='padding:6px'>")

	if i == -1 {
		return Task{}, errInvalidTaskResp
	}

	b = b[i:]
	i = len("<div style='padding:6px'>")
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return Task{}, errInvalidTaskResp
	}

	b = b[i:]
	i = strings.Index(b, "Due on ")

	if i != -1 {
		b = b[i:]
		i = len("Due on ")
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return Task{}, errInvalidTaskResp
		}

		dueStr := b[:i]
		b = b[i:]
		var due time.Time

		if !strings.Contains(dueStr, ":") {
			due, err = time.Parse("2/01/2006", dueStr)
		} else {
			due, err = time.Parse("2/01/2006 3:04 PM", dueStr)
		}

		if err != nil {
			newErr := errors.NewError("daymap: GetTask", "failed to parse time", err)
			return Task{}, newErr
		}

		task.Due = time.Date(
			due.Year(), due.Month(), due.Day(),
			due.Hour(), due.Minute(), 0, 0,
			creds.Timezone,
		)
	}

	i = strings.Index(b, "My Work</div>")

	if i != -1 {
		b = b[i:]
		i = strings.Index(b, "<div><div>")

		if i == -1 {
			return Task{}, errInvalidTaskResp
		}

		b = b[i:]
		i = strings.Index(b, "</div></div></div></div>")

		if i == -1 {
			return Task{}, errInvalidTaskResp
		}

		wlHtml := b[:i]
		b = b[i:]
		x := strings.Index(wlHtml, `<a href="`)

		for x != -1 {
			x += len(`<a href="`)
			wlHtml = wlHtml[x:]
			x = strings.Index(wlHtml, `"`)

			if x == -1 {
				return Task{}, errInvalidTaskResp
			}

			wll := wlHtml[:x]
			wlHtml = wlHtml[x:]
			link := "https://gihs.daymap.net" + wll
			x = strings.Index(wlHtml, "&nbsp;")

			if x == -1 {
				return Task{}, errInvalidTaskResp
			}

			x += len("&nbsp;")
			wlHtml = wlHtml[x:]
			x = strings.Index(wlHtml, "</a>")

			if x == -1 {
				return Task{}, errInvalidTaskResp
			}

			name := wlHtml[:x]
			wlHtml = wlHtml[x:]
			task.WorkLinks = append(task.WorkLinks, [2]string{link, name})
			x = strings.Index(wlHtml, `<a href="`)
		}
	}

	i = strings.Index(b, "Grade:")

	if i != -1 {
		i = strings.Index(b, "TaskGrade'>")

		if i == -1 {
			return Task{}, errInvalidTaskResp
		}

		b = b[i:]
		i = len("TaskGrade'>")
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return Task{}, errInvalidTaskResp
		}

		task.Grade = b[:i]
		b = b[i:]
	}

	i = strings.Index(b, "Mark:")

	if i != -1 {
		i = strings.Index(b, "TaskGrade'>")

		if i == -1 {
			return Task{}, errInvalidTaskResp
		}

		b = b[i:]
		i = len("TaskGrade'>")
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return Task{}, errInvalidTaskResp
		}

		markStr := b[:i]
		b = b[i:]

		x := strings.Index(markStr, " / ")

		if x == -1 {
			return Task{}, errInvalidTaskResp
		}

		st := markStr[:x]
		sb := markStr[x+3:]

		it, err := strconv.Atoi(st)
		if err != nil {
			newErr := errors.NewError("daymap: GetTask", "(1) string -> int conversion failed", err)
			return Task{}, newErr
		}

		ib, err := strconv.Atoi(sb)
		if err != nil {
			newErr := errors.NewError("daymap: GetTask", "(2) string -> int conversion failed", err)
			return Task{}, newErr
		}

		percent := float64(it) / float64(ib) * 100

		if task.Grade == "" {
			task.Grade = fmt.Sprintf("%.f%%", percent)
		} else {
			task.Grade += fmt.Sprintf(" (%.f%%)", percent)
		}
	}

	i = strings.Index(b, `class="WhiteBox">`)

	if i != -1 {
		b = b[i:]
		i = len(`class="WhiteBox">`)
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return Task{}, errInvalidTaskResp
		}

		task.Comment = b[:i]
		b = b[i:]
	}

	i = strings.Index(b, "Attachments</div>")

	if i != -1 {
		b = b[i:]
		i = len("Attachments</div>")
		b = b[i:]
		i = strings.Index(b, `class='WhiteBox' style='padding:5px;margin:2px'>`)

		if i == -1 {
			i = strings.Index(b, "\n")
		}

		if i == -1 {
			return Task{}, errInvalidTaskResp
		}

		rlHtml := b[:i]
		b = b[i:]
		x := strings.Index(rlHtml, "DMU.OpenAttachment(")

		for x != -1 {
			x += len("DMU.OpenAttachment(")
			rlHtml = rlHtml[x:]
			x = strings.Index(rlHtml, ")")

			if x == -1 {
				return Task{}, errInvalidTaskResp
			}

			rlId := rlHtml[:x]
			rlHtml = rlHtml[x:]
			link := "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + rlId
			x = strings.Index(rlHtml, "&nbsp;")

			if x == -1 {
				return Task{}, errInvalidTaskResp
			}

			x += len("&nbsp;")
			rlHtml = rlHtml[x:]
			x = strings.Index(rlHtml, "</a>")

			if x == -1 {
				return Task{}, errInvalidTaskResp
			}

			name := rlHtml[:x]
			rlHtml = rlHtml[x:]
			task.ResLinks = append(task.ResLinks, [2]string{link, name})
			x = strings.Index(rlHtml, "DMU.OpenAttachment(")
		}
	}

	i = strings.Index(b, `class='WhiteBox' style='padding:5px;margin:2px'>`)

	if i != -1 {
		b = b[i:]
		i = len(`class='WhiteBox' style='padding:5px;margin:2px'>`)
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return Task{}, errInvalidTaskResp
		}

		task.Desc = b[:i]
	}

	task.Submitted = true
	return task, nil
}

// TODO: Complete the below function.
// https://gihs.daymap.net/daymap/Resources/AttachmentAdd.aspx?t=2&LinkID=78847
func UploadWork(creds User, id string, r *http.Request) error {
	/*
		uploadUrl := "URL TO UPLOAD TO"

		mimeDiv := "--MultipartMimeHtmlFormBoundaryPiFa8ZSp8tLEoC81"
		mimeHead := `Content-Disposition: form-data; name="file"; filename="`
		mimeHead += filename + "\"\nContent-Type: application/octet-stream\n\n"

		uploader := fileUploader{
			MimeDivider: mimeDiv,
			MimeHeader: mimeHead,
			FileReader: *f,
		}

		client := &http.Client{}
		req, err := http.NewRequest("POST", uploadUrl, uploader)

		if err != nil {
			// use errors.NewError()
			return err
		}

		req.Header.Set(
			"Content-Type",
			`multipart/form-data; boundary="` + mimeDiv + `"`,
		)

		req.Header.Set("Cookie", creds.Token)
		_, err := client.Do(req)
		return err
	*/

	return nil
}

/*
ISSUE: Although the below function theoretically works, in practice, for some
reason, it does not.
*/

func RemoveWork(creds User, id string, filenames []string) error {
	removeUrl := "https://gihs.daymap.net/daymap/student/attachments.aspx?Type=1&LinkID="
	removeUrl += id
	client := &http.Client{}

	req, err := http.NewRequest("GET", removeUrl, nil)
	if err != nil {
		newErr := errors.NewError("daymap: RemoveWork", "GET request failed", err)
		return newErr
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		newErr := errors.NewError("daymap: RemoveWork", "failed to get resp", err)
		return newErr
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := errors.NewError("daymap: RemoveWork", "failed to read resp.Body", err)
		return newErr
	}

	b := string(respBody)
	i := strings.Index(b, "<form")

	if i == -1 {
		return errInvalidTaskResp
	}

	b = b[i:]
	i = strings.Index(b, ` action="`)

	if i == -1 {
		return errInvalidTaskResp
	}

	b = b[i:]
	i = len(` action="`)
	b = b[i:]
	i = strings.Index(b, `"`)

	if i == -1 {
		return errInvalidTaskResp
	}

	rwUrl := b[:i]
	rwUrl = html.UnescapeString(rwUrl)
	b = b[i:]
	rwForm := url.Values{}
	i = strings.Index(b, "<input ")

	for i != -1 {
		var name, value string
		b = b[i:]
		i = strings.Index(b, ` type=`)

		if i == -1 {
			return errInvalidTaskResp
		}

		b = b[i:]
		i = len(` type=`)
		b = b[i:]
		i = strings.Index(b, ` `)

		if i == -1 {
			return errInvalidTaskResp
		}

		inputType := b[:i]
		b = b[i:]
		i = strings.Index(b, `name="`)

		if i == -1 {
			return errInvalidTaskResp
		}

		b = b[i:]
		i = len(`name="`)
		b = b[i:]
		i = strings.Index(b, `"`)

		if i == -1 {
			return errInvalidTaskResp
		}

		name = b[:i]
		b = b[i:]

		i = strings.Index(b, "\n")

		if i == -1 {
			return errInvalidTaskResp
		}

		valTest := b[:i]
		i = strings.Index(valTest, ` value="`)

		if i != -1 {
			b = b[i:]
			i = len(` value="`)
			b = b[i:]
			i = strings.Index(b, `"`)

			if i == -1 {
				return errInvalidTaskResp
			}

			value = b[:i]
			b = b[i:]
		}

		if inputType != "checkbox" {
			rwForm.Set(name, value)
			i = strings.Index(b, "<input ")
			continue
		}

		i = strings.Index(b, `<span name=filename>`)

		if i == -1 {
			return errInvalidTaskResp
		}

		b = b[i:]
		i = len(`<span name=filename>`)
		b = b[i:]
		i = strings.Index(b, `</span>`)

		if i == -1 {
			return errInvalidTaskResp
		}

		fname := b[:i]
		b = b[i:]

		if contains(filenames, fname) {
			rwForm.Set(name, "del")
		}

		i = strings.Index(b, "<input ")
	}

	rwForm.Set("Cmd", "delete")
	rwForm.Set("__EVENTTARGET", "")
	rwForm.Set("__EVENTARGUMENT", "")

	rwData := strings.NewReader(rwForm.Encode())
	rwfurl := "https://gihs.daymap.net/daymap/student" + rwUrl[1:]
	fmt.Println(rwForm)

	post, err := http.NewRequest("POST", rwfurl, rwData)
	if err != nil {
		newErr := errors.NewError("daymap: RemoveWork", "POST request failed", err)
		return newErr
	}

	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.Header.Set("Cookie", creds.Token)

	fail, err := client.Do(req)
	if err != nil {
		newErr := errors.NewError("daymap: RemoveWork", "error returning response body", err)
		return newErr
	}

	logger.Error("%v", fail.Body)
	return nil
}
