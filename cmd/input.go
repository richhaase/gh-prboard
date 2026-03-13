package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var scanner *bufio.Scanner

func getScanner() *bufio.Scanner {
	if scanner == nil {
		scanner = bufio.NewScanner(os.Stdin)
	}
	return scanner
}

// PromptLine prints the prompt and reads a single line from stdin.
func PromptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	s := getScanner()
	if !s.Scan() {
		if err := s.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("EOF")
	}
	return strings.TrimSpace(s.Text()), nil
}

// ParseNumberSelection parses a selection string like "1-3,5,7" or "all" into
// a sorted, deduplicated slice of ints in [1, max].
func ParseNumberSelection(input string, max int) ([]int, error) {
	input = strings.TrimSpace(input)
	if strings.EqualFold(input, "all") {
		result := make([]int, max)
		for i := range result {
			result[i] = i + 1
		}
		return result, nil
	}

	// Normalize: replace commas and spaces with a single delimiter
	input = strings.ReplaceAll(input, ",", " ")
	tokens := strings.Fields(input)

	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty selection")
	}

	seen := make(map[int]bool)
	for _, token := range tokens {
		if strings.Contains(token, "-") {
			parts := strings.SplitN(token, "-", 2)
			lo, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid number: %q", parts[0])
			}
			hi, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid number: %q", parts[1])
			}
			if lo > hi {
				return nil, fmt.Errorf("invalid range: %s", token)
			}
			for i := lo; i <= hi; i++ {
				if i < 1 || i > max {
					return nil, fmt.Errorf("number %d out of range (1-%d)", i, max)
				}
				seen[i] = true
			}
		} else {
			n, err := strconv.Atoi(token)
			if err != nil {
				return nil, fmt.Errorf("invalid input: %q", token)
			}
			if n < 1 || n > max {
				return nil, fmt.Errorf("number %d out of range (1-%d)", n, max)
			}
			seen[n] = true
		}
	}

	result := make([]int, 0, len(seen))
	for n := range seen {
		result = append(result, n)
	}
	sort.Ints(result)
	return result, nil
}

// ParseSince parses a duration shorthand (1d, 2w, 12h) or ISO date (2026-03-10)
// into a time.Time cutoff point.
func ParseSince(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	// Try duration shorthands: Nd, Nw (not supported by time.ParseDuration)
	if len(s) > 1 {
		suffix := s[len(s)-1]
		numStr := s[:len(s)-1]
		if n, err := strconv.Atoi(numStr); err == nil {
			switch suffix {
			case 'd':
				return time.Now().Add(-time.Duration(n) * 24 * time.Hour), nil
			case 'w':
				return time.Now().Add(-time.Duration(n) * 7 * 24 * time.Hour), nil
			}
		}
	}

	// Try Go duration (e.g., 12h, 30m)
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}

	// Try ISO date
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid --since value %q: use a duration (1d, 2w, 12h) or date (2026-03-10)", s)
}

// FormatRelativeTime returns a human-friendly relative time string.
func FormatRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	case d < 14*24*time.Hour:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	case d < 60*24*time.Hour:
		weeks := int(d.Hours() / 24 / 7)
		return fmt.Sprintf("%dw ago", weeks)
	default:
		months := int(d.Hours() / 24 / 30)
		if months < 1 {
			months = 1
		}
		return fmt.Sprintf("%dmo ago", months)
	}
}
