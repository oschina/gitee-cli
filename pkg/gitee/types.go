package gitee

import "time"

type User struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Remark    string `json:"remark"`
	Type      string `json:"type"`
}

type Namespace struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	HTMLURL string `json:"html_url"`
}

type RepoPermission struct {
	Pull  bool `json:"pull"`
	Push  bool `json:"push"`
	Admin bool `json:"admin"`
}

type Repository struct {
	ID                  int            `json:"id"`
	FullName            string         `json:"full_name"`
	HumanName           string         `json:"human_name"`
	Name                string         `json:"name"`
	Path                string         `json:"path"`
	Description         string         `json:"description"`
	Private             bool           `json:"private"`
	Public              bool           `json:"public"`
	Internal            bool           `json:"internal"`
	Fork                bool           `json:"fork"`
	HTMLURL             string         `json:"html_url"`
	SSHURL              string         `json:"ssh_url"`
	DefaultBranch       string         `json:"default_branch"`
	Language            string         `json:"language"`
	ForksCount          int            `json:"forks_count"`
	StargazersCount     int            `json:"stargazers_count"`
	WatchersCount       int            `json:"watchers_count"`
	OpenIssuesCount     int            `json:"open_issues_count"`
	HasIssues           bool           `json:"has_issues"`
	HasWiki             bool           `json:"has_wiki"`
	HasPage             bool           `json:"has_page"`
	License             string         `json:"license"`
	Owner               User           `json:"owner"`
	Namespace           Namespace      `json:"namespace"`
	Permission          RepoPermission `json:"permission"`
	Relation            string         `json:"relation"`
	PullRequestsEnabled bool           `json:"pull_requests_enabled"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	PushedAt            time.Time      `json:"pushed_at"`
}

type PRUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Remark    string `json:"remark"`
}

type PRAssignee struct {
	ID     int    `json:"id"`
	Login  string `json:"login"`
	Name   string `json:"name"`
	Accept bool   `json:"accept"`
	Remark string `json:"remark"`
}

type BranchRef struct {
	Label string `json:"label"`
	Ref   string `json:"ref"`
	SHA   string `json:"sha"`
}

type PullRequest struct {
	ID        int          `json:"id"`
	Number    int          `json:"number"`
	State     string       `json:"state"`
	Title     string       `json:"title"`
	Body      string       `json:"body"`
	Draft     bool         `json:"draft"`
	Mergeable bool         `json:"mergeable"`
	HTMLURL   string       `json:"html_url"`
	DiffURL   string       `json:"diff_url"`
	PatchURL  string       `json:"patch_url"`
	Head      BranchRef    `json:"head"`
	Base      BranchRef    `json:"base"`
	User      PRUser       `json:"user"`
	Assignees []PRAssignee `json:"assignees"`
	Testers   []PRAssignee `json:"testers"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	MergedAt  *time.Time   `json:"merged_at"`
	ClosedAt  *time.Time   `json:"closed_at"`
}

type Issue struct {
	ID         int        `json:"id"`
	Number     string     `json:"number"`
	State      string     `json:"state"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	HTMLURL    string     `json:"html_url"`
	User       User       `json:"user"`
	Assignee   *User      `json:"assignee"`
	Labels     []Label    `json:"labels"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

type Label struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type SSHKey struct {
	ID  int    `json:"id"`
	Key string `json:"key"`
}

type Comment struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PatchInfo struct {
	Diff        string `json:"diff"`
	NewPath     string `json:"new_path"`
	OldPath     string `json:"old_path"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
	TooLarge    bool   `json:"too_large"`
}

type DiffFile struct {
	SHA       string    `json:"sha"`
	Filename  string    `json:"filename"`
	Status    *string   `json:"status"`
	Additions string    `json:"additions"`
	Deletions string    `json:"deletions"`
	BlobURL   string    `json:"blob_url"`
	RawURL    string    `json:"raw_url"`
	Patch     PatchInfo `json:"patch"`
}
