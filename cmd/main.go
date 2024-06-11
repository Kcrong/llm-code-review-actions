package main

import (
	"fmt"
	"github.com/Kcrong/groq-code-review-actions/pkg/prompt"
	"github.com/magicx-ai/groq-go/groq"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	defaultModel    = groq.ModelIDLLAMA370B
	defaultDiffName = "diff.txt"
)

func main() {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatalf("GROQ_API_KEY environment variable not set")
	}

	// Read the PR diff
	diff, err := os.ReadFile(filepath.Join(os.Getenv("GITHUB_WORKSPACE"), defaultDiffName))
	if err != nil {
		log.Fatalf("Error reading diff: %+v", err)
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

	fmt.Println(results)
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
