package vcs

import (
	"context"
	"fmt"
	"os"
	"time"

	git "github.com/libgit2/git2go/v34"
)


func NewRepo(path string) *Repo{
	return &Repo{
		config: GitConfig{RepoPath: path},
	}
}

func (g *Repo) Create(ctx context.Context, path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory")
	}
	repo, err := git.InitRepository(path, false)
	if err != nil{
		return fmt.Errorf("failed to create directory: %s", err)
	}
	g.repo = repo
	g.config.RepoPath = path
	config, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	defer config.Free()
	if g.config.UserName != "" {
		if err := config.SetString("user.name", g.config.UserName); err != nil {
			return fmt.Errorf("failed to set user.name: %w", err)
		}
	}
	if g.config.UserEmail != "" {
		if err := config.SetString("user.email", g.config.UserEmail); err != nil {
			return fmt.Errorf("failed to set user.email: %w", err)
		}
	}
	return nil
}

func (g *Repo) Open(path string) error {
	repo, err := git.OpenRepository(path)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}
	g.repo = repo
	g.config.RepoPath = path
	return nil
}

func (g *Repo) Commit(ctx context.Context, message string) (string, error) {
	if g.repo == nil {
		return "", fmt.Errorf("repository not initialized")
	}
	index, err := g.repo.Index()
	if err != nil {
		return "", fmt.Errorf("failed to get index: %w", err)
	}
	defer index.Free()
	if err := index.AddAll([]string{}, git.IndexAddDefault, nil); err != nil {
		return "", fmt.Errorf("failed to add files to index: %w", err)
	}
	if err := index.Write(); err != nil {
		return "", fmt.Errorf("failed to write index: %w", err)
	}
	treeOid, err := index.WriteTree()
	if err != nil {
		return "", fmt.Errorf("failed to write tree: %w", err)
	}

	tree, err := g.repo.LookupTree(treeOid)
	if err != nil {
		return "", fmt.Errorf("failed to lookup tree: %w", err)
	}
	defer tree.Free()
	var parents []*git.Commit
	head, err := g.repo.Head()
	if err == nil {
		defer head.Free()
		headCommit, err := g.repo.LookupCommit(head.Target())
		if err == nil {
			parents = append(parents, headCommit)
			defer headCommit.Free()
		}
	}
	sig := &git.Signature{
		Name:  g.config.UserName,
		Email: g.config.UserEmail,
		When:  time.Now(),
	}
	commitOid, err := g.repo.CreateCommit("HEAD", sig, sig, message, tree, parents...)
	if err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}
	return commitOid.String(), nil
}


func (g *Repo) Diff(ctx context.Context, fileName string) (string, error) {
	if g.repo == nil {
		return "", fmt.Errorf("repository not initialized")
	}
	// Get HEAD tree
	head, err := g.repo.Head()
	if err != nil {
		return g.diffAgainstEmpty(fileName)
	}
	defer head.Free()
	headCommit, err := g.repo.LookupCommit(head.Target())
	if err != nil {
		return "", fmt.Errorf("failed to lookup HEAD commit: %w", err)
	}
	defer headCommit.Free()
	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD tree: %w", err)
	}
	defer headTree.Free()
	index, err := g.repo.Index()
	if err != nil {
		return "", fmt.Errorf("failed to get index: %w", err)
	}
	defer index.Free()
	diff, err := g.repo.DiffTreeToIndex(headTree, index, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create diff: %w", err)
	}
	defer diff.Free()
	if fileName != "" {
		return g.extractFileDiff(diff, fileName)
	}
	var diffStr string
	err = diff.ForEach(func(delta git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
		diffStr += fmt.Sprintf("diff --git a/%s b/%s\n", delta.OldFile.Path, delta.NewFile.Path)
		return func(hunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
			diffStr += fmt.Sprintf("@@ -%d,%d +%d,%d @@ %s\n", 
				hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines, hunk.Header)
			return func(line git.DiffLine) error {
				diffStr += string(line.Origin) + line.Content
				return nil
			}, nil
		}, nil
	}, git.DiffDetailLines)
	if err != nil {
		return "", fmt.Errorf("failed to process diff: %w", err)
	}
	return diffStr, nil
}

func (g *Repo) diffAgainstEmpty(fileName string) (string, error) {
	index, err := g.repo.Index()
	if err != nil {
		return "", fmt.Errorf("failed to get index: %w", err)
	}
	defer index.Free()
	diff, err := g.repo.DiffIndexToWorkdir(index, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create diff: %w", err)
	}
	defer diff.Free()
	if fileName != "" {
		return g.extractFileDiff(diff, fileName)
	}
	var diffStr string
	err = diff.ForEach(func(delta git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
		diffStr += fmt.Sprintf("diff --git a/%s b/%s\n", delta.OldFile.Path, delta.NewFile.Path)
		return func(hunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
			diffStr += fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", 
				hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
			return func(line git.DiffLine) error {
				diffStr += string(line.Origin) + line.Content
				return nil
			}, nil
		}, nil
	}, git.DiffDetailLines)

	return diffStr, err
}

// extractFileDiff extracts diff for a specific file
func (g *Repo) extractFileDiff(diff *git.Diff, fileName string) (string, error) {
	var fileDiff string
	found := false

	err := diff.ForEach(func(delta git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
		if delta.NewFile.Path == fileName || delta.OldFile.Path == fileName {
			found = true
			fileDiff += fmt.Sprintf("diff --git a/%s b/%s\n", delta.OldFile.Path, delta.NewFile.Path)
			
			return func(hunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
				fileDiff += fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", 
					hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
				
				return func(line git.DiffLine) error {
					fileDiff += string(line.Origin) + line.Content
					return nil
				}, nil
			}, nil
		}
		return nil, nil
	}, git.DiffDetailLines)

	if err != nil {
		return "", err
	}

	if !found {
		return "", fmt.Errorf("file %s not found in diff", fileName)
	}

	return fileDiff, nil
}

// AddRemote adds a remote repository
func (g *Repo) AddRemote(ctx context.Context, name, url string) error {
	if g.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	_, err := g.repo.Remotes.Create(name, url)
	if err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	return nil
}

// Push pushes commits to remote repository
func (g *Repo) Push(ctx context.Context, remoteName, branch string) error {
	if g.repo == nil {
		return fmt.Errorf("repository not initialized")
	}
	remote, err := g.repo.Remotes.Lookup(remoteName)
	if err != nil {
		return fmt.Errorf("failed to lookup remote: %w", err)
	}
	defer remote.Free()

	// Prepare refspec
	refspec := fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch)

	// Push options
	pushOptions := &git.PushOptions{}

	// Perform push
	err = remote.Push([]string{refspec}, pushOptions)
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// Close closes the repository and frees resources
func (g *Repo) Close() error {
	if g.repo != nil {
		g.repo.Free()
		g.repo = nil
	}
	return nil
}

// Example usage function
func ExampleUsage() {
	repo := NewRepo("/path/to/repo")
	ctx := context.Background()
	defer repo.Close()

	if err := repo.Create(ctx, "/path/to/repo"); err != nil {
		fmt.Printf("Error creating repo: %v\n", err)
		return
	}

	// Commit changes
	commitHash, err := repo.Commit(ctx, "Initial commit")
	if err != nil {
		fmt.Printf("Error committing: %v\n", err)
		return
	}
	fmt.Printf("Created commit: %s\n", commitHash)

	// Get diff for a file
	diff, err := repo.Diff(ctx, "main.go")
	if err != nil {
		fmt.Printf("Error getting diff: %v\n", err)
	} else {
		fmt.Printf("Diff:\n%s\n", diff)
	}

	// Add remote
	if err := repo.AddRemote(ctx, "origin", "https://github.com/user/repo.git"); err != nil {
		fmt.Printf("Error adding remote: %v\n", err)
		return
	}

	// Push to remote
	if err := repo.Push(ctx, "origin", "main"); err != nil {
		fmt.Printf("Error pushing: %v\n", err)
		return
	}

	fmt.Println("Successfully pushed to remote!")
}
