package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

type Service struct {
	auth   *AuthManager
	client *httpClient
}

func NewService(auth *AuthManager) *Service {
	return &Service{
		auth:   auth,
		client: newHTTPClient(),
	}
}

func (s *Service) Auth() *AuthManager { return s.auth }

// ListInstallations lists installations of the GitHub App itself (app JWT, not
// an installation token).
func (s *Service) ListInstallations(ctx context.Context) ([]Installation, error) {
	jwt, err := s.auth.AppJWT()
	if err != nil {
		return nil, err
	}
	req := s.client.request(ctx, "GET", "/app/installations?per_page=100", jwt, nil)
	var out []Installation
	_, err = s.client.do(ctx, req, &out, "")
	return out, err
}

// GetInstallation fetches a single installation by ID (app JWT).
func (s *Service) GetInstallation(ctx context.Context, installationID int64) (*Installation, error) {
	jwt, err := s.auth.AppJWT()
	if err != nil {
		return nil, err
	}
	req := s.client.request(ctx, "GET", "/app/installations/"+strconv.FormatInt(installationID, 10), jwt, nil)
	var out Installation
	_, err = s.client.do(ctx, req, &out, "")
	return &out, err
}

func (s *Service) SetRateLimitCallback(cb func(RateLimitInfo)) {
	s.client.onRate = cb
}

func (s *Service) doAuth(ctx context.Context, method, path string, installationID int64, body any, out any) error {
	token, err := s.auth.InstallationToken(ctx, installationID)
	if err != nil {
		return err
	}
	req := s.client.request(ctx, method, path, token, body)
	resp, err := s.client.do(ctx, req, out, strconv.FormatInt(installationID, 10))
	if err != nil {
		return err
	}
	if resp != nil && resp.StatusCode >= 400 {
		return &apiError{Status: resp.StatusCode, Msg: "github api error"}
	}
	return nil
}

func (s *Service) doList(ctx context.Context, method, path string, installationID int64, out any) error {
	token, err := s.auth.InstallationToken(ctx, installationID)
	if err != nil {
		return err
	}
	req := s.client.request(ctx, method, path, token, nil)
	resp, err := s.client.do(ctx, req, out, strconv.FormatInt(installationID, 10))
	if err != nil {
		return err
	}
	if resp != nil && resp.StatusCode >= 400 {
		return &apiError{Status: resp.StatusCode, Msg: "github api error"}
	}
	return nil
}

func (s *Service) decodeError(resp *http.Response) error {
	return &apiError{Status: resp.StatusCode, Msg: "github api error"}
}

func (s *Service) GetUser(ctx context.Context, installationID int64, login string) (*User, error) {
	login = trimPrefix(login, "@")
	var out User
	err := s.doAuth(ctx, "GET", "/users/"+login, installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) GetAuthenticatedUser(ctx context.Context, installationID int64) (*User, error) {
	var out User
	err := s.doAuth(ctx, "GET", "/user", installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) GetOrganizations(ctx context.Context, installationID int64) ([]Organization, error) {
	var out []Organization
	err := s.doList(ctx, "GET", "/user/organizations?per_page=100", installationID, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) GetRepositories(ctx context.Context, installationID int64, opts ListOptions) ([]Repository, bool, error) {
	opts.normalize()
	var out struct {
		TotalCount int          `json:"total_count"`
		Repos      []Repository `json:"repositories"`
	}
	path := fmt.Sprintf("/installation/repositories?per_page=%d&page=%d", opts.PerPage, opts.Page)
	err := s.doList(ctx, "GET", path, installationID, &out)
	if err != nil {
		return nil, false, err
	}
	return out.Repos, out.TotalCount > opts.PerPage*opts.Page, nil
}

func (s *Service) GetOrgRepositories(ctx context.Context, installationID int64, org string, opts ListOptions) ([]Repository, bool, error) {
	opts.normalize()
	var out []Repository
	path := fmt.Sprintf("/orgs/%s/repos?per_page=%d&page=%d&type=all", org, opts.PerPage, opts.Page)
	err := s.doList(ctx, "GET", path, installationID, &out)
	if err != nil {
		return nil, false, err
	}
	return out, len(out) >= opts.PerPage, nil
}

func (s *Service) GetRepository(ctx context.Context, installationID int64, owner, repo string) (*Repository, error) {
	var out Repository
	err := s.doAuth(ctx, "GET", "/repos/"+owner+"/"+repo, installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) GetContents(ctx context.Context, installationID int64, owner, repo, path, ref string) (*Content, error) {
	var out Content
	reqPath := "/repos/" + owner + "/" + repo + "/contents"
	if path != "" && path != "/" {
		reqPath += "/" + path
	}
	if ref != "" {
		reqPath += "?ref=" + ref
	}
	err := s.doAuth(ctx, "GET", reqPath, installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) GetCommits(ctx context.Context, installationID int64, owner, repo, branch, path string, opts ListOptions) ([]CommitListItem, bool, error) {
	opts.normalize()
	reqPath := "/repos/" + owner + "/" + repo + "/commits?per_page=" + strconv.Itoa(opts.PerPage) + "&page=" + strconv.Itoa(opts.Page)
	if branch != "" {
		reqPath += "&sha=" + branch
	}
	if path != "" {
		reqPath += "&path=" + path
	}
	var out []CommitListItem
	err := s.doList(ctx, "GET", reqPath, installationID, &out)
	if err != nil {
		return nil, false, err
	}
	return out, len(out) >= opts.PerPage, nil
}

func (s *Service) GetCommit(ctx context.Context, installationID int64, owner, repo, sha string) (*CommitDetail, error) {
	var out CommitDetail
	err := s.doAuth(ctx, "GET", "/repos/"+owner+"/"+repo+"/commits/"+sha, installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) CompareRefs(ctx context.Context, installationID int64, owner, repo, base, head string) (*CompareResult, error) {
	var out CompareResult
	err := s.doAuth(ctx, "GET", "/repos/"+owner+"/"+repo+"/compare/"+base+"..."+head, installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) GetBranches(ctx context.Context, installationID int64, owner, repo string, opts ListOptions) ([]BranchListItem, bool, error) {
	opts.normalize()
	var out []BranchListItem
	path := fmt.Sprintf("/repos/%s/%s/branches?per_page=%d&page=%d", owner, repo, opts.PerPage, opts.Page)
	err := s.doList(ctx, "GET", path, installationID, &out)
	if err != nil {
		return nil, false, err
	}
	return out, len(out) >= opts.PerPage, nil
}

func (s *Service) GetBranch(ctx context.Context, installationID int64, owner, repo, branch string) (*Branch, error) {
	var out Branch
	err := s.doAuth(ctx, "GET", "/repos/"+owner+"/"+repo+"/branches/"+branch, installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) CreateBranch(ctx context.Context, installationID int64, owner, repo, name, fromSHA string) error {
	var ref Ref
	err := s.doAuth(ctx, "POST", "/repos/"+owner+"/"+repo+"/git/refs", installationID, map[string]any{
		"ref": "refs/heads/" + name, "sha": fromSHA,
	}, &ref)
	return err
}

func (s *Service) DeleteBranch(ctx context.Context, installationID int64, owner, repo, branch string) error {
	err := s.doAuth(ctx, "DELETE", "/repos/"+owner+"/"+repo+"/git/refs/heads/"+branch, installationID, nil, nil)
	return err
}

func (s *Service) GetTags(ctx context.Context, installationID int64, owner, repo string, opts ListOptions) ([]Tag, bool, error) {
	opts.normalize()
	var out []Tag
	path := fmt.Sprintf("/repos/%s/%s/tags?per_page=%d&page=%d", owner, repo, opts.PerPage, opts.Page)
	err := s.doList(ctx, "GET", path, installationID, &out)
	if err != nil {
		return nil, false, err
	}
	return out, len(out) >= opts.PerPage, nil
}

func (s *Service) CreateTag(ctx context.Context, installationID int64, owner, repo, name, sha, message string) error {
	var ref Ref
	err := s.doAuth(ctx, "POST", "/repos/"+owner+"/"+repo+"/git/refs", installationID, map[string]any{
		"ref": "refs/tags/" + name, "sha": sha,
	}, &ref)
	return err
}

func (s *Service) DeleteTag(ctx context.Context, installationID int64, owner, repo, tag string) error {
	err := s.doAuth(ctx, "DELETE", "/repos/"+owner+"/"+repo+"/git/refs/tags/"+tag, installationID, nil, nil)
	return err
}

func (s *Service) GetPullRequests(ctx context.Context, installationID int64, owner, repo, state string, opts ListOptions) ([]PullRequest, bool, error) {
	opts.normalize()
	path := fmt.Sprintf("/repos/%s/%s/pulls?state=%s&per_page=%d&page=%d", owner, repo, state, opts.PerPage, opts.Page)
	var out []PullRequest
	err := s.doList(ctx, "GET", path, installationID, &out)
	if err != nil {
		return nil, false, err
	}
	return out, len(out) >= opts.PerPage, nil
}

func (s *Service) GetPullRequest(ctx context.Context, installationID int64, owner, repo string, number int64) (*PullRequest, error) {
	var out PullRequest
	err := s.doAuth(ctx, "GET", "/repos/"+owner+"/"+repo+"/pulls/"+strconv.FormatInt(number, 10), installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) MergePullRequest(ctx context.Context, installationID int64, owner, repo string, number int64, mergeMethod string) error {
	var out struct{ Merged bool }
	err := s.doAuth(ctx, "PUT", "/repos/"+owner+"/"+repo+"/pulls/"+strconv.FormatInt(number, 10)+"/merge", installationID, map[string]any{
		"merge_method": mergeMethod,
	}, &out)
	return err
}

func (s *Service) GetPullRequestCommits(ctx context.Context, installationID int64, owner, repo string, number int64) ([]CommitListItem, error) {
	var out []CommitListItem
	err := s.doList(ctx, "GET", "/repos/"+owner+"/"+repo+"/pulls/"+strconv.FormatInt(number, 10)+"/commits?per_page=100", installationID, &out)
	return out, err
}

func (s *Service) GetPullRequestComments(ctx context.Context, installationID int64, owner, repo string, number int64) ([]Comment, error) {
	var out []Comment
	err := s.doList(ctx, "GET", "/repos/"+owner+"/"+repo+"/issues/"+strconv.FormatInt(number, 10)+"/comments?per_page=100", installationID, &out)
	return out, err
}

func (s *Service) CreatePullRequestComment(ctx context.Context, installationID int64, owner, repo string, number int64, body string) error {
	var out Comment
	err := s.doAuth(ctx, "POST", "/repos/"+owner+"/"+repo+"/issues/"+strconv.FormatInt(number, 10)+"/comments", installationID, map[string]any{"body": body}, &out)
	return err
}

func (s *Service) UpdatePullRequest(ctx context.Context, installationID int64, owner, repo string, number int64, state *string) error {
	body := map[string]any{}
	if state != nil {
		body["state"] = *state
	}
	var out PullRequest
	err := s.doAuth(ctx, "PATCH", "/repos/"+owner+"/"+repo+"/pulls/"+strconv.FormatInt(number, 10), installationID, body, &out)
	return err
}

func (s *Service) GetIssues(ctx context.Context, installationID int64, owner, repo, state string, opts ListOptions) ([]Issue, bool, error) {
	opts.normalize()
	path := fmt.Sprintf("/repos/%s/%s/issues?state=%s&per_page=%d&page=%d", owner, repo, state, opts.PerPage, opts.Page)
	var out []Issue
	err := s.doList(ctx, "GET", path, installationID, &out)
	if err != nil {
		return nil, false, err
	}
	return out, len(out) >= opts.PerPage, nil
}

func (s *Service) GetIssue(ctx context.Context, installationID int64, owner, repo string, number int64) (*Issue, error) {
	var out Issue
	err := s.doAuth(ctx, "GET", "/repos/"+owner+"/"+repo+"/issues/"+strconv.FormatInt(number, 10), installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) CreateIssue(ctx context.Context, installationID int64, owner, repo, title, body string) (*Issue, error) {
	var out Issue
	err := s.doAuth(ctx, "POST", "/repos/"+owner+"/"+repo+"/issues", installationID, map[string]any{
		"title": title, "body": body,
	}, &out)
	return &out, err
}

func (s *Service) GetIssueComments(ctx context.Context, installationID int64, owner, repo string, number int64) ([]Comment, error) {
	var out []Comment
	err := s.doList(ctx, "GET", "/repos/"+owner+"/"+repo+"/issues/"+strconv.FormatInt(number, 10)+"/comments?per_page=100", installationID, &out)
	return out, err
}

func (s *Service) CreateIssueComment(ctx context.Context, installationID int64, owner, repo string, number int64, body string) error {
	var out Comment
	err := s.doAuth(ctx, "POST", "/repos/"+owner+"/"+repo+"/issues/"+strconv.FormatInt(number, 10)+"/comments", installationID, map[string]any{"body": body}, &out)
	return err
}

func (s *Service) UpdateIssue(ctx context.Context, installationID int64, owner, repo string, number int64, state *string) error {
	body := map[string]any{}
	if state != nil {
		body["state"] = *state
	}
	var out Issue
	err := s.doAuth(ctx, "PATCH", "/repos/"+owner+"/"+repo+"/issues/"+strconv.FormatInt(number, 10), installationID, body, &out)
	return err
}

func (s *Service) GetReleases(ctx context.Context, installationID int64, owner, repo string, opts ListOptions) ([]Release, bool, error) {
	opts.normalize()
	path := fmt.Sprintf("/repos/%s/%s/releases?per_page=%d&page=%d", owner, repo, opts.PerPage, opts.Page)
	var out []Release
	err := s.doList(ctx, "GET", path, installationID, &out)
	if err != nil {
		return nil, false, err
	}
	return out, len(out) >= opts.PerPage, nil
}

func (s *Service) GetRelease(ctx context.Context, installationID int64, owner, repo string, id int64) (*Release, error) {
	var out Release
	err := s.doAuth(ctx, "GET", "/repos/"+owner+"/"+repo+"/releases/"+strconv.FormatInt(id, 10), installationID, nil, &out)
	return &out, err
}

func (s *Service) GetReleaseByTag(ctx context.Context, installationID int64, owner, repo, tag string) (*Release, error) {
	var out Release
	err := s.doAuth(ctx, "GET", "/repos/"+owner+"/"+repo+"/releases/tags/"+tag, installationID, nil, &out)
	return &out, err
}

func (s *Service) CreateRelease(ctx context.Context, installationID int64, owner, repo, tagName, target, name, body string, draft, prerelease bool) (*Release, error) {
	var out Release
	err := s.doAuth(ctx, "POST", "/repos/"+owner+"/"+repo+"/releases", installationID, map[string]any{
		"tag_name": tagName, "target_commitish": target, "name": name,
		"body": body, "draft": draft, "prerelease": prerelease,
	}, &out)
	return &out, err
}

func (s *Service) UpdateRelease(ctx context.Context, installationID int64, owner, repo string, id int64, body map[string]any) error {
	var out Release
	err := s.doAuth(ctx, "PATCH", "/repos/"+owner+"/"+repo+"/releases/"+strconv.FormatInt(id, 10), installationID, body, &out)
	return err
}

func (s *Service) DeleteRelease(ctx context.Context, installationID int64, owner, repo string, id int64) error {
	err := s.doAuth(ctx, "DELETE", "/repos/"+owner+"/"+repo+"/releases/"+strconv.FormatInt(id, 10), installationID, nil, nil)
	return err
}

func (s *Service) GetWorkflows(ctx context.Context, installationID int64, owner, repo string, opts ListOptions) ([]Workflow, bool, error) {
	opts.normalize()
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows?per_page=%d&page=%d", owner, repo, opts.PerPage, opts.Page)
	var list struct {
		TotalCount int        `json:"total_count"`
		Workflows  []Workflow `json:"workflows"`
	}
	err := s.doList(ctx, "GET", path, installationID, &list)
	if err != nil {
		return nil, false, err
	}
	return list.Workflows, len(list.Workflows) >= opts.PerPage, nil
}

func (s *Service) GetWorkflow(ctx context.Context, installationID int64, owner, repo string, workflowID int64) (*Workflow, error) {
	var out Workflow
	err := s.doAuth(ctx, "GET", "/repos/"+owner+"/"+repo+"/actions/workflows/"+strconv.FormatInt(workflowID, 10), installationID, nil, &out)
	return &out, err
}

func (s *Service) GetWorkflowRuns(ctx context.Context, installationID int64, owner, repo string, workflowID *int64, branch string, opts ListOptions) ([]WorkflowRun, bool, error) {
	opts.normalize()
	var path string
	if workflowID != nil {
		path = fmt.Sprintf("/repos/%s/%s/actions/workflows/%d/runs?per_page=%d&page=%d", owner, repo, *workflowID, opts.PerPage, opts.Page)
	} else {
		path = fmt.Sprintf("/repos/%s/%s/actions/runs?per_page=%d&page=%d", owner, repo, opts.PerPage, opts.Page)
	}
	if branch != "" {
		path += "&branch=" + branch
	}
	var list struct {
		TotalCount   int           `json:"total_count"`
		WorkflowRuns []WorkflowRun `json:"workflow_runs"`
	}
	err := s.doList(ctx, "GET", path, installationID, &list)
	if err != nil {
		return nil, false, err
	}
	return list.WorkflowRuns, len(list.WorkflowRuns) >= opts.PerPage, nil
}

func (s *Service) GetWorkflowRun(ctx context.Context, installationID int64, owner, repo string, runID int64) (*WorkflowRun, error) {
	var out WorkflowRun
	err := s.doAuth(ctx, "GET", "/repos/"+owner+"/"+repo+"/actions/runs/"+strconv.FormatInt(runID, 10), installationID, nil, &out)
	return &out, err
}

func (s *Service) GetWorkflowRunJobs(ctx context.Context, installationID int64, owner, repo string, runID int64) ([]WorkflowJob, error) {
	var list struct {
		TotalCount int           `json:"total_count"`
		Jobs       []WorkflowJob `json:"jobs"`
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/jobs?per_page=100", owner, repo, runID)
	err := s.doList(ctx, "GET", path, installationID, &list)
	return list.Jobs, err
}

func (s *Service) GetJobLogs(ctx context.Context, installationID int64, owner, repo string, jobID int64) ([]byte, error) {
	token, err := s.auth.InstallationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/jobs/%d/logs", owner, repo, jobID)
	req := s.client.request(ctx, "GET", path, token, nil)
	resp, err := s.client.do(ctx, req, nil, strconv.FormatInt(installationID, 10))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readAll(resp.Body)
}

func (s *Service) GetRunLogs(ctx context.Context, installationID int64, owner, repo string, runID int64) ([]byte, error) {
	token, err := s.auth.InstallationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/logs", owner, repo, runID)
	req := s.client.request(ctx, "GET", path, token, nil)
	resp, err := s.client.do(ctx, req, nil, strconv.FormatInt(installationID, 10))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readAll(resp.Body)
}

func (s *Service) RunWorkflow(ctx context.Context, installationID int64, owner, repo string, workflowID int64, ref string, inputs map[string]any) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%d/dispatches", owner, repo, workflowID)
	var out any
	body := map[string]any{"ref": ref}
	if inputs != nil {
		body["inputs"] = inputs
	}
	err := s.doAuth(ctx, "POST", path, installationID, body, &out)
	return err
}

func (s *Service) CancelWorkflow(ctx context.Context, installationID int64, owner, repo string, runID int64) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/cancel", owner, repo, runID)
	var out any
	err := s.doAuth(ctx, "POST", path, installationID, nil, &out)
	return err
}

func (s *Service) RerunWorkflow(ctx context.Context, installationID int64, owner, repo string, runID int64, failed bool) error {
	action := "rerun"
	if failed {
		action = "rerun-failed-jobs"
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/%s", owner, repo, runID, action)
	var out any
	err := s.doAuth(ctx, "POST", path, installationID, map[string]any{}, &out)
	return err
}

func (s *Service) GetArtifacts(ctx context.Context, installationID int64, owner, repo string, runID *int64, opts ListOptions) ([]Artifact, bool, error) {
	opts.normalize()
	var path string
	if runID != nil {
		path = fmt.Sprintf("/repos/%s/%s/actions/runs/%d/artifacts?per_page=%d&page=%d", owner, repo, *runID, opts.PerPage, opts.Page)
	} else {
		path = fmt.Sprintf("/repos/%s/%s/actions/artifacts?per_page=%d&page=%d", owner, repo, opts.PerPage, opts.Page)
	}
	var list struct {
		TotalCount int        `json:"total_count"`
		Artifacts  []Artifact `json:"artifacts"`
	}
	err := s.doList(ctx, "GET", path, installationID, &list)
	return list.Artifacts, len(list.Artifacts) >= opts.PerPage, err
}

func (s *Service) DownloadArtifact(ctx context.Context, installationID int64, owner, repo string, artifactID int64) ([]byte, error) {
	token, err := s.auth.InstallationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/artifacts/%d/zip", owner, repo, artifactID)
	req := s.client.request(ctx, "GET", path, token, nil)
	resp, err := s.client.do(ctx, req, nil, strconv.FormatInt(installationID, 10))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readAll(resp.Body)
}

func (s *Service) GetDeployments(ctx context.Context, installationID int64, owner, repo string, opts ListOptions) ([]Deployment, bool, error) {
	opts.normalize()
	path := fmt.Sprintf("/repos/%s/%s/deployments?per_page=%d&page=%d", owner, repo, opts.PerPage, opts.Page)
	var out []Deployment
	err := s.doList(ctx, "GET", path, installationID, &out)
	if err != nil {
		return nil, false, err
	}
	return out, len(out) >= opts.PerPage, nil
}

func (s *Service) GetDeploymentStatuses(ctx context.Context, installationID int64, owner, repo string, deploymentID int64) ([]DeploymentStatus, error) {
	var out []DeploymentStatus
	path := fmt.Sprintf("/repos/%s/%s/deployments/%d/statuses?per_page=100", owner, repo, deploymentID)
	err := s.doList(ctx, "GET", path, installationID, &out)
	return out, err
}

func (s *Service) SearchCode(ctx context.Context, installationID int64, owner, repo, query string, opts ListOptions) (*CodeSearchResult, error) {
	opts.normalize()
	q := query
	if repo != "" {
		q = "repo:" + repo + " " + query
	}
	path := fmt.Sprintf("/search/code?q=%s&per_page=%d&page=%d", encodeQuery(q), opts.PerPage, opts.Page)
	var out CodeSearchResult
	err := s.doAuth(ctx, "GET", path, installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) SearchRepositories(ctx context.Context, installationID int64, query string, opts ListOptions) ([]Repository, error) {
	opts.normalize()
	path := fmt.Sprintf("/search/repositories?q=%s&per_page=%d&page=%d", encodeQuery(query), opts.PerPage, opts.Page)
	var list struct {
		Items []Repository `json:"items"`
	}
	err := s.doAuth(ctx, "GET", path, installationID, nil, &list)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (s *Service) SearchIssues(ctx context.Context, installationID int64, query string, opts ListOptions) ([]Issue, error) {
	opts.normalize()
	path := fmt.Sprintf("/search/issues?q=%s&per_page=%d&page=%d", encodeQuery(query), opts.PerPage, opts.Page)
	var list struct {
		Items []Issue `json:"items"`
	}
	err := s.doAuth(ctx, "GET", path, installationID, nil, &list)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (s *Service) CreateRepository(ctx context.Context, installationID int64, name, description string, private, init bool) (*Repository, error) {
	var out Repository
	body := map[string]any{
		"name": name, "private": private,
	}
	if description != "" {
		body["description"] = description
	}
	if init {
		body["auto_init"] = true
	}
	err := s.doAuth(ctx, "POST", "/user/repos", installationID, body, &out)
	return &out, err
}

func (s *Service) CreateOrUpdateFile(ctx context.Context, installationID int64, owner, repo, path, message, content, branch, sha string) error {
	var out Content
	body := map[string]any{
		"message": message, "content": content, "branch": branch,
	}
	if sha != "" {
		body["sha"] = sha
	}
	err := s.doAuth(ctx, "PUT", "/repos/"+owner+"/"+repo+"/contents/"+path, installationID, body, &out)
	return err
}

func (s *Service) DeleteFile(ctx context.Context, installationID int64, owner, repo, path, message, sha, branch string) error {
	body := map[string]any{
		"message": message, "sha": sha, "branch": branch,
	}
	var out any
	err := s.doAuth(ctx, "DELETE", "/repos/"+owner+"/"+repo+"/contents/"+path, installationID, body, &out)
	return err
}

func (s *Service) GetOrgMembers(ctx context.Context, installationID int64, org string) ([]Member, error) {
	var out []Member
	err := s.doList(ctx, "GET", "/orgs/"+org+"/members?per_page=100&role=all", installationID, &out)
	return out, err
}

func (s *Service) GetOrgTeams(ctx context.Context, installationID int64, org string, opts ListOptions) ([]Team, bool, error) {
	opts.normalize()
	var out []Team
	path := fmt.Sprintf("/orgs/%s/teams?per_page=%d&page=%d", org, opts.PerPage, opts.Page)
	err := s.doList(ctx, "GET", path, installationID, &out)
	return out, len(out) >= opts.PerPage, err
}

func (s *Service) GetCheckRuns(ctx context.Context, installationID int64, owner, repo string, sha string) ([]CheckRun, error) {
	var list struct {
		TotalCount int        `json:"total_count"`
		CheckRuns  []CheckRun `json:"check_runs"`
	}
	path := "/repos/" + owner + "/" + repo + "/commits/" + sha + "/check-runs?per_page=100"
	err := s.doList(ctx, "GET", path, installationID, &list)
	return list.CheckRuns, err
}

func (s *Service) GetOrg(ctx context.Context, installationID int64, org string) (*Organization, error) {
	var out Organization
	err := s.doAuth(ctx, "GET", "/orgs/"+org, installationID, nil, &out)
	return &out, err
}

func (s *Service) GetInstallationForAccount(ctx context.Context, installationID int64) (*Installation, error) {
	var out Installation
	err := s.doAuth(ctx, "GET", fmt.Sprintf("/app/installations/%d", installationID), installationID, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func readAll(r interface{ Read(p []byte) (int, error) }) ([]byte, error) {
	buf := make([]byte, 0, 8192)
	tmp := make([]byte, 64*1024)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			return buf, nil
		}
	}
}

func trimPrefix(s, prefix string) string {
	for len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		s = s[len(prefix):]
	}
	return s
}

func encodeQuery(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ':
			b = append(b, '%', '2', '0')
		case c == '+':
			b = append(b, '%', '2', 'B')
		case c == '"':
			b = append(b, '%', '2', '2')
		default:
			b = append(b, c)
		}
	}
	return string(b)
}
