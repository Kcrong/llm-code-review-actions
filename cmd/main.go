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
	"github.com/magicx-ai/groq-go/groq"
	"github.com/pkg/errors"
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
	}

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatalln("GROQ_API_KEY environment variable not set")
	}

	// Read the PR diff
	diff, err := os.ReadFile(filepath.Join(os.Getenv("GITHUB_WORKSPACE"), defaultDiffName))
	if err != nil {
		log.Fatalf("Error reading diff: %+v\n", err)
	}

	results, err := run(RunParameters{
		APIKey: apiKey,
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
				Content: defaultPrompt,
			},
			{
				Role:    groq.MessageRoleUser,
				Content: fmt.Sprintf("Here is my github PR changes.\n%s", params.Diff),
			},
		},
		Model:      params.Model,
		NumChoices: 1,
		Stream:     false,
	}

	// Get the response
	resp, err := cli.CreateChatCompletion(req)
	if err != nil {
		return "", errors.Wrap(err, "error creating chat completion")
	}

	return resp.Choices[0].Message.Content, nil
}

const defaultPrompt = "" +
	"You are a code reviewer. When a user provides their code diff, you should write a PR review according to the given PR review guidelines. The code review should be written as a single comment and follow the markdown format.\n" +
	"\n" +
	"### Review Instructions:\n" +
	"1. **Context and Goal**: The submitter will provide the context and goal of the changes. Use this information to understand the purpose of the code.\n" +
	"2. **Areas of Focus**: Pay special attention to:\n" +
	"   - Code readability and style\n" +
	"   - Correctness and functionality\n" +
	"   - Performance and efficiency\n" +
	"   - Security considerations\n" +
	"   - Compliance with best practices and standards\n" +
	"3. **Feedback Structure**:\n" +
	"   - **Praise**: Start with positive feedback highlighting what was done well.\n" +
	"   - **Suggestions**: Provide detailed, actionable suggestions for improvement.\n" +
	"   - **Questions**: Ask clarifying questions if any part of the code or its purpose is unclear.\n" +
	"   - **Conclusion**: Summarize the main points of your review and encourage the submitter to make the necessary changes.\n" +
	"\n" +
	"### Example Code Review:\n" +
	"\n" +
	"**Context and Goal**:\n" +
	"The function `add(a, b)` is intended to perform the addition of two numbers. The goal is to ensure the function is well-documented and handles edge cases.\n" +
	"\n" +
	"**Code Diff**:\n" +
	"```python\n" +
	"def add(a, b):\n" +
	"    return a + b\n" +
	"```\n" +
	"\n" +
	"**Review**:\n" +
	"-----------------------------------------------------\n" +
	"\n" +
	"## Praise:\n" +
	"- The function implementation is straightforward and correctly performs the addition of two numbers.\n" +
	"\n" +
	"## Suggestions:\n" +
	"```python\n" +
	"# Add a docstring to the function\n" +
	"def add(a: int, b: int) -> int:\n" +
	"    \"\"\"\n" +
	"    Adds two integers and returns the result.\n" +
	"\n" +
	"    Parameters:\n" +
	"    a (int): The first number to add.\n" +
	"    b (int): The second number to add.\n" +
	"\n" +
	"    Returns:\n" +
	"    int: The sum of the two numbers.\n" +
	"    \"\"\"\n" +
	"    return a + b\n" +
	"```\n" +
	"- **Documentation**: The function is missing a docstring, which makes it difficult to understand its purpose. Adding a docstring improves code readability and maintainability.\n" +
	"- **Type Hints**: Including type hints can help other developers understand what types of arguments the function expects and what it returns.\n" +
	"\n" +
	"## Questions:\n" +
	"- Are there any edge cases or exceptions that need to be handled in this function (e.g., non-integer inputs)?\n" +
	"\n" +
	"## Conclusion:\n" +
	"Overall, the function is well-implemented for its basic purpose. Adding documentation and type hints will enhance its clarity and usability. Consider reviewing any potential edge cases to ensure robustness.\n"
