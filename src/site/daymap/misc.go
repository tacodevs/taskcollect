package daymap

import (
	"strings"

	"git.sr.ht/~kvo/go-std/errors"
)

// Daymap tasks page size.
// Must be extremely high so that all tasks can be retrieved.
// NOTE: Daymap has an (unknown) upper bound to how many tasks can be retrieved.
const auxSize = `1000000`

// In case you are wondering: yes we need these strings, and no we cannot get rid of them.
// Please follow up any mental wellbeing concerns with the Daymap developers.

const (
	auxClient      = `ctl00_ctl00_cp_cp_grdAssignments_ctl00_ctl03_ctl01_PageSizeComboBox_ClientState`
	auxState       = `{"logEntries":[],"value":"50","text":"50","enabled":true,"checkedIndices":[],"checkedItemsTextOverflows":false}`
	auxPage        = `ctl00$ctl00$cp$cp$grdAssignments$ctl00$ctl03$ctl01$PageSizeComboBox`
	auxTarget      = `__EVENTTARGET`
	auxAssignments = `ctl00$ctl00$cp$cp$grdAssignments`
	auxArg         = `__EVENTARGUMENT`
	auxCommand     = `FireCommand:ctl00$ctl00$cp$cp$grdAssignments$ctl00;PageSize;` + auxSize
	auxScript      = `ctl00_ctl00_cp_cp_ScriptManager_TSM`
	auxKey         = `;;System.Web.Extensions, Version=4.0.0.0, Culture=neutral, PublicKeyToken=31bf3856ad364e35:en-AU:9ddf364d-d65d-4f01-a69e-8b015049e026:ea597d4b:b25378d2;Telerik.Web.UI, Version=2020.1.219.45, Culture=neutral, PublicKeyToken=121fae78165ba3d4:en-AU:bb184598-9004-47ca-9e82-5def416be84b:16e4e7cd:33715776:58366029:f7645509:24ee1bba:f46195d3:2003d0b8:c128760b:88144a7a:1e771326:aa288e2d:258f1c72`
)

var tasksFormValues = map[string]string{
	auxClient: auxState,
	auxPage:   auxSize,
	auxTarget: auxAssignments,
	auxArg:    auxCommand,
	auxScript: auxKey,
}

// Helper types and functions for substring extraction.

type Page string

type Extractor interface {
	Advance(bound string) error
	UpTo(bound string) (string, error)
}

// Advance advances page past the substring bound in page, discarding the
// contents of page before the substring bound. If bound does not exist in page,
// an error is returned.
func (page *Page) Advance(bound string) error {
	strPage := string(*page)
	i := strings.Index(strPage, bound)
	if i == -1 {
		return errors.New("can't find substring", nil)
	}
	i += len(bound)
	*page = Page(strPage[i:])
	return nil
}

// UpTo extracts substring sub (which starts from the beginning of page and
// is enclosed by substring bound on the right) from page. If bound does not
// exist in page, an error is returned.
func (page Page) UpTo(bound string) (string, error) {
	i := strings.Index(string(page), bound)
	if i == -1 {
		return "", errors.New("can't find substring", nil)
	}
	return string(page)[:i], nil
}
