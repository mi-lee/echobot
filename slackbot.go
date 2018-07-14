package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

const (
	configJson = "config/config.dev.json"
	MSG_LIMIT  = 950
)

var (
	userMessages    = make(map[string][]Message)
	userIDs         = make(map[string]string)
	slackChannelIDs []string
	usernames       []string
	conf            Config
)

type Config struct {
	BOT_TOKEN string
	API_TOKEN string
}

type UserResponse struct {
	OK      bool     `json:"ok"`
	Members []Member `json:"members"`
}

type ChannelResponse struct {
	OK       bool      `json:"ok"`
	Channels []Channel `json:"channels"`
}

type Channel struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Created    int    `json:"created"`
	IsArchived bool   `json:"is_archived"`
	Creator    string `json:"creator"`
}

type Member struct {
	ID       string `json:"id"`
	TeamID   string `json:"team_id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	RealName string `json:"real_name"`
	Profile  Profile
}

type Profile struct {
	RealName      string `json:"real_name"`
	DisplayName   string `json:"display_name"`
	StatusText    string `json:"status_text"`
	StatusEmoji   string `json:"status_emoji"`
	ImageOriginal string `json:"image_original"`
	Email         string `json:"email"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

type HistoryResponse struct {
	Ok        bool      `json:"ok"`
	Messages  []Message `json:"messages"`
	HasMore   bool      `json:"has_more"`
	IsLimited bool      `json:"is_limited"`
}
type Message struct {
	Type        string     `json:"type"`
	Subtype     string     `json:"subtype"`
	User        string     `json:"user"`
	Text        string     `json:"text"`
	ClientMsgId string     `json:"client_msg_Id"`
	Ts          string     `json:"ts"`
	Reactions   []Reaction `json:"reactions"`
	File        File       `json:"file"`
}

type Reaction struct {
	Name  string   `json:"name"`
	Users []string `json:"users"`
	Count int      `json:"count"`
}

type File struct {
	URLPrivate string `json:"url_private"`
}

type GenericResponse interface {
	filter(word string)
}

func init() {
	getConfig()
	getUsers()
	setUserMessages()
}

func (h *HistoryResponse) filterByUser(userID string) (result []Message) {
	for _, m := range h.Messages {
		if m.User == userID {
			if m.Subtype == "file_share" {
				m.Text = m.Text + " " + m.File.URLPrivate
			}
			result = append(result, m)
		}
	}
	return result
}

func (h *HistoryResponse) filter(word string) {
	for i, m := range h.Messages {
		if strings.Contains(m.Text, word) {
			num := i + 1
			if len(h.Messages) >= num {
				num = i
			}
			h.Messages = append(h.Messages[:i], h.Messages[num:]...)
		}
	}
}

func getUsers() {
	userResp := UserResponse{}
	userEndpoint := fmt.Sprintf("https://slack.com/api/users.list?token=%s&pretty=1", conf.API_TOKEN)
	getResponse(userEndpoint, &userResp)
	err := writeToFile("users", &userResp)
	check(err)
	setUserIDs(&userResp)
}

func getChannels() []string {
	channelResp := ChannelResponse{}
	channelEndpoint := fmt.Sprintf("https://slack.com/api/channels.list?token=%s&pretty=1", conf.API_TOKEN)
	getResponse(channelEndpoint, &channelResp)
	err := writeToFile("channels", &channelResp)
	check(err)

	if channelResp.OK {
		for _, c := range channelResp.Channels {
			slackChannelIDs = append(slackChannelIDs, c.Id)
		}
	}
	return slackChannelIDs
}

func getResponse(url string, v interface{}) {
	resp, err := http.Get(url)
	defer resp.Body.Close()
	check(err)
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, v)
	check(err)
}

func getConfig() {
	dir, err := os.Getwd()
	check(err)
	c, err := ioutil.ReadFile(dir + configJson)
	check(err)
	err = json.Unmarshal([]byte(string(c)), &conf)
	check(err)
}

func setUserIDs(u *UserResponse) {
	if u.OK {
		for _, c := range u.Members {
			name := strings.Split(c.Name, ".")[0]
			userIDs[name] = c.ID
			usernames = append(usernames, strings.ToLower(name))
		}
	}
}

func writeToFile(filename string, v interface{}) (err error) {
	str, err := json.Marshal(v)
	check(err)
	dir, err := os.Getwd()
	check(err)
	file := fmt.Sprintf("%s/data/%s.json", dir, filename)
	err = ioutil.WriteFile(file, str, 0644)
	check(err)
	return err
}

func getChannelHistory(chanID string, h *HistoryResponse) {
	histEndpoint := fmt.Sprintf("https://slack.com/api/channels.history?token=%s&channel=%s&count=%d&pretty=1", conf.API_TOKEN, chanID, MSG_LIMIT)
	getResponse(histEndpoint, h)
	cleanEchobotMsg(h)
	err := writeToFile(chanID, h)
	check(err)
}

func cleanEchobotMsg(v GenericResponse) {
	v.filter(getUserIDs("echobot"))
}

func setUserMessages() {
	chanIDs := getChannels()

	for _, cID := range chanIDs {
		histResp := HistoryResponse{}
		getChannelHistory(cID, &histResp)

		for _, username := range usernames {
			uid := getUserIDs(username)
			result := histResp.filterByUser(uid)
			userMessages[username] = append(result, userMessages[username]...)
		}
	}

	writeToFile("userIDs", userIDs)
	writeToFile("userMessages", userMessages)
	writeToFile("slackChannelIDs", slackChannelIDs)
}

func getUserIDs(name string) (userID string) {
	return userIDs[name]
}

func respond(rtm *slack.RTM, msg *slack.MessageEvent, prefix string) {
	text := msg.Text
	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)

	if userMessages[text] != nil {
		for i := 0; i < 3; i++ {
			length := len(userMessages[text])
			responseIndex := rand.Intn(length)
			str := strings.Split(userMessages[text][responseIndex].Text, " ")
			userMessages[text] = append(userMessages[text][:responseIndex], userMessages[text][responseIndex+1:]...)
			response := strings.Join(str, " ")
			rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
}

func main() {
	api := slack.New(conf.BOT_TOKEN)
	rtm := api.NewRTM()
	go rtm.ManageConnection()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			fmt.Print("-- Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				fmt.Println("-- Connection counter:", ev.ConnectionCount)

			case *slack.MessageEvent:
				//fmt.Printf("-- Message: %v\n", ev)
				info := rtm.GetInfo()
				prefix := fmt.Sprintf("<@%s> ", info.User.ID)

				if ev.User != info.User.ID && strings.HasPrefix(ev.Text, prefix) {
					respond(rtm, ev, prefix)
				}

			case *slack.RTMError:
				fmt.Printf("-- Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Println("-- Invalid credentials")
				break Loop

			default:
				// Take no action
			}
		}
	}
}

func check(e error) {
	if e != nil {
		fmt.Printf("-- e = %v \n", e)
		panic(e)
	}
}
