package giteego

import (
	"fmt"
	"net/url"
	"strings"
)

// OpenURL builds the gitee-go open URL for a repo on the Gitee platform:
//
//	{host}/{owner}/{repo}/gitee_go/open?qt=path
//
// e.g. https://gitee.com/mr-chenguang/test-gitee-go/gitee_go/open?qt=path
//
// platformBaseURL is a gitee platform base URL (e.g. https://gitee.com/api/v5);
// only its scheme and host are used. Opening is intentionally left to the
// caller/AI workflow: this helper only exposes the target URL while the open
// endpoint semantics are being validated.
func OpenURL(platformBaseURL, owner, repo string) (string, error) {
	base, err := url.Parse(platformBaseURL)
	if err != nil {
		return "", fmt.Errorf("giteego: invalid platform base URL: %w", err)
	}
	host := base.Scheme + "://" + base.Host
	if strings.Trim(owner, "/") == "" || strings.Trim(repo, "/") == "" {
		return "", fmt.Errorf("owner and repo are required")
	}
	return fmt.Sprintf("%s/%s/%s/gitee_go/open?qt=path", host,
		url.PathEscape(owner), url.PathEscape(repo)), nil
}
