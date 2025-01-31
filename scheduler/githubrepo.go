/*
Copyright 2023 microsoft.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"net/url"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v49/github"
	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"golang.org/x/oauth2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	reconcilerName = "reconciler"
	namespaceName  = "namespace"
	configName     = "platform-config"
)

type GithubRepo interface {
	CreatePR(prBranchName string, content *schedulerv1alpha1.RepoContentType) (*string, error)
	UpdateIssue(gitIssueNumber *int, title string, message *string) (*int, error)
}

// implements GithubRepo interface
type githubRepo struct {
	repo        *schedulerv1alpha1.GitOpsRepoSpec
	sourceOwner string
	sourceRepo  string
	client      *github.Client
	ctx         context.Context
	logger      logr.Logger
}

// validate githubRepo implements GithubRepo interface
var _ GithubRepo = (*githubRepo)(nil)

var (
	authorName              string = "Kalypso Scheduler"
	authorEmail             string = "kalypso.scheduler@email.com"
	commitMessage           string = "Kalypso Scheduler commit"
	Promoted_Commit_Id_Path string = ".github/tracking/Promoted_Commit_Id"
	prometedLabel           string = "promoted"
	readmeFilename          string = "README.md"
	readmeContent           string = "This folder contains deployment targets scheduled on the cluster type"
)

func getBaseAPIURL(domainName string) string {
	if domainName == "github.com" {
		return "https://api.github.com/", nil
	}

	// GitHub Enterprise API base URL (e.g., https://github.example.com/api/v3/)
	return fmt.Sprintf("https://%s/api/v3/", domainName)
}

func getGitHubClient(ctx context.Context, domainName string) *github.Client {
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	// Determine API base URL
	baseURL= getBaseAPIURL(domainName)
	client = github.NewClient(tc)
	if baseURL != "https://api.github.com/" {
		client = client.WithEnterpriseURLs(baseURL, baseUrl)
	}
	return client
}

// new githubRepo function
func NewGithubRepo(ctx context.Context, repo *schedulerv1alpha1.GitOpsRepoSpec) (GithubRepo, error) {
	//parse url into owner and repo
	domainName, sourceOwner, sourceRepo, err := parseRepoURL(repo.Repo)
	if err != nil {
		return nil, err
	}

	return &githubRepo{
		repo:        repo,
		sourceOwner: *sourceOwner,
		sourceRepo:  *sourceRepo,
		client:      getGitHubClient(ctx, domainName),
		ctx:         ctx,
		logger:      log.FromContext(ctx),
	}, nil
}

// implement CreatePR function
func (g *githubRepo) CreatePR(prBranchName string, content *schedulerv1alpha1.RepoContentType) (*string, error) {
	baseBranch, err := g.getBaseBranch(g.repo.Branch)
	if err != nil {
		return nil, err
	}

	newBranch, err := g.getBranch(prBranchName, baseBranch)
	if err != nil {
		return nil, err
	}

	if newBranch == nil {
		return nil, errors.New("failed to create new branch")
	}

	tree, isPromoted, err := g.getTree(newBranch, content)
	if err != nil {
		return nil, err
	}

	err = g.pushCommit(newBranch, tree)
	if err != nil {
		return nil, err
	}

	err = g.cleanPullRequests(g.repo.Branch)
	if err != nil {
		return nil, err
	}

	//TODO: don't create PR if there is no changes
	pr, err := g.createPullRequest(g.repo.Branch, prBranchName, isPromoted)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// implement parse function
func parseRepoURL(repoUrl string) (domainName, owner, repo *string, err error) {
	u, err := url.Parse(repoUrl)
	if err != nil {
		return nil, nil, nil, err
	}
	urlPart := strings.Split(u.Path, "/")
	if len(urlPart) < 3 {
		return nil, nil, nil, errors.New("invalid repo url")

	}

	domainName = u.Host
	owner = &urlPart[1]
	repo = &urlPart[2]

	return domainName, owner, repo, nil
}

func (g *githubRepo) getBaseBranch(baseBranchName string) (ref *github.Reference, err error) {

	if ref, _, err = g.client.Git.GetRef(g.ctx, g.sourceOwner, g.sourceRepo, "refs/heads/"+baseBranchName); err != nil {
		return nil, err
	}

	return ref, nil
}

// gets the branch to commit to
func (g *githubRepo) getBranch(prBranchName string, baseRef *github.Reference) (ref *github.Reference, err error) {
	if ref, _, err = g.client.Git.GetRef(g.ctx, g.sourceOwner, g.sourceRepo, "refs/heads/"+prBranchName); err == nil {
		return ref, nil
	}

	newRef := &github.Reference{Ref: github.String("refs/heads/" + prBranchName), Object: &github.GitObject{SHA: baseRef.Object.SHA}}
	ref, _, err = g.client.Git.CreateRef(g.ctx, g.sourceOwner, g.sourceRepo, newRef)
	return ref, err
}

func (g *githubRepo) getTreeEntry(path, fileName, content string) *github.TreeEntry {
	return &github.TreeEntry{
		Path:    github.String(path + "/" + fileName),
		Type:    github.String("blob"),
		Content: github.String(content),
		Mode:    github.String("100644"),
	}
}

// // convert the content of the unstructured slice into yaml string
// func (g *githubRepo) getManifestsYamlUnstructured(manifests []unstructured.Unstructured) (string, error) {
// 	var manifestsYaml string
// 	for _, manifest := range manifests {
// 		manifestYaml, err := yaml.Marshal(manifest.Object)
// 		if err != nil {
// 			return "", err
// 		}
// 		if manifestsYaml != "" {
// 			manifestsYaml += "---\n"
// 		}
// 		manifestsYaml += string(manifestYaml)
// 	}
// 	return manifestsYaml, nil
// }

// convert the content of the string slice into yaml string
func (g *githubRepo) getManifestsYamlString(manifests []string) (string, error) {
	var manifestsYaml string
	for _, manifest := range manifests {
		if manifestsYaml != "" {
			manifestsYaml += "---\n"
		}
		manifestsYaml += manifest
	}
	return manifestsYaml, nil
}

func (g *githubRepo) getTree(ref *github.Reference, content *schedulerv1alpha1.RepoContentType) (tree *github.Tree, isPromoted bool, err error) {
	// Create a tree with what to commit.
	entries := []*github.TreeEntry{}
	existingTree, _, err := g.client.Git.GetTree(g.ctx, g.sourceOwner, g.sourceRepo, *ref.Object.SHA, true)
	if err != nil {
		return nil, false, err
	}

	//iterate through the existing tree and delete the files that are not in the content
	for _, entry := range existingTree.Entries {
		if entry.GetType() == "blob" {
			// get root folder of the entry
			path := strings.Split(entry.GetPath(), "/")
			if len(path) > 1 { // ignore the root folder
				clusterTypeFolder := path[0]
				if !strings.HasPrefix(clusterTypeFolder, ".") { // not something like .github
					clusterTypeContent, ok := content.ClusterTypes[clusterTypeFolder]
					if ok {
						deploymentTargetFolder := path[1]
						if deploymentTargetFolder != readmeFilename {
							if clusterTypeContent.DeploymentTargets != nil {
								_, ok = clusterTypeContent.DeploymentTargets[deploymentTargetFolder]
							} else {
								ok = false
							}
						}

					}
					if !ok {
						// delete the entry
						g.logger.Info("--------------------------deleting file", "path", entry.GetPath())

						entries = append(entries, &github.TreeEntry{
							Path: github.String(entry.GetPath()),
							Mode: github.String("100644"),
						})
					} else {
						g.logger.Info("--------------------------keeping file", "path", entry.GetPath())
					}
				}
			}
		}
	}

	var commitIdEntry *github.TreeEntry
	commitIdEntry, isPromoted, err = g.addPromotedCommitId(existingTree.Entries, content)
	if err != nil {
		return nil, false, err
	}

	if commitIdEntry != nil {
		entries = append(entries, commitIdEntry)
	}

	//iterate through the content and add the files
	for kct, ct := range *&content.ClusterTypes {
		if ct.DeploymentTargets != nil {
			// iterate through the deployment targets
			for kdt, dt := range ct.DeploymentTargets {
				path := kct + "/" + kdt
				var filename string
				reconcilerManifests, err := g.getManifestsYamlString(dt.ReconcilerManifests)
				if err != nil {
					return nil, false, err
				}
				filename = g.getFullManifestsFileName(reconcilerName, dt.ReconcilerManifestsContentType)
				manifestsEntry := g.getTreeEntry(path, filename, reconcilerManifests)
				entries = append(entries, manifestsEntry)

				namespaceManifests, err := g.getManifestsYamlString(dt.NamespaceManifests)
				if err != nil {
					return nil, false, err
				}
				filename = g.getFullManifestsFileName(namespaceName, dt.NamespaceManifestsContentType)
				namespaceEntry := g.getTreeEntry(path, filename, namespaceManifests)
				entries = append(entries, namespaceEntry)

				configManifests, err := g.getManifestsYamlString(dt.ConfigManifests)
				if err != nil {
					return nil, false, err
				}
				if configManifests != "" {
					filename = g.getFullManifestsFileName(configName, dt.ConfigManifestsContentType)
					configEntry := g.getTreeEntry(path, filename, configManifests)
					entries = append(entries, configEntry)
				}

			}
		}
		readmeEntry := g.getTreeEntry(kct, readmeFilename, readmeContent)
		entries = append(entries, readmeEntry)
	}

	tree, _, err = g.client.Git.CreateTree(g.ctx, g.sourceOwner, g.sourceRepo, *ref.Object.SHA, entries)
	return tree, isPromoted, err
}

func (g *githubRepo) getFullManifestsFileName(fileName string, contentType string) string {
	var fileExtension string
	if contentType == schedulerv1alpha1.EnvContentType {
		fileExtension = "sh"
	} else {
		fileExtension = "yaml"
	}

	return fileName + "." + fileExtension
}

func (g *githubRepo) addPromotedCommitId(existingEntries []*github.TreeEntry, content *schedulerv1alpha1.RepoContentType) (commitEntry *github.TreeEntry, isPromoted bool, err error) {
	//get the promoted commit id
	promotedCommitId := content.BaseRepo.Commit
	if promotedCommitId == "" {
		return nil, false, nil
	}

	// iterate over existingEntries and find the promotedCommitId file
	for _, entry := range existingEntries {
		if entry.GetType() == "blob" {
			// if patth ends with Promoted_Commit_Id then that's it
			if entry.GetPath() == Promoted_Commit_Id_Path {
				// if the content is same as the promotedCommitId then return
				// get the content of the file
				blobs, _, err :=
					g.client.Git.GetBlobRaw(context.Background(), g.sourceOwner, g.sourceRepo, entry.GetSHA())
				if err != nil {
					return nil, false, err
				}
				existingPromotedCommitId := string(blobs)

				if existingPromotedCommitId == promotedCommitId {
					return nil, false, nil
				}
				break
			}
		}
	}

	commitEntry = &github.TreeEntry{
		Path:    github.String(Promoted_Commit_Id_Path),
		Type:    github.String("blob"),
		Content: github.String(promotedCommitId),
		Mode:    github.String("100644"),
	}
	return commitEntry, true, nil
}

func (g *githubRepo) pushCommit(ref *github.Reference, tree *github.Tree) (err error) {
	// Get the parent commit to attach the commit to.
	parent, _, err := g.client.Repositories.GetCommit(g.ctx, g.sourceOwner, g.sourceRepo, *ref.Object.SHA, nil)
	if err != nil {
		return err
	}
	// This is not always populated, but is needed.
	parent.Commit.SHA = parent.SHA

	// Create the commit using the tree.
	date := time.Now()
	author := &github.CommitAuthor{Date: &date, Name: &authorName, Email: &authorEmail}
	commit := &github.Commit{Author: author, Message: &commitMessage, Tree: tree, Parents: []*github.Commit{parent.Commit}}
	newCommit, _, err := g.client.Git.CreateCommit(g.ctx, g.sourceOwner, g.sourceRepo, commit)
	if err != nil {
		return err
	}

	// Attach the commit to the branch.
	ref.Object.SHA = newCommit.SHA
	_, _, err = g.client.Git.UpdateRef(g.ctx, g.sourceOwner, g.sourceRepo, ref, false)
	return err
}

// delete existing PRs to the branch as they are no longer valid
func (g *githubRepo) cleanPullRequests(baseBranchName string) error {
	//list esisting pull requests
	prs, _, err := g.client.PullRequests.List(g.ctx, g.sourceOwner, g.sourceRepo, &github.PullRequestListOptions{
		State: "open",
		Base:  baseBranchName,
	})
	if err != nil {
		return err
	}

	for _, pr := range prs {
		g.logger.Info("Deleting Branch", "branch", *pr.Head.Ref)
		_, err := g.client.Git.DeleteRef(g.ctx, g.sourceOwner, g.sourceRepo, "heads/"+*pr.Head.Ref)
		if err != nil {
			return err
		}
		pr.State = github.String("closed")
		g.logger.Info("Closing PR", "pr", *pr.Number)
		_, _, err = g.client.PullRequests.Edit(g.ctx, g.sourceOwner, g.sourceRepo, *pr.Number, pr)
		if err != nil {
			return err
		}
	}

	return nil

}

func (g *githubRepo) createPullRequest(baseBranchName, prBranchName string, isPromoted bool) (*string, error) {

	prSubject := fmt.Sprintf("Update manifests in %s from %s", baseBranchName, prBranchName)
	prDescription := "This PR updates the manifests in GitOps Repo"

	newPR := &github.NewPullRequest{
		Title:               &prSubject,
		Head:                &prBranchName,
		Base:                &baseBranchName,
		Body:                &prDescription,
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := g.client.PullRequests.Create(g.ctx, g.sourceOwner, g.sourceRepo, newPR)
	if err != nil {
		return nil, err
	}

	if isPromoted {
		_, _, err = g.client.Issues.AddLabelsToIssue(g.ctx, g.sourceOwner, g.sourceRepo, pr.GetNumber(), []string{prometedLabel})
		if err != nil {
			return nil, err
		}
	}

	prNumber := strconv.Itoa(pr.GetNumber())
	return &prNumber, nil
}

// implement UpdateIssue function
func (g *githubRepo) UpdateIssue(gitIssueNumber *int, title string, message *string) (*int, error) {

	if message == nil {
		//close issue
		if gitIssueNumber != nil {
			//check if issue exists and delete it
			issue, _, err := g.client.Issues.Get(g.ctx, g.sourceOwner, g.sourceRepo, *gitIssueNumber)
			if err != nil || issue == nil {
				return nil, nil
			}

			issueRequest := &github.IssueRequest{
				State: github.String("closed"),
			}
			_, _, err = g.client.Issues.Edit(g.ctx, g.sourceOwner, g.sourceRepo, *gitIssueNumber, issueRequest)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	}

	issueRequest := &github.IssueRequest{
		Title: github.String(title),
		Body:  message,
	}

	//check if issue exists, if not create it
	if gitIssueNumber == nil || *gitIssueNumber == 0 {
		issue, _, err := g.client.Issues.Create(g.ctx, g.sourceOwner, g.sourceRepo, issueRequest)
		if err != nil {
			return nil, err
		}
		gitIssueNumber = issue.Number
	} else {
		_, _, err := g.client.Issues.Edit(g.ctx, g.sourceOwner, g.sourceRepo, *gitIssueNumber, issueRequest)
		if err != nil {
			return nil, err
		}
	}

	return gitIssueNumber, nil

}
