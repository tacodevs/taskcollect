package example

import (
	"main/site"
)

func Classes(user site.User, c chan site.Pair[[]site.Class, error]) {
	var result site.Pair[[]site.Class, error]
	classes := []site.Class{
		{"Biology", "https://example.com", "example", "346756"},
		{"Chemistry", "https://example.com", "example", "509714"},
		{"Core", "https://example.com", "example", "546610"},
		{"English", "https://example.com", "example", "977234"},
		{"French", "https://example.com", "example", "135986"},
		{"History", "https://example.com", "example", "735492"},
		{"Mathematics", "https://example.com", "example", "672435"},
		{"Physics", "https://example.com", "example", "669267"},
	}
	// If an error occurs, set result.Second to err instead.
	result.First = classes
	c <- result
}
