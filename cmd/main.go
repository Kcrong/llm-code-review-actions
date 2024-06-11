package main

import (
	"context"
	"fmt"
	"github.com/Kcrong/groq-code-review-actions/pkg/prompt"
	"github.com/magicx-ai/groq-go/groq"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	defaultModel    = groq.ModelIDLLAMA370B
	defaultDiffName = "diff.txt"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatalln("GITHUB_TOKEN environment variable is not set")
		return
	}

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatalln("GROQ_API_KEY environment variable not set")
	}

	// Read the PR diff
	diff, err := os.ReadFile(filepath.Join(os.Getenv("GITHUB_WORKSPACE"), defaultDiffName))
	if err != nil {
		log.Fatalf("Error reading diff: %+v\n", err)
		return
	}

	results, err := run(RunParameters{
		ApiKey: apiKey,
		Diff:   string(diff[:]),
		Model:  defaultModel,
	})
	if err != nil {
		log.Fatalf("Error running action: %+v", err)
	}

	if err := createComment(token, results); err != nil {
		log.Fatalf("Error creating comment: %+v", err)
	}
}

func createComment(token string, content string) error {
	owner := os.Getenv("GITHUB_REPOSITORY_OWNER")
	repo := os.Getenv("GITHUB_REPOSITORY_NAME")
	prNumber := os.Getenv("GITHUB_PR_NUMBER")

	if owner == "" || repo == "" || prNumber == "" {
		return errors.New("one or more required environment variables are not set")
	}

	// Create a new GitHub client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Create a new comment
	comment := &github.IssueComment{
		Body: &content,
	}

	prNum, err := strconv.Atoi(prNumber)
	if err != nil {
		return errors.Wrap(err, "error converting PR number")
	}

	_, _, err = client.Issues.CreateComment(ctx, owner, repo, prNum, comment)
	if err != nil {
		return errors.Wrap(err, "error creating comment")
	}

	return nil
}

type RunParameters struct {
	ApiKey string
	Diff   string
	Model  groq.ModelID
}

func run(params RunParameters) (string, error) {
	cli := groq.NewClient(params.ApiKey, http.DefaultClient)

	req := groq.ChatCompletionRequest{
		Messages: []groq.Message{
			{
				Role:    groq.MessageRoleSystem,
				Content: prompt.CodeReviewRulePrompt,
			},
			{
				Role:    groq.MessageRoleUser,
				Content: fmt.Sprintf("Here is my github PR changes.\n%s", params.Diff),
			},
		},
		Model:       params.Model,
		MaxTokens:   0,
		Temperature: 0.7,
		TopP:        0.9,
		NumChoices:  1,
		Stream:      false,
	}

	// Get the response
	resp, err := cli.CreateChatCompletion(req)
	if err != nil {
		return "", errors.Wrap(err, "error creating chat completion")
	}

	return resp.Choices[0].Message.Content, nil
}
