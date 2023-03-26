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

	"main/errors"
	"main/plat"
)

type chkJson struct {
	Success bool
	Error   string
	FileId  int64
}

// Return information about a DayMap task by its ID.
func GetTask(creds User, id string) (plat.Task, error) {
	taskUrl := "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + id

	task := plat.Task{
		Link:     taskUrl,
		Platform: "daymap",
		Id:       id,
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", taskUrl, nil)
	if err != nil {
		return plat.Task{}, errors.NewError("daymap.GetTask", "GET request failed", err)
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		return plat.Task{}, errors.NewError("daymap.GetTask", "failed to get resp", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return plat.Task{}, errors.NewError("daymap.GetTask", "failed to read resp.Body", err)
	}

	b := string(respBody)

	if strings.Contains(b, "My&nbsp;Work") || strings.Contains(b, "My Work</div>") {
		task.Upload = true
	}

	i := strings.Index(b, "ctl00_ctl00_cp_cp_divResults")

	if i == -1 {
		return plat.Task{}, errInvalidTaskResp
	}

	b = b[i:]
	i = strings.Index(b, "SectionHeader")

	if i == -1 {
		return plat.Task{}, errInvalidTaskResp
	}

	b = b[i:]
	i = len("SectionHeader") + 2
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return plat.Task{}, errInvalidTaskResp
	}

	task.Name = b[:i]
	b = b[i:]
	i = strings.Index(b, "<div style='padding:6px'>")

	if i == -1 {
		return plat.Task{}, errInvalidTaskResp
	}

	b = b[i:]
	i = len("<div style='padding:6px'>")
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return plat.Task{}, errInvalidTaskResp
	}

	task.Class = b[:i]
	b = b[i:]
	i = strings.Index(b, "<div style='padding:6px'>")

	if i == -1 {
		return plat.Task{}, errInvalidTaskResp
	}

	b = b[i:]
	i = len("<div style='padding:6px'>")
	b = b[i:]
	i = strings.Index(b, "</div>")

	if i == -1 {
		return plat.Task{}, errInvalidTaskResp
	}

	b = b[i:]
	i = strings.Index(b, "Due on ")

	if i != -1 {
		b = b[i:]
		i = len("Due on ")
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return plat.Task{}, errInvalidTaskResp
		}

		dueStr := b[:i]
		b = b[i:]

		if !strings.Contains(dueStr, ":") {
			task.Due, err = time.ParseInLocation("2/01/2006", dueStr, creds.Timezone)
		} else {
			task.Due, err = time.ParseInLocation("2/01/2006 3:04 PM", dueStr, creds.Timezone)
		}

		if err != nil {
			return plat.Task{}, errors.NewError("daymap.GetTask", "failed to parse time", err)
		}
	}

	i = strings.Index(b, "My Work</div>")

	if i != -1 {
		b = b[i:]
		i = strings.Index(b, "<div><div>")

		if i == -1 {
			return plat.Task{}, errInvalidTaskResp
		}

		b = b[i:]
		i = strings.Index(b, "</div></div></div></div>")

		if i == -1 {
			return plat.Task{}, errInvalidTaskResp
		}

		wlHtml := b[:i]
		b = b[i:]
		x := strings.Index(wlHtml, `<a href="`)

		for x != -1 {
			x += len(`<a href="`)
			wlHtml = wlHtml[x:]
			x = strings.Index(wlHtml, `"`)

			if x == -1 {
				return plat.Task{}, errInvalidTaskResp
			}

			wll := wlHtml[:x]
			wlHtml = wlHtml[x:]
			link := "https://gihs.daymap.net" + wll
			x = strings.Index(wlHtml, "&nbsp;")

			if x == -1 {
				return plat.Task{}, errInvalidTaskResp
			}

			x += len("&nbsp;")
			wlHtml = wlHtml[x:]
			x = strings.Index(wlHtml, "</a>")

			if x == -1 {
				return plat.Task{}, errInvalidTaskResp
			}

			name := wlHtml[:x]
			wlHtml = wlHtml[x:]
			task.WorkLinks = append(task.WorkLinks, [2]string{link, name})
			x = strings.Index(wlHtml, `<a href="`)
		}
	}

	task.Result, err = findGrade(&b)
	if err != nil {
		return plat.Task{}, err
	}

	i = strings.Index(b, `class="WhiteBox">`)

	if i != -1 {
		b = b[i:]
		i = len(`class="WhiteBox">`)
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			return plat.Task{}, errInvalidTaskResp
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
			return plat.Task{}, errInvalidTaskResp
		}

		rlHtml := b[:i]
		b = b[i:]
		x := strings.Index(rlHtml, "DMU.OpenAttachment(")

		for x != -1 {
			x += len("DMU.OpenAttachment(")
			rlHtml = rlHtml[x:]
			x = strings.Index(rlHtml, ")")

			if x == -1 {
				return plat.Task{}, errInvalidTaskResp
			}

			rlId := rlHtml[:x]
			rlHtml = rlHtml[x:]
			link := "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + rlId
			x = strings.Index(rlHtml, "&nbsp;")

			if x == -1 {
				return plat.Task{}, errInvalidTaskResp
			}

			x += len("&nbsp;")
			rlHtml = rlHtml[x:]
			x = strings.Index(rlHtml, "</a>")

			if x == -1 {
				return plat.Task{}, errInvalidTaskResp
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
			return plat.Task{}, errInvalidTaskResp
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
func UploadWork(creds User, id string, files []plat.File) error {
	// TODO: Fix issue #68.
	return errors.ErrDaymapUpload

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
		utc, err := time.LoadLocation("UTC")
		if err != nil {
			return errors.NewError("daymap.UploadWork", "failed to load timezone 'UTC'", err)
		}
		timestamp := fmt.Sprintf("%d", time.Now().In(utc).UnixMilli())
	
		locForm := url.Values{}
		locForm.Set("cmd", "UploadSas")
		locForm.Set("taskId", id)
		locForm.Set("bloburi", fmt.Sprintf(blobUrl, blobId, fileExt))
		locForm.Set("_method", "PUT")
		locForm.Set("qqtimestamp", timestamp)
	
		locUrl += "?" + locForm.Encode()
		locReq, err := http.NewRequest("GET", locUrl, nil)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "GET request failed", err)
		}
	
		locReq.Header.Set("Accept", "application/json")
		locReq.Header.Set("Cookie", creds.Token)
		locReq.Header.Set("Referer", selectUrl)
	
		locResp, err := client.Do(locReq)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "failed to get resp", err)
		}
	
		locBody, err := io.ReadAll(locResp.Body)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "failed to read resp.Body", err)
		}
	
		// Stage 2: Request the creation of a file on the DayMap file upload server.
	
		optsUrl := string(locBody)
		optsReq, err := http.NewRequest("OPTIONS", optsUrl, nil)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "OPTIONS request failed", err)
		}

		optsReq.Header.Set("Accept", "*/*")
		accHeaders := "x-ms-blob-type,x-ms-meta-linkid,x-ms-meta-qqfilename,x-ms-meta-t"
		optsReq.Header.Set("Access-Control-Request-Headers", accHeaders)
		optsReq.Header.Set("Access-Control-Request-Method", "PUT")
		optsReq.Header.Set("Cookie", creds.Token)
		optsReq.Header.Set("Origin", "https://gihs.daymap.net")
	
		_, err = client.Do(optsReq)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "failed to get resp", err)
		}
	
		// Stage 3: Give the DayMap file upload server information on the file to upload.
	
		putReq, err := http.NewRequest("PUT", optsUrl, nil)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "PUT request failed", err)
		}
	
		putReq.Header.Set("Accept", "*/*")
		putReq.Header.Set("Content-Type", file.MimeType)
		putReq.Header.Set("Origin", "https://gihs.daymap.net")
		putReq.Header.Set("x-ms-blob-type", "BlockBlob")
		putReq.Header.Set("x-ms-meta-LinkID", id)
		putReq.Header.Set("x-ms-meta-qqfilename", file.Name)
		putReq.Header.Set("x-ms-meta-t", "2")
	
		_, err = client.Do(putReq)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "failed to get resp", err)
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
	
		chkReq, err := http.NewRequest("POST", chkUrl, chkData)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "POST request failed", err)
		}
	
		chkReq.Header.Set("Accept", "application/json")
		chkReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		chkReq.Header.Set("Cookie", creds.Token)
		chkReq.Header.Set("Origin", "https://gihs.daymap.net")
		chkReq.Header.Set("Referer", selectUrl)
		chkReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	
		chkResp, err := client.Do(chkReq)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "failed to get resp", err)
		}
	
		chkBody, err := io.ReadAll(chkResp.Body)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "failed to read resp.Body", err)
		}
	
		jsonResp := chkJson{}
		err = json.Unmarshal(chkBody, &jsonResp)
		if err != nil {
			return errors.NewError("daymap.UploadWork", "failed to unmarshal JSON", err)
		}
	
		if !jsonResp.Success || jsonResp.Error != "" {
			return errors.NewError("daymap.UploadWork", "DayMap returned error", errors.New(jsonResp.Error))
		}
	}

	return nil
}

// Remove the specified student file submissions from a DayMap task.
func RemoveWork(creds User, id string, filenames []string) error {
	removeUrl := "https://gihs.daymap.net/daymap/student/attachments.aspx?Type=1&LinkID="
	removeUrl += id
	client := &http.Client{}

	req, err := http.NewRequest("GET", removeUrl, nil)
	if err != nil {
		return errors.NewError("daymap.RemoveWork", "GET request failed", err)
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		return errors.NewError("daymap.RemoveWork", "failed to get resp", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.NewError("daymap.RemoveWork", "failed to read resp.Body", err)
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

		if std.Contains(filenames, fname) {
			rwForm.Set(name, "del")
		}

		i = strings.Index(b, "<input ")
	}

	rwForm.Set("Cmd", "delete")
	rwForm.Set("__EVENTTARGET", "")
	rwForm.Set("__EVENTARGUMENT", "")

	rwData := strings.NewReader(rwForm.Encode())
	rwfurl := "https://gihs.daymap.net/daymap/student" + rwUrl[1:]
	post, err := http.NewRequest("POST", rwfurl, rwData)
	if err != nil {
		return errors.NewError("daymap.RemoveWork", "POST request failed", err)
	}

	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.Header.Set("Cookie", creds.Token)

	_, err = client.Do(post)
	if err != nil {
		return errors.NewError("daymap.RemoveWork", "error returning response body", err)
	}

	return nil
}
