package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/pikachu0310/pikaGPT/api"
	"os"
	"os/signal"
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

var BotToken = GetToken()
var requestContent = []string{SystemRoleMessage}
var SystemRoleMessage string = "あなたはDiscordのTFMJというサーバーのユーザーを楽しませる娯楽用途のBOTの中に作られた、openaiのモデルgpt-3.5-turboを用いた対話型AIです。身内しかいないSNSで、ユーザーに緩く接してください。そして、ユーザーの言う事に出来る限り従うようにしてください。"

func resetRequestContent() {
	requestContent = []string{SystemRoleMessage}
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
			GptReset(s, m)
		} else {
			GptEditMessage(s, m, msg)
		}
	}
}

func Gpt(s *discordgo.Session, m *discordgo.MessageCreate) {
	requestContent = append(requestContent, m.Message.Content)
	res, err := api.RequestOpenaiAPIByStrings(requestContent)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error: %v", err))
	}
	requestContent = append(requestContent, res.Text())
	s.ChannelMessageSend(m.ChannelID, res.Text())
}

func GptEditMessage(s *discordgo.Session, m *discordgo.MessageCreate, message *discordgo.Message) {
	requestContent = append(requestContent, m.Message.Content)
	res, err := api.RequestOpenaiAPIByStrings(requestContent)
	if err != nil {
		s.ChannelMessageEdit(m.ChannelID, message.ID, fmt.Sprintf("error: %v", err))
	}
	if res.OverTokenCheck() {
		res, err = GptDeleteLogsAndRetry(s, m, res, message)
		if err != nil {
			s.ChannelMessageEdit(m.ChannelID, message.ID, fmt.Sprintf("error: %v", err))
		}
	}
	requestContent = append(requestContent, res.Text())
	s.ChannelMessageEdit(m.ChannelID, message.ID, res.Text())
}

func GptDeleteLogsAndRetry(s *discordgo.Session, m *discordgo.MessageCreate, res api.OpenaiResponse, message *discordgo.Message) (api.OpenaiResponse, error) {
	var err error
	for i := 0; res.OverTokenCheck() && i <= 4; i++ {
		s.ChannelMessageEdit(m.ChannelID, message.ID, "Clearing old history and retrying.["+fmt.Sprintf("%d", i+1)+"] :thinking:")
		if len(requestContent) >= 5 {
			requestContent = requestContent[4:]
			requestContent = append([]string{SystemRoleMessage}, requestContent[4:]...)
		} else if len(requestContent) >= 2 {
			requestContent = append([]string{SystemRoleMessage}, requestContent[1:]...)
		} else if len(requestContent) >= 1 {
			requestContent = []string{SystemRoleMessage}
		}
		res, err = api.RequestOpenaiAPIByStrings(requestContent)
		if err != nil {
			s.ChannelMessageEdit(m.ChannelID, message.ID, "Error:"+fmt.Sprint(err))
		}
	}
	return res, err
}

func GptReset(s *discordgo.Session, m *discordgo.MessageCreate) {
	resetRequestContent()
	res, err := api.RequestOpenaiApiByStringOneTime("ユーザーに向けて、<今までの会話履歴を削除し、リセットしました>という旨の文を返してください 謝る必要はありません ダブルクォーテーションも必要ありません")
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error: %v", err))
	}
	s.ChannelMessageSend(m.ChannelID, res.Text())
}

func GptResetEditMessage(s *discordgo.Session, m *discordgo.MessageCreate, message *discordgo.Message) {
	requestContent = append(requestContent, m.Message.Content)
	res, err := api.RequestOpenaiAPIByStrings(requestContent)
	if err != nil {
		s.ChannelMessageEdit(m.ChannelID, message.ID, res.Text())
	}
	requestContent = append(requestContent, res.Text())
	s.ChannelMessageEdit(m.ChannelID, message.ID, res.Text())
}
