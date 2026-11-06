package sqlite

import "time"

func parseSQLiteTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}
