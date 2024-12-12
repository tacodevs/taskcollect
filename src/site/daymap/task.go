package daymap

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~kvo/go-std/errors"
	"git.sr.ht/~kvo/go-std/slices"

	"main/site"
)

func Task(user site.User, id string) (site.Task, error) {
	taskUrl := "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + id

	task := site.Task{
		Link:     taskUrl,
		Platform: "daymap",
		Id:       id,
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", taskUrl, nil)
	if err != nil {
		return site.Task{}, errors.New(err, "cannot create task request")
	}

	req.Header.Set("Cookie", user.SiteTokens["daymap"])

	resp, err := client.Do(req)
	if err != nil {
		return site.Task{}, errors.New(err, "cannot execute task request")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return site.Task{}, errors.New(err, "cannot read task response body")
	}

	page := string(body)
	if strings.Contains(page, "My&nbsp;Work") || strings.Contains(page, "My Work</div>") {
		task.Upload = true
	}

	i := strings.Index(page, "ctl00_ctl00_cp_cp_divResults")
	if i == -1 {
		return site.Task{}, errors.New(nil, "invalid task HTML response")
	}

	page = page[i:]
	i = strings.Index(page, "SectionHeader")

	if i == -1 {
		return site.Task{}, errors.New(nil, "invalid task HTML response")
	}

	page = page[i:]
	i = len("SectionHeader") + 2
	page = page[i:]
	i = strings.Index(page, "</div>")

	if i == -1 {
		return site.Task{}, errors.New(nil, "invalid task HTML response")
	}

	task.Name = page[:i]
	page = page[i:]
	i = strings.Index(page, "<div style='padding:6px'>")

	if i == -1 {
		return site.Task{}, errors.New(nil, "invalid task HTML response")
	}

	page = page[i:]
	i = len("<div style='padding:6px'>")
	page = page[i:]
	i = strings.Index(page, "</div>")

	if i == -1 {
		return site.Task{}, errors.New(nil, "invalid task HTML response")
	}

	task.Class = page[:i]
	page = page[i:]
	i = strings.Index(page, "<div style='padding:6px'>")

	if i == -1 {
		return site.Task{}, errors.New(nil, "invalid task HTML response")
	}

	page = page[i:]
	i = len("<div style='padding:6px'>")
	page = page[i:]
	i = strings.Index(page, "</div>")

	if i == -1 {
		return site.Task{}, errors.New(nil, "invalid task HTML response")
	}

	page = page[i:]
	i = strings.Index(page, "Due on ")

	if i != -1 {
		page = page[i:]
		i = len("Due on ")
		page = page[i:]
		i = strings.Index(page, "</div>")

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		dueStr := page[:i]
		page = page[i:]

		if !strings.Contains(dueStr, ":") {
			task.Due, err = time.ParseInLocation("2/01/2006", dueStr, user.Timezone)
		} else {
			task.Due, err = time.ParseInLocation("2/01/2006 3:04 PM", dueStr, user.Timezone)
		}

		if err != nil {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}
	}

	i = strings.Index(page, "My Work</div>")

	if i != -1 {
		page = page[i:]
		i = strings.Index(page, "<div><div>")

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		page = page[i:]
		i = strings.Index(page, "</div></div></div></div>")

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		wlHtml := page[:i]
		page = page[i:]
		j := strings.Index(wlHtml, `<a href="`)

		for j != -1 {
			j += len(`<a href="`)
			wlHtml = wlHtml[j:]
			j = strings.Index(wlHtml, `"`)

			if j == -1 {
				return site.Task{}, errors.New(nil, "invalid task HTML response")
			}

			wlPath := wlHtml[:j]
			wlHtml = wlHtml[j:]
			link := "https://gihs.daymap.net" + wlPath
			j = strings.Index(wlHtml, "&nbsp;")

			if j == -1 {
				return site.Task{}, errors.New(nil, "invalid task HTML response")
			}

			j += len("&nbsp;")
			wlHtml = wlHtml[j:]
			j = strings.Index(wlHtml, "</a>")

			if j == -1 {
				return site.Task{}, errors.New(nil, "invalid task HTML response")
			}

			name := wlHtml[:j]
			wlHtml = wlHtml[j:]
			task.WorkLinks = append(task.WorkLinks, [2]string{link, name})
			j = strings.Index(wlHtml, `<a href="`)
		}
	}

	i = strings.Index(page, "Grade:")

	if i != -1 {
		i = strings.Index(page, "TaskGrade'>")

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		page = page[i:]
		i = len("TaskGrade'>")
		page = page[i:]
		i = strings.Index(page, "</div>")

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		task.Grade = page[:i]
		page = page[i:]
	}

	i = strings.Index(page, "Mark:")

	if i != -1 {
		i = strings.Index(page, "TaskGrade'>")

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		page = page[i:]
		i = len("TaskGrade'>")
		page = page[i:]
		i = strings.Index(page, "</div>")

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		markStr := page[:i]
		page = page[i:]

		i := strings.Index(markStr, " / ")

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		var marks [2]string
		marks[0] = markStr[:i]
		marks[1] = markStr[i+3:]

		top, err := strconv.ParseFloat(marks[0], 64)
		if err != nil {
			return site.Task{}, errors.New(err, "cannot convert %s to float64", marks[0])
		}

		bottom, err := strconv.ParseFloat(marks[1], 64)
		if err != nil {
			return site.Task{}, errors.New(err, "cannot convert %s to float64", marks[1])
		}

		task.Score = top / bottom * 100
	}

	task.Graded = true
	i = strings.Index(page, `class="WhiteBox">`)

	if i != -1 {
		page = page[i:]
		i = len(`class="WhiteBox">`)
		page = page[i:]
		i = strings.Index(page, "</div>")

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		task.Comment = page[:i]
		page = page[i:]
	}

	i = strings.Index(page, "Attachments</div>")

	if i != -1 {
		page = page[i:]
		i = len("Attachments</div>")
		page = page[i:]
		i = strings.Index(page, `class='WhiteBox' style='padding:5px;margin:2px'>`)

		if i == -1 {
			i = strings.Index(page, "\n")
		}

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		rlHtml := page[:i]
		page = page[i:]
		j := strings.Index(rlHtml, "DMU.OpenAttachment(")

		for j != -1 {
			j += len("DMU.OpenAttachment(")
			rlHtml = rlHtml[j:]
			j = strings.Index(rlHtml, ")")

			if j == -1 {
				return site.Task{}, errors.New(nil, "invalid task HTML response")
			}

			rlId := rlHtml[:j]
			rlHtml = rlHtml[j:]
			link := "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + rlId
			j = strings.Index(rlHtml, "&nbsp;")

			if j == -1 {
				return site.Task{}, errors.New(nil, "invalid task HTML response")
			}

			j += len("&nbsp;")
			rlHtml = rlHtml[j:]
			j = strings.Index(rlHtml, "</a>")

			if j == -1 {
				return site.Task{}, errors.New(nil, "invalid task HTML response")
			}

			name := rlHtml[:j]
			rlHtml = rlHtml[j:]
			task.ResLinks = append(task.ResLinks, [2]string{link, name})
			j = strings.Index(rlHtml, "DMU.OpenAttachment(")
		}
	}

	i = strings.Index(page, `class='WhiteBox' style='padding:5px;margin:2px'>`)

	if i != -1 {
		page = page[i:]
		i = len(`class='WhiteBox' style='padding:5px;margin:2px'>`)
		page = page[i:]
		i = strings.Index(page, "</div>")

		if i == -1 {
			return site.Task{}, errors.New(nil, "invalid task HTML response")
		}

		task.Desc = page[:i]
	}

	// Hide submission option as Daymap has no concept of task submission.
	task.Submitted = true
	return task, nil
}

func Submit(user site.User, id string) error {
	return errors.New(nil, "daymap does not support task submission")
}

type chkJson struct {
	Success bool
	Error   string
	FileId  int64
}

// Generate new block ID for Daymap file upload "chunks".
func newBlock(n int) string {
	zeros := "00000"
	s := zeros + fmt.Sprint(n)
	s = s[len(s)-3:]
	b := []byte(s)
	buf := bytes.NewBufferString("")
	encoder := base64.NewEncoder(base64.StdEncoding, buf)
	encoder.Write(b)
	encoder.Close()
	return buf.String()
}

// Return a string of n random characters.
func randStr(n int) string {
	rand.Seed(time.Now().UnixNano())
	randBytes := make([]byte, n)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)[:n]
}

func UploadWork(user site.User, id string, files *multipart.Reader) error {
	selectUrl := "https://gihs.daymap.net/daymap/Resources/AttachmentAdd.aspx?t=2&LinkID="
	selectUrl += id
	client := &http.Client{}

	file, mimeErr := files.NextPart()
	for mimeErr == nil {
		fileName := file.FileName()
		fileExt := ""
		dotIndex := strings.LastIndex(fileName, ".")
		if dotIndex != -1 {
			fileExt = fileName[dotIndex:]
		}
		blobUrl := "https://glenunga.blob.core.windows.net/daymap/up/%s%s"
		blobId := fmt.Sprintf(
			"%s-%s-%s-%s-%s",
			randStr(8), randStr(4),
			randStr(4), randStr(4),
			randStr(12),
		)

		// Daymap file uploads are series of three repeated requests.
		// Each third request carries the next "chunk" of the file.

		// Daymap uses 4 MB chunks, so TaskCollect does the same.
		// NOTE: Ensure that the host has enough memory to support all users!
		buf := make([]byte, 4000000)

		buflen := 0
		isLast := 0
		var blocks []string
		var chunk io.Reader
		var s1body []byte
		var err error

		// Repeat stages 1-3 until the contents of the entire file are sent.
		for i := 0; ; i++ {
			if isLast == 0 {
				buflen, err = io.ReadFull(file, buf)
				if err != nil {
					isLast = 1
				}
				chunk = bytes.NewReader(buf[:buflen])
			}

			// Stage 1: Retrieve a DayMap upload URL.

			s1url := "https://gihs.daymap.net/daymap/dws/uploadazure.ashx"
			timestamp := fmt.Sprintf("%d", time.Now().In(time.UTC).UnixMilli())

			s1form := url.Values{}
			s1form.Set("cmd", "UploadSas")
			s1form.Set("taskId", id)
			s1form.Set("bloburi", fmt.Sprintf(blobUrl, blobId, fileExt))
			s1form.Set("_method", "PUT")
			s1form.Set("qqtimestamp", timestamp)

			s1url += "?" + s1form.Encode()
			s1req, err := http.NewRequest("GET", s1url, nil)
			if err != nil {
				return errors.New(err, "cannot create stage 1 request")
			}

			s1req.Header.Set("Accept", "application/json")
			s1req.Header.Set("Cookie", user.SiteTokens["daymap"])
			s1req.Header.Set("Referer", selectUrl)

			s1, err := client.Do(s1req)
			if err != nil {
				return errors.New(err, "cannot execute stage 1 request")
			}

			s1body, err = io.ReadAll(s1.Body)
			if err != nil {
				return errors.New(err, "cannot read stage 1 body")
			}

			if isLast == 2 {
				break
			}

			// Stage 2: Request the creation of a file on the DayMap file upload server.

			block := newBlock(i)
			blocks = append(blocks, block)
			s2url := string(s1body) + `&comp=block&blockid=` + blocks[i]

			s2req, err := http.NewRequest("OPTIONS", s2url, nil)
			if err != nil {
				return errors.New(err, "cannot create stage 2 request")
			}

			s2req.Header.Set("Accept", "*/*")
			s2req.Header.Set("Access-Control-Request-Method", "PUT")
			s2req.Header.Set("Cookie", user.SiteTokens["daymap"])
			s2req.Header.Set("Origin", "https://gihs.daymap.net")

			_, err = client.Do(s2req)
			if err != nil {
				return errors.New(err, "cannot execute stage 2 request")
			}

			// Stage 3: Send file contents and metadata to the DayMap file upload server.

			s3req, err := http.NewRequest("PUT", s2url, chunk)
			if err != nil {
				return errors.New(err, "cannot create stage 3 request")
			}

			s3req.Header.Set("Accept", "*/*")
			s3req.Header.Set("Content-Length", fmt.Sprint(buflen))
			s3req.Header.Set("Origin", "https://gihs.daymap.net")
			s3req.Header.Set("x-ms-blob-type", "BlockBlob")
			s3req.Header.Set("x-ms-meta-LinkID", id)
			s3req.Header.Set("x-ms-meta-qqfilename", fileName)
			s3req.Header.Set("x-ms-meta-t", "2")

			_, err = client.Do(s3req)
			if err != nil {
				return errors.New(err, "cannot execute stage 3 request")
			}

			if isLast == 1 {
				isLast++
			}
		}

		// Stage 4: Send final OPTIONS request to Daymap file upload server.

		s4url := string(s1body) + `&comp=blocklist`
		s4req, err := http.NewRequest("OPTIONS", s4url, nil)
		if err != nil {
			return errors.New(err, "cannot create stage 4 request")
		}

		s4req.Header.Set("Accept", "*/*")
		accHeaders := "x-ms-blob-type,x-ms-meta-linkid,x-ms-meta-qqfilename,x-ms-meta-t"
		s4req.Header.Set("Access-Control-Request-Headers", accHeaders)
		s4req.Header.Set("Access-Control-Request-Method", "PUT")
		s4req.Header.Set("Cookie", user.SiteTokens["daymap"])
		s4req.Header.Set("Origin", "https://gihs.daymap.net")

		_, err = client.Do(s4req)
		if err != nil {
			return errors.New(err, "cannot execute stage 4 request")
		}

		// Stage 5: Send final PUT request to the Daymap file upload server.

		s5form := `<BlockList>`
		for _, block := range blocks {
			s5form += fmt.Sprintf(`<Latest>%s</Latest>`, block)
		}
		s5form += `</BlockList>`
		s5data := strings.NewReader(s5form)
		s5len := fmt.Sprint(len([]byte(s5form)))

		s5req, err := http.NewRequest("PUT", s4url, s5data)
		if err != nil {
			return errors.New(err, "cannot create stage 5 request")
		}

		s5req.Header.Set("Accept", "*/*")
		s5req.Header.Set("Content-Length", s5len)
		s5req.Header.Set("Content-Type", "text/plain")
		s5req.Header.Set("Origin", "https://gihs.daymap.net")
		s5req.Header.Set("x-ms-blob-content-type", "")
		s5req.Header.Set("x-ms-meta-LinkID", id)
		s5req.Header.Set("x-ms-meta-qqfilename", fileName)
		s5req.Header.Set("x-ms-meta-t", "2")

		_, err = client.Do(s5req)
		if err != nil {
			return errors.New(err, "cannot execute stage 5 request")
		}

		// Stage 6: Send the concluding POST request to the Daymap server.

		s6form := url.Values{}
		s6form.Set("blob", blobId+fileExt)
		s6form.Set("uuid", blobId)
		s6form.Set("name", fileName)
		s6form.Set("container", "https://glenunga.blob.core.windows.net/daymap/up")
		s6form.Set("t", "2")
		s6form.Set("LinkID", id)

		s6data := strings.NewReader(s6form.Encode())
		s6url := "https://gihs.daymap.net/daymap/dws/uploadazure.ashx?cmd=UploadSuccess&taskId=" + id

		s6req, err := http.NewRequest("POST", s6url, s6data)
		if err != nil {
			return errors.New(err, "cannot create stage 6 request")
		}

		s6req.Header.Set("Accept", "application/json")
		s6req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		s6req.Header.Set("Cookie", user.SiteTokens["daymap"])
		s6req.Header.Set("Origin", "https://gihs.daymap.net")
		s6req.Header.Set("Referer", selectUrl)
		s6req.Header.Set("X-Requested-With", "XMLHttpRequest")

		s6, err := client.Do(s6req)
		if err != nil {
			return errors.New(err, "cannot execute stage 6 request")
		}

		s6body, err := io.ReadAll(s6.Body)
		if err != nil {
			return errors.New(err, "cannot read stage 6 body")
		}

		jresp := chkJson{}
		err = json.Unmarshal(s6body, &jresp)
		if err != nil {
			return errors.New(err, "cannot unmarshal JSON")
		}

		if !jresp.Success || jresp.Error != "" {
			return errors.New(errors.New(nil, jresp.Error), "daymap returned error")
		}

		file, mimeErr = files.NextPart()
	}

	err := errors.New(nil, mimeErr.Error())
	if mimeErr == io.EOF {
		return nil
	} else {
		return errors.New(err, "cannot parse multipart MIME")
	}
}

func RemoveWork(user site.User, id string, filenames []string) error {
	removeUrl := "https://gihs.daymap.net/daymap/student/attachments.aspx?Type=1&LinkID="
	removeUrl += id
	client := &http.Client{}

	s1req, err := http.NewRequest("GET", removeUrl, nil)
	if err != nil {
		return errors.New(err, "cannot create stage 1 request")
	}

	s1req.Header.Set("Cookie", user.SiteTokens["daymap"])

	s1, err := client.Do(s1req)
	if err != nil {
		return errors.New(err, "cannot execute stage 1 request")
	}

	s1body, err := io.ReadAll(s1.Body)
	if err != nil {
		return errors.New(err, "cannot read stage 1 body")
	}

	page := string(s1body)
	i := strings.Index(page, "<form")

	if i == -1 {
		return errors.New(nil, "invalid task HTML response")
	}

	page = page[i:]
	i = strings.Index(page, ` action="`)

	if i == -1 {
		return errors.New(nil, "invalid task HTML response")
	}

	page = page[i:]
	i = len(` action="`)
	page = page[i:]
	i = strings.Index(page, `"`)

	if i == -1 {
		return errors.New(nil, "invalid task HTML response")
	}

	rwUrl := page[:i]
	rwUrl = html.UnescapeString(rwUrl)
	page = page[i:]
	s2form := url.Values{}
	i = strings.Index(page, "<input ")

	for i != -1 {
		var name, value string
		page = page[i:]
		i = strings.Index(page, ` type=`)

		if i == -1 {
			return errors.New(nil, "invalid task HTML response")
		}

		page = page[i:]
		i = len(` type=`)
		page = page[i:]
		i = strings.Index(page, ` `)

		if i == -1 {
			return errors.New(nil, "invalid task HTML response")
		}

		inputType := page[:i]
		page = page[i:]
		i = strings.Index(page, `name="`)

		if i == -1 {
			return errors.New(nil, "invalid task HTML response")
		}

		page = page[i:]
		i = len(`name="`)
		page = page[i:]
		i = strings.Index(page, `"`)

		if i == -1 {
			return errors.New(nil, "invalid task HTML response")
		}

		name = page[:i]
		page = page[i:]

		i = strings.Index(page, "\n")

		if i == -1 {
			return errors.New(nil, "invalid task HTML response")
		}

		valTest := page[:i]
		i = strings.Index(valTest, ` value="`)

		if i != -1 {
			page = page[i:]
			i = len(` value="`)
			page = page[i:]
			i = strings.Index(page, `"`)

			if i == -1 {
				return errors.New(nil, "invalid task HTML response")
			}

			value = page[:i]
			page = page[i:]
		}

		if inputType != "checkbox" {
			s2form.Set(name, value)
			i = strings.Index(page, "<input ")
			continue
		}

		i = strings.Index(page, `<span name=filename>`)

		if i == -1 {
			return errors.New(nil, "invalid task HTML response")
		}

		page = page[i:]
		i = len(`<span name=filename>`)
		page = page[i:]
		i = strings.Index(page, `</span>`)

		if i == -1 {
			return errors.New(nil, "invalid task HTML response")
		}

		fname := page[:i]
		page = page[i:]

		if slices.Has(filenames, fname) {
			s2form.Set(name, "del")
		}

		i = strings.Index(page, "<input ")
	}

	s2form.Set("Cmd", "delete")
	s2form.Set("__EVENTTARGET", "")
	s2form.Set("__EVENTARGUMENT", "")

	s2data := strings.NewReader(s2form.Encode())
	if _, err := slices.Get([]byte(rwUrl), 1); err != nil {
		return errors.New(err, "invalid task HTML response")
	}
	s2url := "https://gihs.daymap.net/daymap/student" + rwUrl[1:]
	s2req, err := http.NewRequest("POST", s2url, s2data)
	if err != nil {
		return errors.New(err, "cannot create stage 2 request")
	}

	s2req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s2req.Header.Set("Cookie", user.SiteTokens["daymap"])

	_, err = client.Do(s2req)
	if err != nil {
		return errors.New(err, "cannot execute stage 2 request")
	}

	return nil
}
