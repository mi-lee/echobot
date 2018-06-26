// fetch messages by user
// find one with most reacted to, or longest

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
	random        = "random.json"
	gettingjacked = "gettingjacked.json"
	kenny         = "kenny.json"
	project       = "project.json"
	thoughts      = "thoughts.json"
	meirl         = "me_irl.json"
	discuss       = "discuss.json"
	tolearn       = "tolearn.json"
	dir           = "/Users/michelle/Dropbox/Work/playground/slackbot/"
)

func check(e error) {
	if e != nil {
		fmt.Printf("-- e = %v \n", e)
		panic(e)
	}
}

func init() {
	setUserIDs()
}

func main() {
	for _, slackchan := range []string{random, gettingjacked, kenny, thoughts, meirl, project, tolearn, discuss} {
		dat, err := ioutil.ReadFile(dir + slackchan)
		check(err)

		response := response{}
		err = json.Unmarshal([]byte(string(dat)), &response)
		check(err)

		for _, username := range []string{"sonal", "alex", "kenny", "mac", "ado", "michelle"} {
			uid := getUserIDs(username)
			result := response.filterByUser(uid)
			userMessages[username] = append(result, userMessages[username]...)
		}
	}

	theRest()
}

var userIDs = make(map[string]string)
var userMessages = make(map[string][]message)

func getUserIDs(name string) (userID string) {
	return userIDs[name]
}

func setUserIDs() {
	userIDs["sonal"] = "U18VAQPB9"
	userIDs["alex"] = "U18V2LAAJ"
	userIDs["kenny"] = "U9APJ3XKN"
	userIDs["mac"] = "U1908M6PL"
	userIDs["ado"] = "U1XRS5MCM"
	userIDs["michelle"] = "U1X4W9U3V"
}

//func setUserMessages(username string) {
//	uid := getUserIDs(username)
//	result := response.filterByUser(uid)
//	userMessages[username] = result
//}

func (r *response) filterByUser(userID string) (result []message) {
	for _, m := range r.Messages {
		if m.User == userID {
			result = append(result, m)
		}
	}
	return result
}

func theRest() {

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
				fmt.Printf("-- Message: %v\n", ev)
				info := rtm.GetInfo()
				prefix := fmt.Sprintf("<@%s> ", info.User.ID)

				if ev.User != info.User.ID && strings.HasPrefix(ev.Text, prefix) {
					respond(rtm, ev, prefix)
				}

			case *slack.RTMError:
				fmt.Printf("-- Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("-- Invalid credentials")
				break Loop

			default:
				//Take no action
			}
		}
	}
}

func respond(rtm *slack.RTM, msg *slack.MessageEvent, prefix string) {
	text := msg.Text
	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)

	if userMessages[text] != nil {
		for i := range []int{1, 2} {
			fmt.Printf("i = %v \n", i)
			length := len(userMessages[text])
			fmt.Printf("length = %v \n", length)
			responseIndex := rand.Intn(length)
			fmt.Printf("responseIndex = %v \n", responseIndex)
			response := userMessages[text][responseIndex].Text
			rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
			time.Sleep(time.Duration(1) * time.Second)
		}
	}

}
