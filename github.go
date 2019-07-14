package main

import (
	"context"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
	"os"
)

type GitHub struct {
	client githubv4.Client
}

func CreateGitHubInstance(token string) GitHub {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := githubv4.NewClient(httpClient)
	return GitHub{*client}
}

func (g GitHub) ListBranch(name string) ([]string, error) {
	type refs struct {
		Name string
	}
	var query struct {
		Repository struct {
			Refs struct {
				Nodes []refs
			} `graphql:"refs(first: 50, refPrefix: \"refs/heads/\")"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner": githubv4.String(os.Getenv("GITHUB_OWNER")),
		"name":  githubv4.String(name),
	}

	err := g.client.Query(context.Background(), &query, variables)
	if err != nil {
		return []string{}, err
	}
	var arr []string
	for _, v := range query.Repository.Refs.Nodes {
		arr = append(arr, v.Name)
	}
	return arr, nil
}
