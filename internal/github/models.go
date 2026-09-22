package github

type Error struct {
	Message string `json:"message"`
}

type Response struct {
	RequestID string
}

type User struct {
	Login       string `json:"login"`
	ID          int64  `json:"id"`
	AvatarURL   string `json:"avatar_url"`
	Name        string `json:"name"`
	Company     string `json:"company"`
	Blog        string `json:"blog"`
	Location    string `json:"location"`
	Email       string `json:"email"`
	Type        string `json:"type"`
	PublicRepos int    `json:"public_repos"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	HTMLURL     string `json:"html_url"`
	CreatedAt   string `json:"created_at"`
}

type Installation struct {
	ID                  int64                `json:"id"`
	Account             *InstallationAccount `json:"account"`
	RepositorySelection string               `json:"repository_selection"`
	AccessTokensURL     string               `json:"access_tokens_url"`
	HTMLURL             string               `json:"html_url"`
	AppID               int64                `json:"app_id"`
	Permissions         map[string]string    `json:"permissions"`
	CreatedAt           string               `json:"created_at"`
	UpdatedAt           string               `json:"updated_at"`
}

type InstallationAccount struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

type Organization struct {
	Login             string `json:"login"`
	ID                int64  `json:"id"`
	AvatarURL         string `json:"avatar_url"`
	Description       string `json:"description"`
	HTMLURL           string `json:"html_url"`
	PublicRepos       int    `json:"public_repos"`
	TotalPrivateRepos int    `json:"total_private_repos"`
}

type Member struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
	Type      string `json:"type"`
}

type Team struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description"`
	Privacy      string `json:"privacy"`
	MembersCount int    `json:"members_count"`
}

type Repository struct {
	ID            int64    `json:"id"`
	Owner         *User    `json:"owner"`
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"`
	Description   string   `json:"description"`
	Private       bool     `json:"private"`
	Archived      bool     `json:"archived"`
	DefaultBranch string   `json:"default_branch"`
	HTMLURL       string   `json:"html_url"`
	CloneURL      string   `json:"clone_url"`
	Fork          bool     `json:"fork"`
	Language      string   `json:"language"`
	Stargazers    int      `json:"stargazers_count"`
	Forks         int      `json:"forks_count"`
	Watchers      int      `json:"watchers_count"`
	Size          int      `json:"size"`
	OpenIssues    int      `json:"open_issues_count"`
	PushedAt      string   `json:"pushed_at"`
	UpdatedAt     string   `json:"updated_at"`
	CreatedAt     string   `json:"created_at"`
	Topics        []string `json:"topics"`
}

type Branch struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
	Protected  bool              `json:"protected"`
	Protection *BranchProtection `json:"-"`
}

type BranchProtection struct {
	Enabled          bool                                        `json:"enabled"`
	RequiredPRReview *struct{ RequiredApprovingReviewCount int } `json:"required_pull_request_reviews"`
	EnforceAdmins    *struct{ Enabled bool }                     `json:"enforce_admins"`
}

type BranchListItem struct {
	Name   string `json:"name"`
	Commit struct {
		SHA    string `json:"sha"`
		Commit struct {
			Author struct {
				Date string `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	} `json:"commit"`
	Protected bool `json:"protected"`
}

type Ref struct {
	Ref    string `json:"ref"`
	NodeID string `json:"node_id"`
	URL    string `json:"url"`
	Object struct {
		Type string `json:"type"`
		SHA  string `json:"sha"`
		URL  string `json:"url"`
	} `json:"object"`
}

type Tag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
}

type GitUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type Commit struct {
	SHA       string               `json:"sha"`
	URL       string               `json:"url"`
	HTMLURL   string               `json:"html_url"`
	Author    *GitUser             `json:"author"`
	Committer *GitUser             `json:"committer"`
	Message   string               `json:"message"`
	Tree      struct{ SHA string } `json:"tree"`
	Verified  bool                 `json:"verification"`
}

type CommitListItem struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author  *GitUser             `json:"author"`
		Message string               `json:"message"`
		Tree    struct{ SHA string } `json:"tree"`
	} `json:"commit"`
	Author *struct {
		Login     string
		AvatarURL string
	} `json:"author"`
	HTMLURL string `json:"html_url"`
}

type CommitDetail struct {
	SHA       string   `json:"sha"`
	HTMLURL   string   `json:"html_url"`
	Author    *GitUser `json:"author"`
	Committer *GitUser `json:"committer"`
	Message   string   `json:"message"`
	Stats     struct {
		Total     int `json:"total"`
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"stats"`
	Files []struct {
		Filename  string `json:"filename"`
		Status    string `json:"status"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
		Changes   int    `json:"changes"`
		Patch     string `json:"patch"`
	} `json:"files"`
}

type CompareResult struct {
	Status       string           `json:"status"`
	AheadBy      int              `json:"ahead_by"`
	BehindBy     int              `json:"behind_by"`
	TotalCommits int              `json:"total_commits"`
	Commits      []CommitListItem `json:"commits"`
	Files        []struct {
		Filename  string `json:"filename"`
		Status    string `json:"status"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
		Changes   int    `json:"changes"`
		Patch     string `json:"patch"`
	} `json:"files"`
}

type Content struct {
	Type        string `json:"type"`
	Encoding    string `json:"encoding"`
	Size        int64  `json:"size"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	SHA         string `json:"sha"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
	Language    string `json:"language"`
	Target      string `json:"target"`
	GitURL      string `json:"git_url"`
}

type CodeSearchResult struct {
	TotalCount int `json:"total_count"`
	Items      []struct {
		Name        string      `json:"name"`
		Path        string      `json:"path"`
		HTMLURL     string      `json:"html_url"`
		Repository  *Repository `json:"repository"`
		TextMatches []struct {
			Fragment string `json:"fragment"`
		} `json:"text_matches"`
	} `json:"items"`
}

type PullRequest struct {
	Number  int64  `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	Merged  bool   `json:"merged"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	User    struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
	Base struct {
		Ref   string      `json:"ref"`
		Sha   string      `json:"sha"`
		Repo  *Repository `json:"repo"`
		Label string      `json:"label"`
	} `json:"base"`
	Head struct {
		Ref   string      `json:"ref"`
		Sha   string      `json:"sha"`
		Repo  *Repository `json:"repo"`
		Label string      `json:"label"`
	} `json:"head"`
	Additions      int    `json:"additions"`
	Deletions      int    `json:"deletions"`
	ChangedFiles   int    `json:"changed_files"`
	Mergeable      *bool  `json:"mergeable"`
	MergeableState string `json:"mergeable_state"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	ClosedAt       string `json:"closed_at"`
	MergedAt       string `json:"merged_at"`
	Labels         []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"labels"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	Commits           int    `json:"commits"`
	AuthorAssociation string `json:"author_association"`
}

type Issue struct {
	Number  int64  `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	User    struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
	Labels []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"labels"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	Comments    int       `json:"comments"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	ClosedAt    string    `json:"closed_at"`
	PullRequest *struct{} `json:"pull_request"`
}

type Comment struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	User    struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Release struct {
	ID              int64  `json:"id"`
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
	Name            string `json:"name"`
	Body            string `json:"body"`
	Draft           bool   `json:"draft"`
	Prerelease      bool   `json:"prerelease"`
	HTMLURL         string `json:"html_url"`
	Author          struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"author"`
	CreatedAt   string `json:"created_at"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		Name        string `json:"name"`
		Size        int64  `json:"size"`
		DownloadURL string `json:"browser_download_url"`
		CreatedAt   string `json:"created_at"`
	} `json:"assets"`
}

type CheckRun struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	HTMLURL     string `json:"html_url"`
	AppName     string `json:"app.name"`
}

type Workflow struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	State     string `json:"state"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type WorkflowRun struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	DisplayTitle string `json:"display_title"`
	RunNumber    int    `json:"run_number"`
	WorkflowID   int64  `json:"workflow_id"`
	HeadBranch   string `json:"head_branch"`
	HeadSHA      string `json:"head_sha"`
	Event        string `json:"event"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	HTMLURL      string `json:"html_url"`
	RunAttempt   int    `json:"run_attempt"`
	Actor        *struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"actor"`
	TriggeringActor *struct {
		Login string `json:"login"`
	} `json:"triggering_actor"`
	JobsURL string `json:"jobs_url"`
}

type WorkflowUsage struct {
	Billable struct {
		UBUNTU struct {
			TotalMS int64 `json:"total_ms"`
		} `json:"UBUNTU"`
		MACOS struct {
			TotalMS int64 `json:"total_ms"`
		} `json:"MACOS"`
		WINDOWS struct {
			TotalMS int64 `json:"total_ms"`
		} `json:"WINDOWS"`
	} `json:"billable"`
}

type WorkflowJob struct {
	ID          int64  `json:"id"`
	RunID       int64  `json:"run_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Steps       []struct {
		Name        string `json:"name"`
		Status      string `json:"status"`
		Conclusion  string `json:"conclusion"`
		Number      int    `json:"number"`
		StartedAt   string `json:"started_at"`
		CompletedAt string `json:"completed_at"`
	} `json:"steps"`
	HeadSHA    string `json:"head_sha"`
	RunnerName string `json:"runner_name"`
}

type Artifact struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	SizeInBytes int64  `json:"size_in_bytes"`
	Expired     bool   `json:"expired"`
	CreatedAt   string `json:"created_at"`
	ExpiresAt   string `json:"expires_at"`
	DownloadURL string `json:"archive_download_url"`
}

type Deployment struct {
	ID          int64  `json:"id"`
	Environment string `json:"environment"`
	State       string `json:"state"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Creator     *struct {
		Login string `json:"login"`
	} `json:"creator"`
	SHA                   string `json:"sha"`
	Ref                   string `json:"ref"`
	Task                  string `json:"task"`
	StatusesURL           string `json:"statuses_url"`
	ProductionEnvironment bool   `json:"production_environment"`
}

type DeploymentStatus struct {
	ID          int64  `json:"id"`
	State       string `json:"state"`
	Description string `json:"description"`
	Environment string `json:"environment"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	TargetURL   string `json:"target_url"`
	Creator     *struct {
		Login string `json:"login"`
	} `json:"creator"`
}

type PageList[T any] struct {
	Items []T
	Next  bool
	Total int
}

type ListOptions struct {
	Page    int
	PerPage int
}

func (o *ListOptions) normalize() {
	if o.PerPage <= 0 {
		o.PerPage = 20
	}
	if o.PerPage > 100 {
		o.PerPage = 100
	}
	if o.Page <= 0 {
		o.Page = 1
	}
}
