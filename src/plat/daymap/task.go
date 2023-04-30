package daymap

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"codeberg.org/kvo/std"
	"codeberg.org/kvo/std/errors"

	"main/plat"
)

type chkJson struct {
	Success bool
	Error   string
	FileId  int64
}

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

// Return a string of n random characters.
func randStr(n int) string {
	rand.Seed(time.Now().UnixNano())
	randBytes := make([]byte, n)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)[:n]
}

// Upload files from an HTTP request as student file submissions for a DayMap task.
func UploadWork(creds User, id string, files []plat.File) errors.Error {
	// TODO: Fix issue #68.
	return plat.ErrDaymapUpload.Here()

	selectUrl := "https://gihs.daymap.net/daymap/Resources/AttachmentAdd.aspx?t=2&LinkID="
	selectUrl += id
	client := &http.Client{}

	for _, file := range files {
		fileExt := ""
		dotIndex := strings.LastIndex(file.Name, ".")
		if dotIndex != -1 {
			fileExt = file.Name[dotIndex:]
		}

		// Stage 1: Retrieve a DayMap upload URL.
	
		locUrl := "https://gihs.daymap.net/daymap/dws/uploadazure.ashx"
		blobUrl := "https://glenunga.blob.core.windows.net/daymap/up/%s%s"
		blobId := fmt.Sprintf("%s-%s-%s-%s-%s", randStr(8), randStr(4), randStr(4), randStr(4), randStr(12))
		utc, e := time.LoadLocation("UTC")
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to load timezone 'UTC'", err)
		}
		timestamp := fmt.Sprintf("%d", time.Now().In(utc).UnixMilli())
	
		locForm := url.Values{}
		locForm.Set("cmd", "UploadSas")
		locForm.Set("taskId", id)
		locForm.Set("bloburi", fmt.Sprintf(blobUrl, blobId, fileExt))
		locForm.Set("_method", "PUT")
		locForm.Set("qqtimestamp", timestamp)
	
		locUrl += "?" + locForm.Encode()
		locReq, e := http.NewRequest("GET", locUrl, nil)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("GET request failed", err)
		}
	
		locReq.Header.Set("Accept", "application/json")
		locReq.Header.Set("Cookie", creds.Token)
		locReq.Header.Set("Referer", selectUrl)
	
		locResp, e := client.Do(locReq)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to get resp", err)
		}
	
		locBody, e := io.ReadAll(locResp.Body)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to read resp.Body", err)
		}
	
		// Stage 2: Request the creation of a file on the DayMap file upload server.
	
		optsUrl := string(locBody)
		optsReq, e := http.NewRequest("OPTIONS", optsUrl, nil)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("OPTIONS request failed", err)
		}

		optsReq.Header.Set("Accept", "*/*")
		accHeaders := "x-ms-blob-type,x-ms-meta-linkid,x-ms-meta-qqfilename,x-ms-meta-t"
		optsReq.Header.Set("Access-Control-Request-Headers", accHeaders)
		optsReq.Header.Set("Access-Control-Request-Method", "PUT")
		optsReq.Header.Set("Cookie", creds.Token)
		optsReq.Header.Set("Origin", "https://gihs.daymap.net")
	
		_, e = client.Do(optsReq)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to get resp", err)
		}
	
		// Stage 3: Give the DayMap file upload server information on the file to upload.
	
		putReq, e := http.NewRequest("PUT", optsUrl, nil)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("PUT request failed", err)
		}
	
		putReq.Header.Set("Accept", "*/*")
		putReq.Header.Set("Content-Type", file.MimeType)
		putReq.Header.Set("Origin", "https://gihs.daymap.net")
		putReq.Header.Set("x-ms-blob-type", "BlockBlob")
		putReq.Header.Set("x-ms-meta-LinkID", id)
		putReq.Header.Set("x-ms-meta-qqfilename", file.Name)
		putReq.Header.Set("x-ms-meta-t", "2")
	
		_, e = client.Do(putReq)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to get resp", err)
		}
	
		// Stage 4: Create the file on the DayMap file upload server.
	
		chkForm := url.Values{}
		chkForm.Set("blob", blobId + fileExt)
		chkForm.Set("uuid", blobId)
		chkForm.Set("name", file.Name)
		chkForm.Set("container", "https://glenunga.blob.core.windows.net/daymap/up")
		chkForm.Set("t", "2")
		chkForm.Set("LinkID", id)
	
		chkData := strings.NewReader(chkForm.Encode())
		chkUrl := "https://gihs.daymap.net/daymap/dws/uploadazure.ashx?cmd=UploadSuccess&taskId=" + id
	
		chkReq, e := http.NewRequest("POST", chkUrl, chkData)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("POST request failed", err)
		}
	
		chkReq.Header.Set("Accept", "application/json")
		chkReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		chkReq.Header.Set("Cookie", creds.Token)
		chkReq.Header.Set("Origin", "https://gihs.daymap.net")
		chkReq.Header.Set("Referer", selectUrl)
		chkReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	
		chkResp, e := client.Do(chkReq)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to get resp", err)
		}
	
		chkBody, e := io.ReadAll(chkResp.Body)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to read resp.Body", err)
		}
	
		jsonResp := chkJson{}
		e = json.Unmarshal(chkBody, &jsonResp)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return errors.New("failed to unmarshal JSON", err)
		}
	
		if !jsonResp.Success || jsonResp.Error != "" {
			return errors.New("DayMap returned error", errors.New(jsonResp.Error, nil))
		}
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
