package vcs

import (
	"context"

	"github.com/libgit2/git2go/v34"
)

type Repo struct{
	repo *git.Repository
	config GitConfig
}

type GitConfig struct{
	UserName string
	UserEmail string
	RepoPath string
}

type GitRepository interface {
	CreateRepo(ctx context.Context, path string) error
	Commit(ctx context.Context, message string) (string, error)
	Diff(ctx context.Context, fileName string) (string, error)
	AddRemote(ctx context.Context, name, url string) error
	Push(ctx context.Context, remoteName, branch string) error
	Close() error
}
