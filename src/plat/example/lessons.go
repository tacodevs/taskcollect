package example

import (
	"time"

	"main/plat"
)

func Lessons(user plat.User, start, end time.Time) ([]plat.Lesson, error) {
	lessons := []plat.Lesson{
		{
			Start: start.Add(8*time.Hour + 55*time.Minute),
			End: start.Add(10 * time.Hour + 35*time.Minute),
			Class: "Biology",
			Room: "SC03",
		},
		{
			Start: start.Add(11*time.Hour + 00*time.Minute),
			End: start.Add(12 * time.Hour + 40*time.Minute),
			Class: "Chemistry",
			Room: "SC02",
		},
		{
			Start: start.Add(13*time.Hour + 25*time.Minute),
			End: start.Add(14 * time.Hour + 25*time.Minute),
			Class: "English",
			Room: "EG01",
		},
		{
			Start: start.Add(14*time.Hour + 25*time.Minute),
			End: start.Add(15 * time.Hour + 25*time.Minute),
			Class: "French",
			Room: "LA03",
		},
		{
			Start: start.AddDate(0, 0, 1).Add(8*time.Hour + 45*time.Minute),
			End: start.AddDate(0, 0, 1).Add(10 * time.Hour + 25*time.Minute),
			Class: "Mathematics",
			Room: "MA01",
		},
	}
	return lessons, nil
}
