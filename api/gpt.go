package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

const model = "gpt-3.5-turbo"
const openaiURL = "https://api.openai.com/v1/chat/completions"

type OpenaiRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenaiResponse struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
	Model   string   `json:"model"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func GetApiKey() (apiKey string) {
	var err = godotenv.Load(".env")
	if err != nil {
		fmt.Printf("error: tokenが読み込めなかった!: %v", err)
	}
	apiKey = os.Getenv("APIKEY")
	return apiKey
}

var apiKey = GetApiKey()

var (
	JsonError = errors.New("Error:invalid character '<' looking for beginning of value")
)

func (res *OpenaiResponse) OverTokenCheck() bool {
	if len(res.Choices) == 0 {
		return true
	}
	if res.Choices[0].FinishReason == "length" {
		return true
	}
	return false
}

func (res *OpenaiResponse) Text() string {
	if len(res.Choices) == 0 {
		return ""
	}
	return res.Choices[0].Message.Content
}

func RequestOpenaiAPIByStrings(strings []string) (OpenaiResponse, error) {
	var requestMessages = []Message{}
	for i, str := range strings {
		if i == 0 {
			requestMessages = append(requestMessages, Message{
				Role:    "system",
				Content: str,
			})
		} else if i%2 == 1 {
			requestMessages = append(requestMessages, Message{
				Role:    "user",
				Content: str,
			})
		} else {
			requestMessages = append(requestMessages, Message{
				Role:    "assistant",
				Content: str,
			})
		}
	}

	var requestBody = OpenaiRequest{
		Model:    model,
		Messages: requestMessages,
	}
	return RequestOpenaiAPI(requestBody)
}

func RequestOpenaiApiByStringOneTime(str string) (OpenaiResponse, error) {
	var requestBody = OpenaiRequest{
		Model: model,
		Messages: []Message{
			{
				Role:    "user",
				Content: str,
			},
		},
	}
	return RequestOpenaiAPI(requestBody)
}

func RequestOpenaiApiByMessages(messages []Message) (OpenaiResponse, error) {
	var requestBody = OpenaiRequest{
		Model:    model,
		Messages: messages,
	}
	return RequestOpenaiAPI(requestBody)
}

func RequestOpenaiAPI(requestBody OpenaiRequest) (OpenaiResponse, error) {
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return OpenaiResponse{}, err
	}

	req, err := http.NewRequest("POST", openaiURL, bytes.NewBuffer(requestJSON))
	if err != nil {
		return OpenaiResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return OpenaiResponse{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return OpenaiResponse{}, err
	}

	var response OpenaiResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return OpenaiResponse{}, JsonError
	}

	return response, nil
}

func main() {
	res, err := RequestOpenaiAPIByStrings([]string{"あなたは質問に答えるaiです。", "こんにちは!"})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(res.Choices)
}
