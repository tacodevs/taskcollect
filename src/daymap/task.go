package daymap

import (
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Task struct {
	Name      string
	Class     string
	Link      string
	Desc      string
	Due       time.Time
	ResLinks  [][2]string
	Upload    bool
	WorkLinks [][2]string
	Submitted bool
	Grade     string
	Comment   string
	Platform  string
	Id        string
}

type fileUploader struct {
	MimeDivider string
	MimeHeader  string
	FileReader  io.Reader
}

/*func (u fileUploader) Read(p []byte) (int, error) {
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

func contains(a []string, s string) bool {
	for _, c := range a {
		if s == c {
			return true
		}
	}
	return false
}

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
		return Task{}, err
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		return Task{}, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Task{}, err
	}

	b := string(respBody)

	if strings.Index(b, "My&nbsp;Work") != -1 || strings.Index(b, "My Work</div>") != -1 {
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

		if strings.Index(dueStr, ":") == -1 {
			due, err = time.Parse("2/01/2006", dueStr)
		} else {
			due, err = time.Parse("2/01/2006 3:04 PM", dueStr)
		}

		if err != nil {
			return Task{}, err
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
			return Task{}, err
		}

		ib, err := strconv.Atoi(sb)

		if err != nil {
			return Task{}, err
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
		b = b[i:]
	}

	task.Submitted = true
	return task, nil
}

// TODO: Complete the below function.
// https://gihs.daymap.net/daymap/Resources/AttachmentAdd.aspx?t=2&LinkID=78847
func UploadWork(creds User, id string, filename string, f *io.Reader) error {
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

	_, err := io.Copy(os.Stdout, *f)
	return err
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
		return err
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	respBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
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
		return err
	}

	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.Header.Set("Cookie", creds.Token)

	fail, err := client.Do(req)
	if err != nil {
		return err
	}

	io.Copy(os.Stderr, fail.Body)
	return nil
}
