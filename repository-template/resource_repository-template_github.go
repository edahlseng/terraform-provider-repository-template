package repositoryTemplate

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/v19/github"
	"github.com/hashicorp/terraform/helper/schema"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"io/ioutil"
	"os"
	"regexp"
	"time"
)

func resourceRepositoryTemplateGitHub() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryTemplateGitHubCreate,
		Read:   resourceRepositoryTemplateGitHubRead,
		Update: resourceRepositoryTemplateGitHubUpdate,
		Delete: resourceRepositoryTemplateGitHubDelete,

		Schema: map[string]*schema.Schema{
			"repository_owner": {
				Type:     schema.TypeString,
				Required: true,
			},
			"repository_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"target_branch": {
				Type:     schema.TypeString,
				Required: true,
			},
			"working_branch": {
				Type:     schema.TypeString,
				Required: true,
			},
			"files": {
				Type:     schema.TypeMap,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func cloneRepository(d *schema.ResourceData, client *Client) (*git.Repository, error) {
	if client.GitHubClient == nil {
		return nil, errors.New("GitHub was not configured. Make sure that the provider configuration contains all required GitHub credentials")
	}

	prs, _, gitHubErr := client.GitHubClient.PullRequests.List(context.Background(), d.Get("repository_owner").(string), d.Get("repository_name").(string), &github.PullRequestListOptions{
		State:     "open",
		Base:      d.Get("target_branch").(string),
		Head:      fmt.Sprintf("%s:%s", d.Get("repository_owner").(string), d.Get("working_branch").(string)),
		Sort:      "updated",
		Direction: "desc",
	})

	if gitHubErr != nil {
		return nil, fmt.Errorf("Error while searching for PRs on GitHub: %s", gitHubErr)
	}

	branchName := d.Get("target_branch").(string)
	if len(prs) > 0 {
		branchName = *prs[0].Head.Ref
	}

	fs := memfs.New()
	storer := memory.NewStorage()

	// Clones the repository into the worktree (fs) and store all the .git content into the storer
	repository, cloneErr := git.Clone(storer, fs, &git.CloneOptions{
		URL:               fmt.Sprintf("https://github.com/%s/%s.git", d.Get("repository_owner").(string), d.Get("repository_name").(string)),
		Auth:              client.GitHubGitAuth,
		RemoteName:        "origin",
		ReferenceName:     plumbing.NewBranchReferenceName(branchName),
		SingleBranch:      true,
		NoCheckout:        false,
		Depth:             1,
		RecurseSubmodules: git.NoRecurseSubmodules,
		Progress:          nil,
		Tags:              git.NoTags,
	})

	if cloneErr != nil {
		return nil, fmt.Errorf("Error while checking out repository: %s", cloneErr)
	}

	return repository, nil
}

func resourceRepositoryTemplateGitHubCreate(d *schema.ResourceData, meta interface{}) error {
	err := resourceRepositoryTemplateGitHubUpdate(d, meta)

	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%s/%s", d.Get("repository_owner").(string), d.Get("repository_name").(string)))
	return nil
}

func resourceRepositoryTemplateGitHubUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	repository, cloneErr := cloneRepository(d, client)

	if cloneErr != nil {
		return cloneErr
	}

	worktree, worktreeError := repository.Worktree()

	if worktreeError != nil {
		return fmt.Errorf("Error getting repository worktree: %s", worktreeError)
	}

	head, headErr := repository.Head()

	if headErr != nil {
		return fmt.Errorf("Error getting repository head: %s", headErr)
	}

	if head.Name().String() != fmt.Sprintf("refs/heads/%s", d.Get("working_branch").(string)) {
		checkoutErr := worktree.Checkout(&git.CheckoutOptions{
			Hash:   head.Hash(),
			Branch: plumbing.NewBranchReferenceName(d.Get("working_branch").(string)),
			Create: true,
		})

		if checkoutErr != nil {
			return fmt.Errorf("Error checking out working branch: %s", checkoutErr)
		}
	}

	for filePath, contents := range d.Get("files").(map[string]interface{}) {
		file, openErr := worktree.Filesystem.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		if openErr != nil {
			return fmt.Errorf("Error while opening file at path %s: %s", filePath, openErr)
		}

		_, writeErr := file.Write([]byte(contents.(string)))
		if writeErr != nil {
			return fmt.Errorf("Error while writing file at path %s: %s", filePath, writeErr)
		}

		closeErr := file.Close()
		if closeErr != nil {
			return fmt.Errorf("Error while closing file at path %s: %s", filePath, closeErr)
		}

		_, addErr := worktree.Add(filePath)
		if addErr != nil {
			return fmt.Errorf("Error while adding file at path %s to the Git index: %s", addErr)
		}
	}

	status, statusErr := worktree.Status()

	if statusErr != nil {
		return fmt.Errorf("Error getting the working tree status: %s", statusErr)
	}

	if !status.IsClean() {
		_, commitErr := worktree.Commit(client.CommitMessage, &git.CommitOptions{
			All: false,
			Author: &object.Signature{
				Name: client.CommitAuthorName,
				When: time.Now(),
			},
		})

		if commitErr != nil {
			return fmt.Errorf("Error committing changes: %s", commitErr)
		}

		pushErr := repository.Push(&git.PushOptions{
			RemoteName: "origin",
			RefSpecs:   []config.RefSpec{config.RefSpec("+refs/heads/*:refs/heads/*")},
			Auth:       client.GitHubGitAuth,
			Progress:   nil,
		})

		if pushErr != nil {
			return fmt.Errorf("Error pushing changes: %s", pushErr)
		}

		_, _, pullRequestErr := client.GitHubClient.PullRequests.Create(context.Background(), d.Get("repository_owner").(string), d.Get("repository_name").(string), &github.NewPullRequest{
			Title:               github.String(client.CommitMessage),
			Base:                github.String(d.Get("target_branch").(string)),
			Head:                github.String(d.Get("working_branch").(string)),
			Body:                github.String("This PR was created by Terraform."),
			MaintainerCanModify: github.Bool(true),
		})

		if !pullRequestAlreadyExists(pullRequestErr) {
			if pullRequestErr != nil {
				return fmt.Errorf("Error creating pull request: %s", pullRequestErr)
			}
		}
	}

	return nil
}

func pullRequestAlreadyExists(pullRequestErr error) bool {
	return pullRequestErr != nil && regexp.MustCompile(`^POST https://api.github.com/repos/[^/]+/[^/]+/pulls: 422 Validation Failed \[{Resource:PullRequest Field: Code:custom Message:A pull request already exists for .+\.}\]`).MatchString(pullRequestErr.Error())
}

func resourceRepositoryTemplateGitHubRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	repository, cloneErr := cloneRepository(d, client)

	if cloneErr != nil {
		return cloneErr
	}

	worktree, worktreeError := repository.Worktree()

	if worktreeError != nil {
		return fmt.Errorf("Error getting repository worktree: %s", worktreeError)
	}

	files := map[string]string{}

	for filePath, _ := range d.Get("files").(map[string]interface{}) {
		file, openErr := worktree.Filesystem.OpenFile(filePath, os.O_RDONLY, 0664)
		if openErr != nil {
			fmt.Errorf("Error while opening file: %s", filePath)
		}

		contents, readErr := ioutil.ReadAll(file)
		files[filePath] = string(contents)
		if readErr != nil {
			fmt.Errorf("Error while reading file at path %s: %s", filePath, readErr)
		}

		file.Close()
	}

	d.Set("files", files)

	return nil
}

func resourceRepositoryTemplateGitHubDelete(d *schema.ResourceData, meta interface{}) error {
	// The delete step essentially does nothing, as it's unclear what it means to
	// "delete" a template. Should the files be removed? Should the files be reverted
	// to what they were before the template was first applied? How do we know when
	// the template was first applied? The safest option, therefore, is to just
	// leave the repository in the state that it's currently in.

	return nil
}
