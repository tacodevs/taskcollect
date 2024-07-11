package example

import (
	"time"

	"main/plat"
)

func Lessons(user plat.User, start, end time.Time) ([]plat.Lesson, error) {
	lessons := []plat.Lesson{
		{
			Start: start.Add(8*time.Hour + 55*time.Minute),
			End: start.Add(10*time.Hour + 35*time.Minute),
			Class: "Biology",
			Room: "SC03",
		},
		{
			Start: start.Add(11*time.Hour + 00*time.Minute),
			End: start.Add(12*time.Hour + 40*time.Minute),
			Class: "Chemistry",
			Room: "SC02",
		},
		{
			Start: start.Add(13*time.Hour + 25*time.Minute),
			End: start.Add(14*time.Hour + 25*time.Minute),
			Class: "English",
			Room: "EG01",
		},
		{
			Start: start.Add(14*time.Hour + 25*time.Minute),
			End: start.Add(15*time.Hour + 25*time.Minute),
			Class: "French",
			Room: "LA03",
		},
		{
			Start: start.AddDate(0, 0, 1).Add(8*time.Hour + 45*time.Minute),
			End: start.AddDate(0, 0, 1).Add(10*time.Hour + 25*time.Minute),
			Class: "Mathematics",
			Room: "MA01",
		},
		{
			Start: start.AddDate(0, 0, 1).Add(10*time.Hour + 50*time.Minute),
			End: start.AddDate(0, 0, 1).Add(11*time.Hour + 50*time.Minute),
			Class: "Physics",
			Room: "SC01",
		},
		{
			Start: start.AddDate(0, 0, 1).Add(11*time.Hour + 50*time.Minute),
			End: start.AddDate(0, 0, 1).Add(12*time.Hour + 40*time.Minute),
			Class: "Core",
			Room: "CL01",
		},
		{
			Start: start.AddDate(0, 0, 1).Add(13*time.Hour + 25*time.Minute),
			End: start.AddDate(0, 0, 1).Add(14*time.Hour + 25*time.Minute),
			Class: "History",
			Room: "HS01",
		},
		{
			Start: start.AddDate(0, 0, 1).Add(14*time.Hour + 25*time.Minute),
			End: start.AddDate(0, 0, 1).Add(15*time.Hour + 25*time.Minute),
			Class: "Chemistry",
			Room: "SC02",
		},
		{
			Start: start.AddDate(0, 0, 2).Add(9*time.Hour + 50*time.Minute),
			End: start.AddDate(0, 0, 2).Add(11*time.Hour + 5*time.Minute),
			Class: "French",
			Room: "LA03",
		},
		{
			Start: start.AddDate(0, 0, 2).Add(11*time.Hour + 30*time.Minute),
			End: start.AddDate(0, 0, 2).Add(12*time.Hour + 40*time.Minute),
			Class: "English",
			Room: "EG01",
		},
		{
			Start: start.AddDate(0, 0, 2).Add(13*time.Hour + 25*time.Minute),
			End: start.AddDate(0, 0, 2).Add(14*time.Hour + 25*time.Minute),
			Class: "Biology",
			Room: "SC03",
		},
		{
			Start: start.AddDate(0, 0, 2).Add(14*time.Hour + 25*time.Minute),
			End: start.AddDate(0, 0, 2).Add(15*time.Hour + 25*time.Minute),
			Class: "Physics",
			Room: "SC01",
		},
		{
			Start: start.AddDate(0, 0, 3).Add(8*time.Hour + 45*time.Minute),
			End: start.AddDate(0, 0, 3).Add(10*time.Hour + 5*time.Minute),
			Class: "Chemistry",
			Room: "SC02",
		},
		{
			Start: start.AddDate(0, 0, 3).Add(10*time.Hour + 30*time.Minute),
			End: start.AddDate(0, 0, 3).Add(11*time.Hour + 20*time.Minute),
			Class: "Core",
			Room: "CL01",
		},
		{
			Start: start.AddDate(0, 0, 3).Add(11*time.Hour + 20*time.Minute),
			End: start.AddDate(0, 0, 3).Add(12*time.Hour + 40*time.Minute),
			Class: "Physics",
			Room: "SC01",
		},
		{
			Start: start.AddDate(0, 0, 3).Add(13*time.Hour + 25*time.Minute),
			End: start.AddDate(0, 0, 3).Add(14*time.Hour + 25*time.Minute),
			Class: "Mathematics",
			Room: "MA01",
		},
		{
			Start: start.AddDate(0, 0, 3).Add(14*time.Hour + 25*time.Minute),
			End: start.AddDate(0, 0, 3).Add(15*time.Hour + 25*time.Minute),
			Class: "History",
			Room: "HS01",
		},
		{
			Start: start.AddDate(0, 0, 4).Add(8*time.Hour + 55*time.Minute),
			End: start.AddDate(0, 0, 4).Add(10*time.Hour + 15*time.Minute),
			Class: "Mathematics",
			Room: "MA01",
		},
		{
			Start: start.AddDate(0, 0, 4).Add(10*time.Hour + 40*time.Minute),
			End: start.AddDate(0, 0, 4).Add(11*time.Hour + 40*time.Minute),
			Class: "Biology",
			Room: "SC03",
		},
		{
			Start: start.AddDate(0, 0, 4).Add(11*time.Hour + 40*time.Minute),
			End: start.AddDate(0, 0, 4).Add(12*time.Hour + 40*time.Minute),
			Class: "History",
			Room: "HS01",
		},
		{
			Start: start.AddDate(0, 0, 4).Add(13*time.Hour + 25*time.Minute),
			End: start.AddDate(0, 0, 4).Add(14*time.Hour + 25*time.Minute),
			Class: "French",
			Room: "LA03",
		},
		{
			Start: start.AddDate(0, 0, 4).Add(14*time.Hour + 25*time.Minute),
			End: start.AddDate(0, 0, 4).Add(15*time.Hour + 25*time.Minute),
			Class: "English",
			Room: "EG01",
		},
	}
	return lessons, nil
}
