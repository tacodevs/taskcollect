package daymap

import (
	"strings"

	"codeberg.org/kvo/std/errors"
)

// Arbitrary strings generation mandatory for making requests to Daymap.
// There is no way around using these, unless you can convince the Daymap
// developers to redesign their software.

var auxClient = []string{
	`ctl00_ctl00_cp_cp_grdAssignments_ctl00_`,
	`ctl03_ctl01_PageSizeComboBox_ClientState`,
}

var auxState = []string{
	`{"logEntries":[],"value":"50","text":"50"`,
	`,"enabled":true,"checkedIndices":[]`,
	`,"checkedItemsTextOverflows":false}`,
}

var auxPage = []string{
	`ctl00$ctl00$cp$cp$grdAssignments`,
	`$ctl00$ctl03$ctl01$PageSizeComboBox`,
}

// Daymap tasks page size.
// Must be extremely high so that all tasks can be retrieved.
// NOTE: Daymap has an (unknown) upper bound to how many tasks can be retrieved.
var auxSize = []string{
	`1000000`,
}

var auxTarget = []string{
	`__EVENTTARGET`,
}

var auxAssignments = []string{
	`ctl00$ctl00$cp$cp$grdAssignments`,
}

var auxArg = []string{
	`__EVENTARGUMENT`,
}

var auxCommand = []string{
	`FireCommand:ctl00$ctl00$cp$cp$grdAssignments$ctl00;PageSize;`,
	auxSize[0],
}

var auxScript = []string{
	`ctl00_ctl00_cp_cp_ScriptManager_TSM`,
}

var auxKey = []string{
	`;;System.Web.Extensions, `,
	`Version=4.0.0.0, `,
	`Culture=neutral, `,
	`PublicKeyToken=31bf3856ad364e35:`,
	`en-AU:9ddf364d-d65d-4f01-a69e-`,
	`8b015049e026:ea597d4b:b25378d2;Telerik.Web.UI, `,
	`Version=2020.1.219.45, `,
	`Culture=neutral, `,
	`PublicKeyToken=121fae78165ba3d4:en-AU:`,
	`bb184598-9004-47ca-9e82-5def416be84b:`,
	`16e4e7cd:33715776:58366029:f7645509:24ee1bba:`,
	`f46195d3:2003d0b8:c128760b:88144a7a:`,
	`1e771326:aa288e2d:258f1c72`,
}

var tasksFormStructs = [][2][]string{
	{auxClient, auxState},
	{auxPage, auxSize},
	{auxTarget, auxAssignments},
	{auxArg, auxCommand},
	{auxScript, auxKey},
}

var tasksFormValues = parseAux(tasksFormStructs)

// Convert a map of []string structures to a map of strings. Used to convert
// easier-to-read []string structures to long dwindly strings that might be
// necessary for DayMap data fetching.
func parseAux(auxStructs [][2][]string) map[string]string {
	auxValues := map[string]string{}

	for _, array := range auxStructs {
		var key, value string

		for _, s := range array[0] {
			key += s
		}

		for _, s := range array[1] {
			value += s
		}

		auxValues[key] = value
	}

	return auxValues
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
func (page *Page) Advance(bound string) errors.Error {
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
func (page Page) UpTo(bound string) (string, errors.Error) {
	i := strings.Index(string(page), bound)
	if i == -1 {
		return "", errors.New("can't find substring", nil)
	}
	return string(page)[:i], nil
}
