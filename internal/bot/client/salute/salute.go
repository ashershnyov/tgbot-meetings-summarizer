package salute

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/client/auth"
)

const (
	baseURL        = "https://smartspeech.sber.ru/rest/v1"
	uploadEndpoint = baseURL + "/data:upload"
	taskEndpoint   = baseURL + "/speech:async_recognize"
	statusEndpoint = baseURL + "/task:get"
	resultEndpoint = baseURL + "/data:download"

	pollInterval = 3 * time.Second
	pollTimeout  = 5 * time.Minute

	sampleRate      = 16000
	channels        = 1
	language        = "ru-RU"
	hypothesesCount = 1

	authScope = "SALUTE_SPEECH_PERS"
)

// Client defines SaluteSpeech transcription client.
type Client struct {
	http *http.Client
	auth *auth.Client
}

// NewClient returns a newly created SaluteSpeech client.
func NewClient(authToken string) (*Client, error) {
	c := &Client{
		http: &http.Client{Timeout: 60 * time.Second},
	}

	authClient, err := auth.NewClient(authToken, authScope)
	if err != nil {
		return nil, fmt.Errorf("error creating auth client: %w", err)
	}

	c.auth = authClient

	return c, nil
}

func (c *Client) do(method, url string, body io.Reader, contentType string) ([]byte, error) {
	token, err := c.auth.Token()
	if err != nil {
		return []byte{}, fmt.Errorf("error getting GigaChat token: %w", err)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTP error with code %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func (c *Client) uploadFile(src io.ReadCloser, name string) (string, error) {
	defer src.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	part, err := mw.CreateFormFile("file", name)
	if err != nil {
		return "", fmt.Errorf("error creating multipart: %w", err)
	}
	if _, err = io.Copy(part, src); err != nil {
		return "", fmt.Errorf("error writing to multipart: %w", err)
	}
	mw.Close()

	raw, err := c.do("POST", uploadEndpoint, &buf, mw.FormDataContentType())
	if err != nil {
		return "", fmt.Errorf("error uploading file to SaluteSpeech: %w", err)
	}

	var result uploadResponse
	if err = json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("error parsing SaluteSpeech response: %w", err)
	}
	if result.Result.RequestFileID == "" {
		return "", errors.New("empty request_file_id in SaluteSpeech response")
	}

	return result.Result.RequestFileID, nil
}

func (c *Client) createTask(opts transcriptionOptions, fileID string) (string, error) {
	payload := transcribeRequest{Options: opts, RequestFileID: fileID}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error creating SaluteSpeech request: %w", err)
	}

	raw, err := c.do("POST", taskEndpoint, bytes.NewReader(body), "application/json")
	if err != nil {
		return "", fmt.Errorf("error creating new SaluteSpeech task: %w", err)
	}

	var result taskResponse
	if err = json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("error parsing SaluteSpeech response: %w", err)
	}
	if result.Result.ID == "" {
		return "", errors.New("empty task id received")
	}

	return result.Result.ID, nil
}

func (c *Client) waitForCompletion(taskID string) (string, error) {
	deadline := time.Now().Add(pollTimeout)
	url := fmt.Sprintf("%s?id=%s", statusEndpoint, taskID)

	for {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("SaluteSpeech task timeout %s", taskID)
		}

		raw, err := c.do("GET", url, nil, "")
		if err != nil {
			return "", fmt.Errorf("error getting SaluteSpeech task status: %w", err)
		}

		var result taskResponse
		if err = json.Unmarshal(raw, &result); err != nil {
			return "", fmt.Errorf("error parsing SaluteSpeech response: %w", err)
		}

		status := result.Result.Status

		switch status {
		case "DONE":
			fileID := result.Result.ResponseFileID
			if fileID == "" {
				return "", errors.New("response_file_id is empty")
			}
			return fileID, nil

		case "ERROR":
			return "", errors.New("error completing SaluteSpeech task")

		case "CANCELED":
			return "", errors.New("task cancelled")

		case "NEW", "RUNNING":
		}

		time.Sleep(pollInterval)
	}
}

func (c *Client) downloadResult(responseFileID string) (*transcriptionResult, error) {
	url := fmt.Sprintf("%s?response_file_id=%s", resultEndpoint, responseFileID)
	raw, err := c.do("GET", url, nil, "")
	if err != nil {
		return nil, fmt.Errorf("error downloading SaluteSpeech result: %w", err)
	}

	var result []transcriptionResult
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("error parsing SaluteSpeech result: %w", err)
	}

	return &result[0], nil
}

func getEncodingFromMIME(mime string) string {
	switch mime {
	case "audio/mpeg", "audio/mp3":
		return "MP3"
	case "audio/ogg":
		return "OPUS"
	case "audio/wav", "audio/x-wav":
		return "PCM_S16LE"
	case "audio/flac", "audio/x-flac":
		return "FLAC"
	case "audio/alaw":
		return "ALAW"
	case "audio/mulaw":
		return "MULAW"
	}
	return ""
}

// Transcribe invokes full transcription pipeline: Upload -> Create task -> Wait for task to complete -> Fetch result.
// Returns a string with transcription.
func (c *Client) Transcribe(src io.ReadCloser, name string, mime string) (string, error) {
	fileID, err := c.uploadFile(src, name)
	if err != nil {
		return "", fmt.Errorf("error loading audio: %w", err)
	}

	opts := transcriptionOptions{
		AudioEncoding:   getEncodingFromMIME(mime),
		SampleRate:      sampleRate,
		Channels:        channels,
		Language:        language,
		HypothesesCount: hypothesesCount,
	}
	taskID, err := c.createTask(opts, fileID)
	if err != nil {
		return "", fmt.Errorf("error creating task: %w", err)
	}

	responseFileID, err := c.waitForCompletion(taskID)
	if err != nil {
		return "", fmt.Errorf("error waiting for task completion: %w", err)
	}

	result, err := c.downloadResult(responseFileID)
	if err != nil {
		return "", fmt.Errorf("error loading task result: %w", err)
	}

	return result.Results[0].Text, nil
}
