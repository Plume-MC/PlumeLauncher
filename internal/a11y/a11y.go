package a11y

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	buttonPattern  = regexp.MustCompile(`<button\b[^>]*>`)
	inputPattern   = regexp.MustCompile(`<input\b[^>]*>`)
	selectPattern  = regexp.MustCompile(`<select\b[^>]*>`)
	textareaPattern = regexp.MustCompile(`<textarea\b[^>]*>`)
	imgPattern     = regexp.MustCompile(`<img\b[^>]*>`)

	ariaLabelPattern   = regexp.MustCompile(`aria-label=`)
	ariaLabelByPattern = regexp.MustCompile(`aria-labelledby=`)
	ariaDescribedBy    = regexp.MustCompile(`aria-describedby=`)
	altPattern         = regexp.MustCompile(`\balt=`)
	rolePattern        = regexp.MustCompile(`role=`)
	tabIndexPattern    = regexp.MustCompile(`tabIndex=`)
)

type Violation struct {
	File    string
	Line    int
	Element string
	Message string
}

func hasAriaAttribute(tag string) bool {
	return ariaLabelPattern.MatchString(tag) ||
		ariaLabelByPattern.MatchString(tag) ||
		ariaDescribedBy.MatchString(tag) ||
		rolePattern.MatchString(tag) ||
		tabIndexPattern.MatchString(tag)
}

// ValidateComponentFile checks a single .tsx file for basic a11y patterns.
func ValidateComponentFile(path string) ([]Violation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var violations []Violation
	lineNum := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, m := range buttonPattern.FindAllString(line, -1) {
			if !hasAriaAttribute(m) {
				violations = append(violations, Violation{
					File:    path,
					Line:    lineNum,
					Element: truncate(m, 60),
					Message: "button missing aria-label or role",
				})
			}
		}
		for _, m := range inputPattern.FindAllString(line, -1) {
			if !hasAriaAttribute(m) {
				violations = append(violations, Violation{
					File:    path,
					Line:    lineNum,
					Element: truncate(m, 60),
					Message: "input missing aria-label or aria-labelledby",
				})
			}
		}
		for _, m := range selectPattern.FindAllString(line, -1) {
			if !hasAriaAttribute(m) {
				violations = append(violations, Violation{
					File:    path,
					Line:    lineNum,
					Element: truncate(m, 60),
					Message: "select missing aria-label or aria-labelledby",
				})
			}
		}
		for _, m := range textareaPattern.FindAllString(line, -1) {
			if !hasAriaAttribute(m) {
				violations = append(violations, Violation{
					File:    path,
					Line:    lineNum,
					Element: truncate(m, 60),
					Message: "textarea missing aria-label or aria-labelledby",
				})
			}
		}
		for _, m := range imgPattern.FindAllString(line, -1) {
			if !altPattern.MatchString(m) {
				violations = append(violations, Violation{
					File:    path,
					Line:    lineNum,
					Element: truncate(m, 60),
					Message: "img missing alt attribute",
				})
			}
		}
	}

	return violations, scanner.Err()
}

// ValidateDirectory scans all .tsx files in a directory tree for a11y patterns.
func ValidateDirectory(dir string) ([]Violation, error) {
	var all []Violation
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".tsx") {
			return nil
		}
		violations, err := ValidateComponentFile(path)
		if err != nil {
			return nil
		}
		all = append(all, violations...)
		return nil
	})
	return all, err
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func FormatViolations(violations []Violation) string {
	if len(violations) == 0 {
		return "no violations"
	}
	var sb strings.Builder
	for _, v := range violations {
		fmt.Fprintf(&sb, "%s:%d: %s — %s\n", v.File, v.Line, v.Element, v.Message)
	}
	return sb.String()
}
