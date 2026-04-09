package giga

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/client/auth"
)

const (
	chatEndpoint = "https://gigachat.devices.sberbank.ru/api/v1/chat/completions"

	workerCount = 3
	queueSize   = 10

	model       = "GigaChat-Pro"
	temperature = 0.3
	maxTokens   = 500

	summaryPrompt = `Сделай краткую суммаризацию следующей транскрипции. Выдели ключевые тезисы, сохрани структуру мысли. Максимум — 5–7 предложений.\n\nТранскрипция:\n%s`

	authScope = "GIGACHAT_API_PERS"
)

var ErrQueueOverflow = errors.New("job queue overflow")

// job defines singular job for GigaChat to process.
type job struct {
	Message string
	ReplyCh chan Result
}

// Result defines singular job's result.
type Result struct {
	Message string
	Err     error
}

// Client is a GigaChat client with job queue.
type Client struct {
	http   *http.Client
	auth   *auth.Client
	jobCh  chan job
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewClient returns a newly created GigaChat client.
func NewClient(authToken string) (*Client, error) {
	c := &Client{
		http:   &http.Client{Timeout: 60 * time.Second},
		jobCh:  make(chan job, queueSize),
		stopCh: make(chan struct{}),
	}

	authClient, err := auth.NewClient(authToken, authScope)
	if err != nil {
		return nil, fmt.Errorf("error creating auth client: %w", err)
	}

	c.auth = authClient

	for i := 0; i < workerCount; i++ {
		c.wg.Add(1)
		go c.worker(i + 1)
	}

	return c, nil
}

// SubmitSummaryJob adds a job to summarize passed text to the queue.
func (c *Client) SubmitSummaryJob(transcript string) (<-chan Result, error) {
	job := job{
		Message: fmt.Sprintf(summaryPrompt, transcript),
		ReplyCh: make(chan Result),
	}
	select {
	case c.jobCh <- job:
		return job.ReplyCh, nil
	default:
		return nil, ErrQueueOverflow
	}
}

// SubmitChatJob adds a job to regularly chat with the user to the queue.
func (c *Client) SubmitChatJob(msg string) (<-chan Result, error) {
	job := job{
		Message: msg,
		ReplyCh: make(chan Result),
	}
	select {
	case c.jobCh <- job:
		return job.ReplyCh, nil
	default:
		return nil, ErrQueueOverflow
	}
}

// Stop stops all workers.
func (c *Client) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

func (c *Client) worker(id int) {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			return

		case job, ok := <-c.jobCh:
			if !ok {
				return
			}

			summary, err := c.sendUserMessage(job.Message)
			job.ReplyCh <- Result{
				Message: summary,
				Err:     err,
			}
		}
	}
}

func (c *Client) sendUserMessage(msg string) (string, error) {
	token, err := c.auth.Token()
	if err != nil {
		return "", fmt.Errorf("error getting GigaChat token: %w", err)
	}

	messages := []message{
		{Role: "user", Content: msg},
	}

	payload := chatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error generating GigaChat request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, chatEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("error sending GigaChat request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending GigaChat request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading GigaChat response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GigaChat HTTP error with code %d: %s", resp.StatusCode, string(raw))
	}

	var cr chatResponse
	if err = json.Unmarshal(raw, &cr); err != nil {
		return "", fmt.Errorf("error parsing GigaChat reponse: %w", err)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices from GigaChat")
	}

	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}
