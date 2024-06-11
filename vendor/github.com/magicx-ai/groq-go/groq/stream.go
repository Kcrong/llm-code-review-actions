package groq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/tmaxmax/go-sse"
)

type ChatCompletionStreamResponse struct {
	Response ChatCompletionResponse
	Error    error
}

func (c *client) CreateChatCompletionStream(req ChatCompletionRequest) (<-chan *ChatCompletionStreamResponse, error) {
	if !req.Stream {
		return nil, fmt.Errorf("stream must be set to true")
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.baseURL)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	responseCh := make(chan *ChatCompletionStreamResponse)

	conn := sse.DefaultClient.NewConnection(httpReq)
	conn.SubscribeToAll(func(event sse.Event) {
		var chatResp ChatCompletionResponse
		if err := json.Unmarshal([]byte(event.Data), &chatResp); err != nil {
			responseCh <- &ChatCompletionStreamResponse{Error: errors.Wrap(err, "failed to unmarshal response")}
			return
		}
		responseCh <- &ChatCompletionStreamResponse{Response: chatResp}
	})

	go func() {
		if err := conn.Connect(); err != nil {
			responseCh <- &ChatCompletionStreamResponse{
				Error: errors.Wrap(err, "failed to connect to the server"),
			}
		}
	}()

	return responseCh, nil
}
