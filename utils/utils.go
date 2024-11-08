package utils

import (
	"fmt"
	"time"
)

func ParseTime(at string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05.000Z",
		"2006-01-02 15:04:05.999999",
	}

	for _, format := range formats {
		t, err := time.Parse(format, at)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", at)
}
