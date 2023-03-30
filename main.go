package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/pikachu0310/pikaGPT/api"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
)

func GetToken() (token string) {
	var err = godotenv.Load(".env")
	if err != nil {
		fmt.Printf("error: tokenが読み込めなかった!: %v", err)
	}
	token = os.Getenv("TOKEN")
	return token
}

var (
	BotToken          = GetToken()
	requestContent    = []api.Message{firstMessage}
	responses         []api.OpenaiResponse
	SystemRoleMessage string = "あなたはDiscordのTFMJというサーバーのユーザーを楽しませる娯楽用途のBOTの中に作られた、openaiのモデルgpt-3.5-turboを用いた対話型AIです。身内しかいないSNSで、ユーザーに緩く接してください。そして、ユーザーの言う事に出来る限り従うようにしてください。"
	ResetMessage             = "ユーザーに向けて、<今までの会話履歴を削除し、リセットしました>という旨の文を返してください 謝る必要はありません ダブルクォーテーションも必要ありません"
	firstMessage             = api.Message{
		Role:    "system",
		Content: SystemRoleMessage,
	}
	DollToYen float32 = 132.54
)

func resetRequestContent() {
	requestContent = []api.Message{firstMessage}
}

func resetResponses() {
	responses = []api.OpenaiResponse{}
}

func addRequestContent(role string, content string) {
	var message api.Message
	message.Role = role
	message.Content = content
	requestContent = append(requestContent, message)
}

func main() {
	discord, err := discordgo.New("Bot " + BotToken)
	if err != nil {
		panic(err)
	}

	discord.AddHandler(onMessageCreate)

	fmt.Println("DONE!")

	err = discord.Open()

	stopBot := make(chan os.Signal, 1)

	signal.Notify(stopBot, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	<-stopBot

	err = discord.Close()

	return
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	fmt.Printf("Message: %s, Author: %s", m.Message.Content, m.Author.Username)

	if m.Message.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "pong")
	}

	if strings.Contains(m.Message.Content, "/gpt") {
		msg, err := s.ChannelMessageSend(m.ChannelID, ":thinking:")
		if err != nil {
			fmt.Println(err)
			return
		}
		EditMessage := func(content string) {
			s.ChannelMessageEdit(m.ChannelID, msg.ID, content)
		}
		if strings.Contains(m.Message.Content, "/gpt reset") || strings.Contains(m.Message.Content, "/gpt new") {
			GptReset(EditMessage)
		} else if strings.Contains(m.Message.Content, "/gpt debug") || strings.Contains(m.Message.Content, "/gptdebug") {
			GptDebug(EditMessage)
		} else {
			Gpt(regexp.MustCompile("/gpt").ReplaceAllString(m.Message.Content, ""), EditMessage)
		}
	}
}

func SendOrEditError(SendOrEdit func(string), err error) {
	SendOrEdit(fmt.Sprintf("error: %v", err))
}

func Gpt(content string, SendOrEdit func(string)) (api.OpenaiResponse, error) {
	addRequestContent("user", content)
	res, err := api.RequestOpenaiApiByMessages(requestContent)
	if err != nil {
		SendOrEditError(SendOrEdit, err)
		return res, err
	}
	res, err = GptDeleteLogsAndRetry(res, SendOrEdit)
	if err != nil {
		SendOrEditError(SendOrEdit, err)
		return res, err
	}
	addRequestContent("assistant", res.Text())
	responses = append(responses, res)
	SendOrEdit(res.Text())
	return res, err
}

func GptDeleteLogsAndRetry(res api.OpenaiResponse, SendOrEdit func(string)) (api.OpenaiResponse, error) {
	var err error
	for i := 0; res.OverTokenCheck() && i <= 4; i++ {
		SendOrEdit("Clearing old history and retrying.[" + fmt.Sprintf("%d", i+1) + "] :thinking:")
		if len(requestContent) >= 5 {
			requestContent = requestContent[4:]
			requestContent = append([]api.Message{firstMessage}, requestContent[4:]...)
		} else if len(requestContent) >= 2 {
			requestContent = append([]api.Message{firstMessage}, requestContent[1:]...)
		} else if len(requestContent) >= 1 {
			requestContent = []api.Message{firstMessage}
		}
		res, err = api.RequestOpenaiApiByMessages(requestContent)
		if err != nil {
			SendOrEditError(SendOrEdit, err)
		}
	}
	return res, err
}

func GptReset(SendOrEdit func(string)) (api.OpenaiResponse, error) {
	resetRequestContent()
	//resetResponses()
	res, err := api.RequestOpenaiApiByStringOneTime(ResetMessage)
	if err != nil {
		SendOrEditError(SendOrEdit, err)
		return res, err
	}
	SendOrEdit(res.Text())
	return res, err
}

func Sum(arr []float32) float32 {
	var res float32 = 0
	for i := 0; i < len(arr); i++ {
		res += arr[i]
	}
	return res
}

func GptDebug(SendOrEdit func(string)) {
	returnString := "```\n"
	for _, message := range requestContent {
		chatText := regexp.MustCompile("```").ReplaceAllString(message.Content, "")
		if len(chatText) >= 40 {
			returnString += message.Role + ": " + chatText[:40] + "...\n"
		} else {
			returnString += message.Role + ": " + chatText + "\n"
		}
	}
	returnString += "```\n```\n"
	var prices []float32
	for _, r := range responses {
		if strings.Contains(r.Model, "gpt-4") {
			prices = append(prices, float32(r.Usage.PromptTokens)*(DollToYen/1000)*0.03+float32(r.Usage.CompletionTokens)*(131.34/1000)*0.06)
		} else if strings.Contains(r.Model, "gpt-3.5") {
			prices = append(prices, float32(r.Usage.TotalTokens)*(DollToYen/1000)*0.002)
		}
	}
	if len(responses) == 0 || len(prices) == 0 {
		SendOrEdit("まだ会話がありません")
		return
	}
	r := responses[len(responses)-1]
	returnString += fmt.Sprintf("PromptTokens: %d\nCompletionTokens: %d\nTotalTokens: %d\n最後の一回で使った金額: %.2f円\n最後にリセットされてから使った合計金額:  %.2f円\n```", r.Usage.PromptTokens, r.Usage.CompletionTokens, r.Usage.TotalTokens, prices[len(prices)-1], Sum(prices))
	SendOrEdit(returnString)
}
