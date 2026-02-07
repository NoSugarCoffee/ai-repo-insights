package report

import (
	"regexp"
	"strings"
)

// SanitizeMarkdown sanitizes LLM content to prevent Markdown formatting issues
func SanitizeMarkdown(content string) string {
	// Remove any potential markdown table delimiters that could break formatting
	content = strings.ReplaceAll(content, "|", "\\|")
	
	// Normalize line breaks
	content = strings.ReplaceAll(content, "\r\n", "\n")
	
	// Remove excessive newlines (more than 2 consecutive)
	re := regexp.MustCompile(`\n{3,}`)
	content = re.ReplaceAllString(content, "\n\n")
	
	// Trim leading and trailing whitespace
	content = strings.TrimSpace(content)
	
	return content
}

// SanitizeURL ensures URL is properly formatted
func SanitizeURL(url string) string {
	// Ensure URL starts with https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "https://" + url
	}
	return url
}

// SanitizeRepoName removes any characters that could break markdown links
func SanitizeRepoName(name string) string {
	// Remove brackets and parentheses that could break markdown links
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	name = strings.ReplaceAll(name, "(", "")
	name = strings.ReplaceAll(name, ")", "")
	
	return strings.TrimSpace(name)
}

// SanitizeDescription truncates and cleans repository descriptions
func SanitizeDescription(desc string) string {
	// Remove newlines from descriptions
	desc = strings.ReplaceAll(desc, "\n", " ")
	desc = strings.ReplaceAll(desc, "\r", " ")
	
	// Remove excessive whitespace
	re := regexp.MustCompile(`\s+`)
	desc = re.ReplaceAllString(desc, " ")
	
	// Trim
	desc = strings.TrimSpace(desc)
	
	// Truncate if too long (max 200 chars)
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}
	
	return desc
}
