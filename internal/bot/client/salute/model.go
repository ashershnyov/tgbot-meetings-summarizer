package salute

// uploadResponse is a response to file being uploaded.
type uploadResponse struct {
	Status int `json:"status"`
	Result struct {
		RequestFileID string `json:"request_file_id"`
	} `json:"result"`
}

// transcribeRequest is a request to transcribe file with given options.
type transcribeRequest struct {
	Options       transcriptionOptions `json:"options"`
	RequestFileID string               `json:"request_file_id"`
}

// transcriptionOptions define transcription options.
type transcriptionOptions struct {
	AudioEncoding   string `json:"audio_encoding"`
	SampleRate      int    `json:"sample_rate,omitempty"`
	Channels        int    `json:"channels_count,omitempty"`
	Language        string `json:"language,omitempty"`
	HypothesesCount int    `json:"hypotheses_count,omitempty"`
}

// taskResponse is a response of task status check.
type taskResponse struct {
	Status int `json:"status"`
	Result struct {
		ID             string `json:"id"`
		Created        string `json:"created_at"`
		Updated        string `json:"updated_at"`
		Status         string `json:"status"`
		ResponseFileID string `json:"response_file_id"`
	} `json:"result"`
}

// transcriptionResult is a result of file transcription.
type transcriptionResult struct {
	Results []result `json:"results,omitempty"`
}

// result is a singular transcription.
type result struct {
	Text string `json:"text"`
}
