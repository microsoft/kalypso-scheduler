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
	"os"
	"strconv"
	"strings"
	"time"

	"net/url"

	"github.com/google/go-github/v49/github"
	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type GithubRepo interface {
	CreatePR(prBranchName string, content *schedulerv1alpha1.RepoContentType) (*string, error)
}

// implements GithubRepo interface
type githubRepo struct {
	repo        *schedulerv1alpha1.GitOpsRepoSpec
	sourceOwner string
	sourceRepo  string
	client      *github.Client
	ctx         context.Context
}

// validate githubRepo implements GithubRepo interface
var _ GithubRepo = (*githubRepo)(nil)

var (
	authorName    string = "Kalypso Scheduler"
	authorEmail   string = "kalypso.scheduler@email.com"
	commitMessage string = "Kalypso Scheduler commit"
)

func getGitHubClient(ctx context.Context) *github.Client {
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return client
}

// new githubRepo function
func NewGithubRepo(ctx context.Context, repo *schedulerv1alpha1.GitOpsRepoSpec) (GithubRepo, error) {
	//parse url into owner and repo
	sourceOwner, sourceRepo, err := parseRepoURL(repo.Repo)
	if err != nil {
		return nil, err
	}

	return &githubRepo{
		repo:        repo,
		sourceOwner: *sourceOwner,
		sourceRepo:  *sourceRepo,
		client:      getGitHubClient(ctx),
		ctx:         ctx,
	}, nil
}

// implement CreatePR function
func (g *githubRepo) CreatePR(prBranchName string, content *schedulerv1alpha1.RepoContentType) (*string, error) {
	newBranch, err := g.getBranch(g.repo.Branch, prBranchName)
	if err != nil {
		return nil, err
	}

	if newBranch == nil {
		return nil, errors.New("failed to create new branch")
	}

	tree, err := g.getTree(newBranch, content)
	if err != nil {
		return nil, err
	}

	err = g.pushCommit(newBranch, tree)
	if err != nil {
		return nil, err
	}

	//TODO: don't create PR if there is no changes
	pr, err := g.createPR(g.repo.Branch, prBranchName)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// implement parse function
func parseRepoURL(repoUrl string) (owner, repo *string, err error) {
	u, err := url.Parse(repoUrl)
	if err != nil {
		return nil, nil, err
	}
	urlPart := strings.Split(u.Path, "/")
	if len(urlPart) < 3 {
		return nil, nil, errors.New("invalid repo url")

	}

	owner = &urlPart[1]
	repo = &urlPart[2]

	return owner, repo, nil
}

// gets the branch to commit to
func (g *githubRepo) getBranch(baseBranchName, prBranchName string) (ref *github.Reference, err error) {
	if ref, _, err = g.client.Git.GetRef(g.ctx, g.sourceOwner, g.sourceRepo, "refs/heads/"+prBranchName); err == nil {
		return ref, nil
	}

	var baseRef *github.Reference
	if baseRef, _, err = g.client.Git.GetRef(g.ctx, g.sourceOwner, g.sourceRepo, "refs/heads/"+baseBranchName); err != nil {
		return nil, err
	}

	newRef := &github.Reference{Ref: github.String("refs/heads/" + prBranchName), Object: &github.GitObject{SHA: baseRef.Object.SHA}}
	ref, _, err = g.client.Git.CreateRef(g.ctx, g.sourceOwner, g.sourceRepo, newRef)
	return ref, err
}

func (g *githubRepo) getTreeEntry(clusterType, fileName, content string) *github.TreeEntry {
	return &github.TreeEntry{
		Path:    github.String(clusterType + "/" + fileName),
		Type:    github.String("blob"),
		Content: github.String(content),
		Mode:    github.String("100644"),
	}
}

// convert the content of the map into yaml stri g
func (g *githubRepo) getManifestsYaml(manifests []unstructured.Unstructured) (string, error) {
	var manifestsYaml string
	for _, manifest := range manifests {
		manifestYaml, err := yaml.Marshal(manifest.Object)
		if err != nil {
			return "", err
		}
		manifestsYaml += string(manifestYaml)
	}
	return manifestsYaml, nil
}

func (g *githubRepo) getTree(ref *github.Reference, content *schedulerv1alpha1.RepoContentType) (tree *github.Tree, err error) {
	// Create a tree with what to commit.
	entries := []*github.TreeEntry{}

	//iterate through the w
	for k, v := range *content {
		reconcilerManifests, err := g.getManifestsYaml(v.ReconcilerManifests)
		if err != nil {
			return nil, err
		}
		manifestsEntry := g.getTreeEntry(k, "reconciler.yaml", reconcilerManifests)
		entries = append(entries, manifestsEntry)

		namespaceManifests, err := g.getManifestsYaml(v.NamespaceManifests)
		if err != nil {
			return nil, err
		}
		namespaceEntry := g.getTreeEntry(k, "namespace.yaml", namespaceManifests)
		entries = append(entries, namespaceEntry)

		configManifests, err := g.getManifestsYaml(v.ConfigManifests)
		if err != nil {
			return nil, err
		}
		if configManifests != "" {
			configEntry := g.getTreeEntry(k, "config.yaml", configManifests)
			entries = append(entries, configEntry)
		}
	}

	tree, _, err = g.client.Git.CreateTree(g.ctx, g.sourceOwner, g.sourceRepo, *ref.Object.SHA, entries)
	return tree, err
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

func (g *githubRepo) createPR(baseBranchName string, prBranchName string) (*string, error) {

	prSubject := "Update manifests"
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

	prNumber := strconv.Itoa(pr.GetNumber())
	return &prNumber, nil
}
