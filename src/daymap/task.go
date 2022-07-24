package daymap

import (
	"errors"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"strconv"
	"time"
)

type Task struct {
	Name string
	Class string
	Link string
	Desc string
	Due time.Time
	Reslinks [][2]string
	Upload bool
	Worklinks [][2]string
	Submitted bool
	Grade string
	Comment string
	Platform string
	Id string
}

type fileUploader struct {
	MimeDivider	string
	MimeHeader	string
	FileReader	io.Reader
}

/*func (u fileUploader) Read(p []byte) (int, error) {
	mimediv := strings.NewReader(u.MimeDivider)
	mimehead := strings.NewReader(u.MimeHeader)
	mimeend := strings.NewReader(u.MimeDivider + "--")
	n := 0

	for n < len(p) {
		b, err := mimediv.ReadByte()

		if err != nil && err != io.EOF {
			return n, err
		} else if err == nil {
			continue
		}

		b, err = mimehead.ReadByte()

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
	taskurl := "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + id

	task := Task{
		Link: taskurl,
		Platform: "daymap",
		Id: id,		
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", taskurl, nil)

	if err != nil {
		return Task{}, err
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)

	if err != nil {
		return Task{}, err
	}

	rb, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return Task{}, err
	}

	b := string(rb)

	if strings.Index(b, "My&nbsp;Work") != -1 || strings.Index(b, "My Work</div>") != -1 {
		task.Upload = true
	}

	i := strings.Index(b, "ctl00_ctl00_cp_cp_divResults")

	if i == -1 {
		return Task{}, errors.New("daymap: invalid task HTML response")
	}

	b = b[i:]
	i = strings.Index(b, "SectionHeader")

	if i == -1 {
		return Task{}, errors.New("daymap: invalid task HTML response")
	}

	b = b[i:]
	i = len("SectionHeader") + 2
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return Task{}, errors.New("daymap: invalid task HTML response")
	}

	task.Name = b[:i]
	b = b[i:]
	i = strings.Index(b, "<div style='padding:6px'>")

	if i == -1 {
		return Task{}, errors.New("daymap: invalid task HTML response")
	}

	b = b[i:]
	i = len("<div style='padding:6px'>")
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return Task{}, errors.New("daymap: invalid task HTML response")
	}

	task.Class = b[:i]
	b = b[i:]
	i = strings.Index(b, "<div style='padding:6px'>")

	if i == -1 {
		return Task{}, errors.New("daymap: invalid task HTML response")
	}

	b = b[i:]
	i = len("<div style='padding:6px'>")
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return Task{}, errors.New("daymap: invalid task HTML response")
	}

	b = b[i:]
	i = strings.Index(b, "Due on ")

	if i != -1 {
		b = b[i:]
		i = len("Due on ")
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return Task{}, errors.New("daymap: invalid task HTML response")
		}

		duestr := b[:i]
		b = b[i:]
		var due time.Time

		if strings.Index(duestr, ":") == -1 {
			due, err = time.Parse("2/01/2006", duestr)
		} else {
			due, err = time.Parse("2/01/2006 3:04 PM", duestr)
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
			return Task{}, errors.New("daymap: invalid task HTML response")
		}

		b = b[i:]
		i = strings.Index(b, "</div></div></div></div>")

		if i == -1 {
			return Task{}, errors.New("daymap: invalid task HTML response")
		}

		wlhtml := b[:i]
		b = b[i:]
		x := strings.Index(wlhtml, `<a href="`)

		for x != -1 {
			x += len(`<a href="`)
			wlhtml = wlhtml[x:]
			x = strings.Index(wlhtml, `"`)

			if x == -1 {
				return Task{}, errors.New("daymap: invalid task HTML response")
			}

			wll := wlhtml[:x]
			wlhtml = wlhtml[x:]
			link := "https://gihs.daymap.net" + wll
			x = strings.Index(wlhtml, "&nbsp;")

			if x == -1 {
				return Task{}, errors.New("daymap: invalid task HTML response")
			}

			x += len("&nbsp;")
			wlhtml = wlhtml[x:]
			x = strings.Index(wlhtml, "</a>")

			if x == -1 {
				return Task{}, errors.New("daymap: invalid task HTML response")
			}

			name := wlhtml[:x]
			wlhtml = wlhtml[x:]
			task.Worklinks = append(task.Worklinks, [2]string{link, name})
			x = strings.Index(wlhtml, `<a href="`)
		}
	}

	i = strings.Index(b, "Grade:")

	if i != -1 {
		i = strings.Index(b, "TaskGrade'>")

		if i == -1 {
			return Task{}, errors.New("daymap: invalid task HTML response")
		}

		b = b[i:]
		i = len("TaskGrade'>")
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return Task{}, errors.New("daymap: invalid task HTML response")
		}

		task.Grade = b[:i]
		b = b[i:]
	}

	i = strings.Index(b, "Mark:")

	if i != -1 {
		i = strings.Index(b, "TaskGrade'>")

		if i == -1 {
			return Task{}, errors.New("daymap: invalid task HTML response")
		}

		b = b[i:]
		i = len("TaskGrade'>")
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return Task{}, errors.New("daymap: invalid task HTML response")
		}

		strmark := b[:i]
		b = b[i:]

		x := strings.Index(strmark, " / ")

		if x == -1 {
			return Task{}, errors.New("daymap: invalid task HTML response")
		}		

		st := strmark[:x]
		sb := strmark[x+3:]

		it, err := strconv.Atoi(st)

		if err != nil {
			return Task{}, err
		}

		ib, err := strconv.Atoi(sb)

		if err != nil {
			return Task{}, err
		}

		percent := float64(it)/float64(ib)*100

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
			return Task{}, errors.New("daymap: invalid task HTML response")
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
			return Task{}, errors.New("daymap: invalid task HTML response")
		}

		rlhtml := b[:i]
		b = b[i:]
		x := strings.Index(rlhtml, "DMU.OpenAttachment(")

		for x != -1 {
			x += len("DMU.OpenAttachment(")
			rlhtml = rlhtml[x:]
			x = strings.Index(rlhtml, ")")

			if x == -1 {
				return Task{}, errors.New("daymap: invalid task HTML response")
			}

			rlid := rlhtml[:x]
			rlhtml = rlhtml[x:]
			link := "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + rlid
			x = strings.Index(rlhtml, "&nbsp;")

			if x == -1 {
				return Task{}, errors.New("daymap: invalid task HTML response")
			}

			x += len("&nbsp;")
			rlhtml = rlhtml[x:]
			x = strings.Index(rlhtml, "</a>")

			if x == -1 {
				return Task{}, errors.New("daymap: invalid task HTML response")
			}

			name := rlhtml[:x]
			rlhtml = rlhtml[x:]
			task.Reslinks = append(task.Reslinks, [2]string{link, name})
			x = strings.Index(rlhtml, "DMU.OpenAttachment(")
		}
	}

	i = strings.Index(b, `class='WhiteBox' style='padding:5px;margin:2px'>`)

	if i != -1 {
		b = b[i:]
		i = len(`class='WhiteBox' style='padding:5px;margin:2px'>`)
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return Task{}, errors.New("daymap: invalid task HTML response")
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

	mimediv := "--MultipartMimeHtmlFormBoundaryPiFa8ZSp8tLEoC81"
	mimehead := `Content-Disposition: form-data; name="file"; filename="`
	mimehead += filename + "\"\nContent-Type: application/octet-stream\n\n"

	uploader := fileUploader{
		MimeDivider: mimediv,
		MimeHeader: mimehead,
		FileReader: *f,
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", uploadUrl, uploader)

	if err != nil {
		return err
	}

	req.Header.Set(
		"Content-Type",
		`multipart/form-data; boundary="` + mimediv + `"`,
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

	rb, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	b := string(rb)
	i := strings.Index(b, "<form")

	if i == -1 {
		return errors.New("daymap: invalid task HTML response")
	}

	b = b[i:]
	i = strings.Index(b, ` action="`)

	if i == -1 {
		return errors.New("daymap: invalid task HTML response")
	}

	b = b[i:]
	i = len(` action="`)
	b = b[i:]
	i = strings.Index(b, `"`)

	if i == -1 {
		return errors.New("daymap: invalid task HTML response")
	}
	
	rwurl := b[:i]
	rwurl = html.UnescapeString(rwurl)
	b = b[i:]
	rwform := url.Values{}
	i = strings.Index(b, "<input ")

	for i != -1 {
		var name, value string
		b = b[i:]
		i = strings.Index(b, ` type=`)

		if i == -1 {
			return errors.New("daymap: invalid task HTML response")
		}

		b = b[i:]
		i = len(` type=`)
		b = b[i:]
		i = strings.Index(b, ` `)

		if i == -1 {
			return errors.New("daymap: invalid task HTML response")
		}

		inptype := b[:i]
		b = b[i:]
		i = strings.Index(b, `name="`)

		if i == -1 {
			return errors.New("daymap: invalid task HTML response")
		}

		b = b[i:]
		i = len(`name="`)
		b = b[i:]
		i = strings.Index(b, `"`)

		if i == -1 {
			return errors.New("daymap: invalid task HTML response")
		}

		name = b[:i]
		b = b[i:]

		i = strings.Index(b, "\n")

		if i == -1 {
			return errors.New("daymap: invalid task HTML response")
		}

		valtest := b[:i]
		i = strings.Index(valtest, ` value="`)

		if i != -1 {
			b = b[i:]
			i = len(` value="`)
			b = b[i:]
			i = strings.Index(b, `"`)

			if i == -1 {
				panic("6")
				return errors.New("daymap: invalid task HTML response")
			}

			value = b[:i]
			b = b[i:]
		}

		if inptype != "checkbox" {
			rwform.Set(name, value)
			i = strings.Index(b, "<input ")
			continue
		}

		i = strings.Index(b, `<span name=filename>`)

		if i == -1 {
			return errors.New("daymap: invalid task HTML response")
		}

		b = b[i:]
		i = len(`<span name=filename>`)
		b = b[i:]
		i = strings.Index(b, `</span>`)

		if i == -1 {
			return errors.New("daymap: invalid task HTML response")
		}

		fname := b[:i]
		b = b[i:]

		if contains(filenames, fname) {
			rwform.Set(name, "del")
		}

		i = strings.Index(b, "<input ")
	}

	rwform.Set("Cmd", "delete")
	rwform.Set("__EVENTTARGET", "")
	rwform.Set("__EVENTARGUMENT", "")

	rwdata := strings.NewReader(rwform.Encode())
	rwfurl := "https://gihs.daymap.net/daymap/student" + rwurl[1:]
	post, err := http.NewRequest("POST", rwfurl, rwdata)
	fmt.Println(rwform)

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
