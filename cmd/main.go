package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/magicx-ai/groq-code-review-actions/pkg/prompt"
	// TODO(@magicx-ai): Delete vendor.
	// magicx-ai/groq-go is private for now, we need to keep vendor until it's public.
	"github.com/magicx-ai/groq-go/groq"
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
		APIKey: apiKey,
		Diff:   string(diff),
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
	prNumber := os.Getenv("GITHUB_PR_NUMBER")
	if prNumber == "" {
		return errors.New("GITHUB_PR_NUMBER environment variable is not set")
	}

	repository := os.Getenv("GITHUB_REPOSITORY")
	if repository == "" {
		return errors.New("GITHUB_REPOSITORY environment variable is not set")
	}

	split := strings.Split(repository, "/")
	if len(split) != 2 {
		return errors.New("invalid GITHUB_REPOSITORY format")
	}

	owner, repo := split[0], split[1]

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
	APIKey string
	Diff   string
	Model  groq.ModelID
}

func run(params RunParameters) (string, error) {
	cli := groq.NewClient(params.APIKey, http.DefaultClient)

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
