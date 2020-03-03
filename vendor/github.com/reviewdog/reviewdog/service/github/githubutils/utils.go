package githubutils

import (
	"fmt"

	"github.com/reviewdog/reviewdog"
)

// LinkedMarkdownCheckResult returns Markdown string which contains a link to the
// location in the CheckResult and CheckResult content itself.
func LinkedMarkdownCheckResult(owner, repo, sha string, c *reviewdog.CheckResult) string {
	if c.Path == "" {
		return c.Message
	}
	loc := BasicLocationFormat(c)
	link := PathLink(owner, repo, sha, c.Path, c.Lnum)
	return fmt.Sprintf("[%s](%s) %s", loc, link, c.Message)
}

// PathLink build a link to GitHub path to given sha, file, and line.
func PathLink(owner, repo, sha, path string, line int) string {
	if sha == "" {
		sha = "master"
	}
	fragment := ""
	if line > 0 {
		fragment = fmt.Sprintf("#L%d", line)
	}
	return fmt.Sprintf("http://github.com/%s/%s/blob/%s/%s%s",
		owner, repo, sha, path, fragment)
}

// BasicLocationFormat format check CheckResult to %f|%l col %c| errorformat.
func BasicLocationFormat(c *reviewdog.CheckResult) string {
	loc := c.Path + "|"
	if c.Lnum != 0 {
		loc = fmt.Sprintf("%s%d", loc, c.Lnum)
		if c.Col != 0 {
			loc = fmt.Sprintf("%s col %d", loc, c.Col)
		}
	}
	return loc + "|"
}
