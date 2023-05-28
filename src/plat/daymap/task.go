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
	"strings"
	"time"

	"codeberg.org/kvo/std"
	"codeberg.org/kvo/std/errors"

	"main/plat"
)

// Return information about a DayMap task by its ID.
func GetTask(creds User, id string) (plat.Task, errors.Error) {
	taskUrl := "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + id

	task := plat.Task{
		Link:     taskUrl,
		Platform: "daymap",
		Id:       id,
	}

	client := &http.Client{}

	req, e := http.NewRequest("GET", taskUrl, nil)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return plat.Task{}, errors.New("GET request failed", err)
	}

	req.Header.Set("Cookie", creds.Token)

	resp, e := client.Do(req)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return plat.Task{}, errors.New("failed to get resp", err)
	}

	respBody, e := io.ReadAll(resp.Body)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return plat.Task{}, errors.New("failed to read resp.Body", err)
	}

	b := string(respBody)

	if strings.Contains(b, "My&nbsp;Work") || strings.Contains(b, "My Work</div>") {
		task.Upload = true
	}

	i := strings.Index(b, "ctl00_ctl00_cp_cp_divResults")

	if i == -1 {
		return plat.Task{}, plat.ErrInvalidTaskResp.Here()
	}

	b = b[i:]
	i = strings.Index(b, "SectionHeader")

	if i == -1 {
		return plat.Task{}, plat.ErrInvalidTaskResp.Here()
	}

	b = b[i:]
	i = len("SectionHeader") + 2
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return plat.Task{}, plat.ErrInvalidTaskResp.Here()
	}

	task.Name = b[:i]
	b = b[i:]
	i = strings.Index(b, "<div style='padding:6px'>")

	if i == -1 {
		return plat.Task{}, plat.ErrInvalidTaskResp.Here()
	}

	b = b[i:]
	i = len("<div style='padding:6px'>")
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return plat.Task{}, plat.ErrInvalidTaskResp.Here()
	}

	task.Class = b[:i]
	b = b[i:]
	i = strings.Index(b, "<div style='padding:6px'>")

	if i == -1 {
		return plat.Task{}, plat.ErrInvalidTaskResp.Here()
	}

	b = b[i:]
	i = len("<div style='padding:6px'>")
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return plat.Task{}, plat.ErrInvalidTaskResp.Here()
	}

	b = b[i:]
	i = strings.Index(b, "Due on ")

	if i != -1 {
		b = b[i:]
		i = len("Due on ")
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return plat.Task{}, plat.ErrInvalidTaskResp.Here()
		}

		dueStr := b[:i]
		b = b[i:]

		if !strings.Contains(dueStr, ":") {
			task.Due, e = time.ParseInLocation("2/01/2006", dueStr, creds.Timezone)
		} else {
			task.Due, e = time.ParseInLocation("2/01/2006 3:04 PM", dueStr, creds.Timezone)
		}

		if e != nil {
			err := errors.New(e.Error(), nil)
			return plat.Task{}, errors.New("failed to parse time", err)
		}
	}

	i = strings.Index(b, "My Work</div>")

	if i != -1 {
		b = b[i:]
		i = strings.Index(b, "<div><div>")

		if i == -1 {
			return plat.Task{}, plat.ErrInvalidTaskResp.Here()
		}

		b = b[i:]
		i = strings.Index(b, "</div></div></div></div>")

		if i == -1 {
			return plat.Task{}, plat.ErrInvalidTaskResp.Here()
		}

		wlHtml := b[:i]
		b = b[i:]
		x := strings.Index(wlHtml, `<a href="`)

		for x != -1 {
			x += len(`<a href="`)
			wlHtml = wlHtml[x:]
			x = strings.Index(wlHtml, `"`)

			if x == -1 {
				return plat.Task{}, plat.ErrInvalidTaskResp.Here()
			}

			wll := wlHtml[:x]
			wlHtml = wlHtml[x:]
			link := "https://gihs.daymap.net" + wll
			x = strings.Index(wlHtml, "&nbsp;")

			if x == -1 {
				return plat.Task{}, plat.ErrInvalidTaskResp.Here()
			}

			x += len("&nbsp;")
			wlHtml = wlHtml[x:]
			x = strings.Index(wlHtml, "</a>")

			if x == -1 {
				return plat.Task{}, plat.ErrInvalidTaskResp.Here()
			}

			name := wlHtml[:x]
			wlHtml = wlHtml[x:]
			task.WorkLinks = append(task.WorkLinks, [2]string{link, name})
			x = strings.Index(wlHtml, `<a href="`)
		}
	}

	result, err := findGrade(&b)
	if err != nil {
		return plat.Task{}, err
	}
	task.Graded = result.Exists
	task.Grade = result.Grade
	task.Score = result.Mark

	i = strings.Index(b, `class="WhiteBox">`)

	if i != -1 {
		b = b[i:]
		i = len(`class="WhiteBox">`)
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return plat.Task{}, plat.ErrInvalidTaskResp.Here()
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
			return plat.Task{}, plat.ErrInvalidTaskResp.Here()
		}

		rlHtml := b[:i]
		b = b[i:]
		x := strings.Index(rlHtml, "DMU.OpenAttachment(")

		for x != -1 {
			x += len("DMU.OpenAttachment(")
			rlHtml = rlHtml[x:]
			x = strings.Index(rlHtml, ")")

			if x == -1 {
				return plat.Task{}, plat.ErrInvalidTaskResp.Here()
			}

			rlId := rlHtml[:x]
			rlHtml = rlHtml[x:]
			link := "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + rlId
			x = strings.Index(rlHtml, "&nbsp;")

			if x == -1 {
				return plat.Task{}, plat.ErrInvalidTaskResp.Here()
			}

			x += len("&nbsp;")
			rlHtml = rlHtml[x:]
			x = strings.Index(rlHtml, "</a>")

			if x == -1 {
				return plat.Task{}, plat.ErrInvalidTaskResp.Here()
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
			return plat.Task{}, plat.ErrInvalidTaskResp.Here()
		}

		task.Desc = b[:i]
	}

	task.Submitted = true
	return task, nil
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

// Upload files from an HTTP request as student file submissions for a DayMap task.
func UploadWork(creds User, id string, files *multipart.Reader) errors.Error {
	selectUrl := "https://gihs.daymap.net/daymap/Resources/AttachmentAdd.aspx?t=2&LinkID="
	selectUrl += id
	client := &http.Client{}
	var e error

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

		// Repeat stages 1-3 until the contents of the entire file are sent.
		for i := 0; ; i++ {
			if isLast == 0 {
				buflen, e = io.ReadFull(file, buf)
				if e != nil {
					isLast = 1
				}
				chunk = bytes.NewReader(buf[:buflen])
			}

			// Stage 1: Retrieve a DayMap upload URL.

			s1url := "https://gihs.daymap.net/daymap/dws/uploadazure.ashx"
			utc, e := time.LoadLocation("UTC")
			if e != nil {
				err := errors.New(e.Error(), nil)
				return errors.New(`failed to load timezone "UTC"`, err)
			}
			timestamp := fmt.Sprintf("%d", time.Now().In(utc).UnixMilli())

			s1form := url.Values{}
			s1form.Set("cmd", "UploadSas")
			s1form.Set("taskId", id)
			s1form.Set("bloburi", fmt.Sprintf(blobUrl, blobId, fileExt))
			s1form.Set("_method", "PUT")
			s1form.Set("qqtimestamp", timestamp)

			s1url += "?" + s1form.Encode()
			s1req, e := http.NewRequest("GET", s1url, nil)
			if e != nil {
				err := errors.New(e.Error(), nil)
				return errors.New("GET request failed", err)
			}

			s1req.Header.Set("Accept", "application/json")
			s1req.Header.Set("Cookie", creds.Token)
			s1req.Header.Set("Referer", selectUrl)

			s1resp, e := client.Do(s1req)
			if e != nil {
				err := errors.New(e.Error(), nil)
				return errors.New("failed to get resp", err)
			}

			s1body, e = io.ReadAll(s1resp.Body)
			if e != nil {
				err := errors.New(e.Error(), nil)
				return errors.New("failed to read resp.Body", err)
			}

			if isLast == 2 {
				break
			}

			// Stage 2: Request the creation of a file on the DayMap file upload server.

			block := newBlock(i)
			blocks = append(blocks, block)
			s2url := string(s1body) + `&comp=block&blockid=` + blocks[i]

			s2req, e := http.NewRequest("OPTIONS", s2url, nil)
			if e != nil {
				err := errors.New(e.Error(), nil)
				return errors.New("OPTIONS request failed", err)
			}

			s2req.Header.Set("Accept", "*/*")
			s2req.Header.Set("Access-Control-Request-Method", "PUT")
			s2req.Header.Set("Cookie", creds.Token)
			s2req.Header.Set("Origin", "https://gihs.daymap.net")

			_, e = client.Do(s2req)
			if e != nil {
				err := errors.New(e.Error(), nil)
				return errors.New("failed to get resp", err)
			}

			// Stage 3: Send file contents and metadata to the DayMap file upload server.

			s3req, e := http.NewRequest("PUT", s2url, chunk)
			if e != nil {
				err := errors.New(e.Error(), nil)
				return errors.New("PUT request failed", err)
			}

			s3req.Header.Set("Accept", "*/*")
			s3req.Header.Set("Content-Length", fmt.Sprint(buflen))
			s3req.Header.Set("Origin", "https://gihs.daymap.net")
			s3req.Header.Set("x-ms-blob-type", "BlockBlob")
			s3req.Header.Set("x-ms-meta-LinkID", id)
			s3req.Header.Set("x-ms-meta-qqfilename", fileName)
			s3req.Header.Set("x-ms-meta-t", "2")

			_, e = client.Do(s3req)
			if e != nil {
				err := errors.New(e.Error(), nil)
				return errors.New("failed to get resp", err)
			}

			if isLast == 1 {
				isLast++
			}
		}

		// Stage 4: Send final OPTIONS request to Daymap file upload server.

		s4url := string(s1body) + `&comp=blocklist`
		s4req, e := http.NewRequest("OPTIONS", s4url, nil)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("OPTIONS request failed", err)
		}

		s4req.Header.Set("Accept", "*/*")
		accHeaders := "x-ms-blob-type,x-ms-meta-linkid,x-ms-meta-qqfilename,x-ms-meta-t"
		s4req.Header.Set("Access-Control-Request-Headers", accHeaders)
		s4req.Header.Set("Access-Control-Request-Method", "PUT")
		s4req.Header.Set("Cookie", creds.Token)
		s4req.Header.Set("Origin", "https://gihs.daymap.net")

		_, e = client.Do(s4req)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to get resp", err)
		}

		// Stage 5: Send final PUT request to the Daymap file upload server.

		s5form := `<BlockList>`
		for _, block := range blocks {
			s5form += fmt.Sprintf(`<Latest>%s</Latest>`, block)
		}
		s5form += `</BlockList>`
		s5data := strings.NewReader(s5form)
		s5len := fmt.Sprint(len([]byte(s5form)))

		s5req, e := http.NewRequest("PUT", s4url, s5data)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("PUT request failed", err)
		}

		s5req.Header.Set("Accept", "*/*")
		s5req.Header.Set("Content-Length", s5len)
		s5req.Header.Set("Content-Type", "text/plain")
		s5req.Header.Set("Origin", "https://gihs.daymap.net")
		s5req.Header.Set("x-ms-blob-content-type", "")
		s5req.Header.Set("x-ms-meta-LinkID", id)
		s5req.Header.Set("x-ms-meta-qqfilename", fileName)
		s5req.Header.Set("x-ms-meta-t", "2")

		_, e = client.Do(s5req)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to get resp", err)
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

		s6req, e := http.NewRequest("POST", s6url, s6data)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("POST request failed", err)
		}

		s6req.Header.Set("Accept", "application/json")
		s6req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		s6req.Header.Set("Cookie", creds.Token)
		s6req.Header.Set("Origin", "https://gihs.daymap.net")
		s6req.Header.Set("Referer", selectUrl)
		s6req.Header.Set("X-Requested-With", "XMLHttpRequest")

		s6resp, e := client.Do(s6req)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to get resp", err)
		}

		s6body, e := io.ReadAll(s6resp.Body)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to read resp.Body", err)
		}

		jsonResp := chkJson{}
		e = json.Unmarshal(s6body, &jsonResp)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to unmarshal JSON", err)
		}

		if !jsonResp.Success || jsonResp.Error != "" {
			return errors.New("DayMap returned error", errors.New(jsonResp.Error, nil))
		}

		file, mimeErr = files.NextPart()
	}

	err := errors.New(mimeErr.Error(), nil)
	if mimeErr == io.EOF {
		return nil
	} else {
		return errors.New("failed parsing files from multipart MIME request", err)
	}

	return nil
}

// Remove the specified student file submissions from a DayMap task.
func RemoveWork(creds User, id string, filenames []string) errors.Error {
	removeUrl := "https://gihs.daymap.net/daymap/student/attachments.aspx?Type=1&LinkID="
	removeUrl += id
	client := &http.Client{}

	req, e := http.NewRequest("GET", removeUrl, nil)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return errors.New("GET request failed", err)
	}

	req.Header.Set("Cookie", creds.Token)

	resp, e := client.Do(req)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return errors.New("failed to get resp", err)
	}

	respBody, e := io.ReadAll(resp.Body)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return errors.New("failed to read resp.Body", err)
	}

	b := string(respBody)
	i := strings.Index(b, "<form")

	if i == -1 {
		return plat.ErrInvalidTaskResp.Here()
	}

	b = b[i:]
	i = strings.Index(b, ` action="`)

	if i == -1 {
		return plat.ErrInvalidTaskResp.Here()
	}

	b = b[i:]
	i = len(` action="`)
	b = b[i:]
	i = strings.Index(b, `"`)

	if i == -1 {
		return plat.ErrInvalidTaskResp.Here()
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
			return plat.ErrInvalidTaskResp.Here()
		}

		b = b[i:]
		i = len(` type=`)
		b = b[i:]
		i = strings.Index(b, ` `)

		if i == -1 {
			return plat.ErrInvalidTaskResp.Here()
		}

		inputType := b[:i]
		b = b[i:]
		i = strings.Index(b, `name="`)

		if i == -1 {
			return plat.ErrInvalidTaskResp.Here()
		}

		b = b[i:]
		i = len(`name="`)
		b = b[i:]
		i = strings.Index(b, `"`)

		if i == -1 {
			return plat.ErrInvalidTaskResp.Here()
		}

		name = b[:i]
		b = b[i:]

		i = strings.Index(b, "\n")

		if i == -1 {
			return plat.ErrInvalidTaskResp.Here()
		}

		valTest := b[:i]
		i = strings.Index(valTest, ` value="`)

		if i != -1 {
			b = b[i:]
			i = len(` value="`)
			b = b[i:]
			i = strings.Index(b, `"`)

			if i == -1 {
				return plat.ErrInvalidTaskResp.Here()
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
			return plat.ErrInvalidTaskResp.Here()
		}

		b = b[i:]
		i = len(`<span name=filename>`)
		b = b[i:]
		i = strings.Index(b, `</span>`)

		if i == -1 {
			return plat.ErrInvalidTaskResp.Here()
		}

		fname := b[:i]
		b = b[i:]

		if std.Contains(filenames, fname) {
			rwForm.Set(name, "del")
		}

		i = strings.Index(b, "<input ")
	}

	rwForm.Set("Cmd", "delete")
	rwForm.Set("__EVENTTARGET", "")
	rwForm.Set("__EVENTARGUMENT", "")

	rwData := strings.NewReader(rwForm.Encode())
	if _, err := std.Access([]byte(rwUrl), 1); err != nil {
		return errors.New("invalid task HTML response", err)
	}
	rwfurl := "https://gihs.daymap.net/daymap/student" + rwUrl[1:]
	post, e := http.NewRequest("POST", rwfurl, rwData)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return errors.New("POST request failed", err)
	}

	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.Header.Set("Cookie", creds.Token)

	_, e = client.Do(post)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return errors.New("error returning response body", err)
	}

	return nil
}
