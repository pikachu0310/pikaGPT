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
		if strings.Contains(m.Message.Content, "/gpt reset") || strings.Contains(m.Message.Content, "/gpt new") {
			GptResetEditMessage(s, m, msg)
		} else if strings.Contains(m.Message.Content, "/gpt debug") || strings.Contains(m.Message.Content, "/gptdebug") {
			GptDebugEditMessage(s, m, msg)
		} else {
			m.Message.Content = regexp.MustCompile("/gpt").ReplaceAllString(m.Message.Content, "")
			GptEditMessage(s, m, msg)
		}
	}
}

func Gpt(s *discordgo.Session, m *discordgo.MessageCreate) {
	addRequestContent("user", m.Message.Content)
	res, err := api.RequestOpenaiApiByMessages(requestContent)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error: %v", err))
	}
	addRequestContent("assistant", res.Text())
	responses = append(responses, res)
	s.ChannelMessageSend(m.ChannelID, res.Text())
}

func GptEditMessage(s *discordgo.Session, m *discordgo.MessageCreate, message *discordgo.Message) {
	addRequestContent("user", m.Message.Content)
	res, err := api.RequestOpenaiApiByMessages(requestContent)
	if err != nil {
		s.ChannelMessageEdit(m.ChannelID, message.ID, fmt.Sprintf("error: %v", err))
	}
	if res.OverTokenCheck() {
		res, err = GptDeleteLogsAndRetry(s, m, res, message)
		if err != nil {
			s.ChannelMessageEdit(m.ChannelID, message.ID, fmt.Sprintf("error: %v", err))
		}
	}
	addRequestContent("assistant", res.Text())
	responses = append(responses, res)
	s.ChannelMessageEdit(m.ChannelID, message.ID, res.Text())
}

func GptDeleteLogsAndRetry(s *discordgo.Session, m *discordgo.MessageCreate, res api.OpenaiResponse, message *discordgo.Message) (api.OpenaiResponse, error) {
	var err error
	for i := 0; res.OverTokenCheck() && i <= 4; i++ {
		s.ChannelMessageEdit(m.ChannelID, message.ID, "Clearing old history and retrying.["+fmt.Sprintf("%d", i+1)+"] :thinking:")
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
			s.ChannelMessageEdit(m.ChannelID, message.ID, "Error:"+fmt.Sprint(err))
		}
	}
	return res, err
}

func GptReset(s *discordgo.Session, m *discordgo.MessageCreate) {
	resetRequestContent()
	resetResponses()
	res, err := api.RequestOpenaiApiByStringOneTime(ResetMessage)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error: %v", err))
	}
	s.ChannelMessageSend(m.ChannelID, res.Text())
}

func GptResetEditMessage(s *discordgo.Session, m *discordgo.MessageCreate, message *discordgo.Message) {
	resetRequestContent()
	res, err := api.RequestOpenaiApiByStringOneTime(ResetMessage)
	if err != nil {
		s.ChannelMessageEdit(m.ChannelID, message.ID, fmt.Sprintf("error: %v", err))
	}
	s.ChannelMessageEdit(m.ChannelID, message.ID, res.Text())
}

func Sum(arr []float32) float32 {
	var res float32 = 0
	for i := 0; i < len(arr); i++ {
		res += arr[i]
	}
	return res
}

func GptDebug(s *discordgo.Session, m *discordgo.MessageCreate) {
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
			prices = append(prices, float32(r.Usage.PromptTokens)*(131.34/1000)*0.03+float32(r.Usage.CompletionTokens)*(131.34/1000)*0.06)
			continue
		} else if strings.Contains(r.Model, "gpt-3.5") {
			prices = append(prices, float32(r.Usage.TotalTokens)*(131.34/1000)*0.002)
			continue
		}
	}
	if len(responses) == 0 {
		s.ChannelMessageSend(m.ChannelID, "まだ会話がありません")
		return
	}
	r := responses[len(responses)-1]
	var price float32
	if strings.Contains(r.Model, "gpt-4") {
		price = float32(r.Usage.PromptTokens)*(131.34/1000)*0.03 + float32(r.Usage.CompletionTokens)*(131.34/1000)*0.06
	} else if strings.Contains(r.Model, "gpt-3.5") {
		price = float32(r.Usage.TotalTokens) * (131.34 / 1000) * 0.002
	}

	returnString += fmt.Sprintf("PromptTokens: %d\nCompletionTokens: %d\nTotalTokens: %d\n最後の一回で使った金額: %.2f円\n最後にリセットされてから使った合計金額:  %.2f円\n", r.Usage.PromptTokens, r.Usage.CompletionTokens, r.Usage.TotalTokens, price, Sum(prices))
	returnString += "```"
	s.ChannelMessageSend(m.ChannelID, returnString)
}

func GptDebugEditMessage(s *discordgo.Session, m *discordgo.MessageCreate, message *discordgo.Message) {
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
			prices = append(prices, float32(r.Usage.PromptTokens)*(131.34/1000)*0.03+float32(r.Usage.CompletionTokens)*(131.34/1000)*0.06)
			continue
		} else if strings.Contains(r.Model, "gpt-3.5") {
			prices = append(prices, float32(r.Usage.TotalTokens)*(131.34/1000)*0.002)
			continue
		}
	}
	if len(responses) == 0 {
		s.ChannelMessageEdit(m.ChannelID, message.ID, "まだ会話がありません")
		return
	}
	r := responses[len(responses)-1]
	var price float32
	if strings.Contains(r.Model, "gpt-4") {
		price = float32(r.Usage.PromptTokens)*(131.34/1000)*0.03 + float32(r.Usage.CompletionTokens)*(131.34/1000)*0.06
	} else if strings.Contains(r.Model, "gpt-3.5") {
		price = float32(r.Usage.TotalTokens) * (131.34 / 1000) * 0.002
	}

	returnString += fmt.Sprintf("PromptTokens: %d\nCompletionTokens: %d\nTotalTokens: %d\n最後の一回で使った金額: %.2f円\n最後にリセットされてから使った合計金額:  %.2f円\n", r.Usage.PromptTokens, r.Usage.CompletionTokens, r.Usage.TotalTokens, price, Sum(prices))
	returnString += "```"
	s.ChannelMessageEdit(m.ChannelID, message.ID, returnString)
}
