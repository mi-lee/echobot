package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"

	"time"

	"github.com/nlopes/slack"
)

type reaction struct {
	Name  string   `json:"name"`
	Users []string `json:"users"`
	Count int      `json:"count"`
}

type message struct {
	Type        string     `json:"type"`
	User        string     `json:"user"`
	Text        string     `json:"text"`
	ClientMsgId string     `json:"client_msg_Id"`
	Ts          string     `json:"ts"`
	Reactions   []reaction `json:"reactions"`
}

type response struct {
	Ok        bool      `json:"ok"`
	Messages  []message `json:"messages"`
	HasMore   bool      `json:"has_more"`
	IsLimited bool      `json:"is_limited"`
}

const (
	dir = "/Users/michelle/Dropbox/Work/playground/slackbot/"
)

var (
	userMessages  = make(map[string][]message)
	userIDs       = make(map[string]string)
	slackChannels = []string{"random.json", "gettingjacked.json", "kenny.json", "thoughts.json", "me_irl.json", "project.json", "tolearn.json", "discuss.json"}
	usernames     = []string{"sonal", "alex", "kenny", "mac", "ado", "michelle"}
)

func init() {
	setUserIDs()
	setUserMessages()
}

func check(e error) {
	if e != nil {
		fmt.Printf("-- e = %v \n", e)
		panic(e)
	}
}

func setUserIDs() {
	userIDs["sonal"] = "U18VAQPB9"
	userIDs["alex"] = "U18V2LAAJ"
	userIDs["kenny"] = "U9APJ3XKN"
	userIDs["mac"] = "U1908M6PL"
	userIDs["ado"] = "U1XRS5MCM"
	userIDs["michelle"] = "U1X4W9U3V"
}

func setUserMessages() {
	for _, slackChannel := range slackChannels {
		dat, err := ioutil.ReadFile(dir + slackChannel)
		check(err)

		response := response{}
		err = json.Unmarshal([]byte(string(dat)), &response)
		check(err)

		for _, username := range usernames {
			uid := getUserIDs(username)
			result := response.filterByUser(uid)
			userMessages[username] = append(result, userMessages[username]...)
		}
	}
}

func getUserIDs(name string) (userID string) {
	return userIDs[name]
}

func (r *response) filterByUser(userID string) (result []message) {
	for _, m := range r.Messages {
		if m.User == userID {
			result = append(result, m)
		}
	}
	return result
}

func respond(rtm *slack.RTM, msg *slack.MessageEvent, prefix string) {
	text := msg.Text
	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)

	if userMessages[text] != nil {
		for i := range []int{1, 2, 3} {
			fmt.Printf("i = %v \n", i)
			length := len(userMessages[text])
			fmt.Printf("length = %v \n", length)
			responseIndex := rand.Intn(length)
			fmt.Printf("responseIndex = %v \n", responseIndex)
			str := strings.Split(userMessages[text][responseIndex].Text, " ")
			userMessages[text] = append(userMessages[text][:responseIndex], userMessages[text][responseIndex+1:]...)
			response := strings.Join(str, " ")
			fmt.Printf("response = %v \n", response)
			rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
			time.Sleep(time.Duration(1) * time.Second)
		}
	}

}

func main() {
	token := os.Getenv("SLACK_TOKEN")
	api := slack.New(token)
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
