package config

import (
	"testing"

	"github.com/matryer/is"
)

func TestNewRepoSource(t *testing.T) {
	repoPath := "./testdata"
	rs, err := NewRepoSource(repoPath)
	repos := []string{
		"z-repo",
		"1-repo",
		"a-repo",
		"m-repo",
		"b-repo",
	}
	for _, r := range repos {
		rs.InitRepo(r, true)
	}
	is := is.New(t)
	is.NoErr(err)
	is.Equal(len(rs.repos), 5) // there should be 5 repos
}

func TestOrderReposAlphabetically(t *testing.T) {
	repoPath := "./testdata"
	rs, err := NewRepoSource(repoPath)
	repos := []string{
		"z-repo",
		"1-repo",
		"a-repo",
		"m-repo",
		"b-repo",
	}
	for _, r := range repos {
		rs.InitRepo(r, true)
	}
	rs.Sort("alphabetical", []RepoConfig{})
	expected := map[string]int{ // repos and their expected index
		"1-repo": 0,
		"a-repo": 1,
		"b-repo": 2,
		"m-repo": 3,
		"z-repo": 4,
	}
	result := rs.AllRepos() // as returned from ls command
	is := is.New(t)
	is.NoErr(err)
	for i, repo := range result {
		is.Equal(expected[repo.Repo()], i) // repos should be alphabetically ordered
	}
}

func TestOrderReposConfig(t *testing.T) {
	repoPath := "./testdata"
	rs, err := NewRepoSource(repoPath)
	config := []RepoConfig{
		{
			Name: "test-repo-a",
			Repo: "a-repo",
		},
		{
			Name: "test-repo-1",
			Repo: "1-repo",
		},
		{
			Name: "test-repo-z",
			Repo: "z-repo",
		},
		{
			Name: "test-repo-b",
			Repo: "b-repo",
		},
		{
			Name: "test-repo-m",
			Repo: "m-repo",
		},
	}
	repos := []string{
		"z-repo",
		"1-repo",
		"a-repo",
		"m-repo",
		"b-repo",
	}
	for _, r := range repos {
		rs.InitRepo(r, true)
	}
	rs.Sort("config", config)
	expected := map[string]int{ // repos and their expected index
		"a-repo": 0,
		"1-repo": 1,
		"z-repo": 2,
		"b-repo": 3,
		"m-repo": 4,
	}
	result := rs.AllRepos() // as returned from ls command
	is := is.New(t)
	is.NoErr(err)
	for i, repo := range result {
		is.Equal(expected[repo.Repo()], i) // repos should be ordered as in config file
	}
}
