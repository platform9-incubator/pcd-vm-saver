package slack

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type SlackClient struct {
	client *slack.Client
	sm     *socketmode.Client
	botID  string
	ctx    context.Context
	cancel context.CancelFunc
}

func NewSlackClient(appToken, botToken string) (*SlackClient, error) {
	client := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
		slack.OptionDebug(true), // optional: remove in production
	)
	sm := socketmode.New(client)

	// Get bot ID
	authResp, err := client.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Slack: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SlackClient{
		client: client,
		sm:     sm,
		botID:  authResp.UserID,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (s *SlackClient) Start() {
	go func() {
		err := s.sm.Run()
		if err != nil {
			log.Printf("Socketmode run error: %v", err)
		}
	}()
}

func (s *SlackClient) SendMessage(channelID, message string) error {
	_, _, err := s.client.PostMessage(channelID, slack.MsgOptionText(message, false))
	return err
}

// SendNotification sends a formatted notification to a specific channel
func (s *SlackClient) SendNotification(channelID, status, message string) error {
	var color string
	var emoji string

	switch status {
	case "success":
		color = "#2eb886"
		emoji = "✅"
	case "failure":
		color = "#ff0000"
		emoji = "❌"
	case "done":
		color = "#666666"
		emoji = "✅"
	default:
		color = "#666666"
		emoji = "ℹ️"
	}

	attachment := slack.Attachment{
		Color:      color,
		Text:       message,
		MarkdownIn: []string{"text"},
		FooterIcon: "https://platform9.io/favicon.ico",
	}

	_, _, err := s.client.PostMessage(
		channelID,
		slack.MsgOptionText(fmt.Sprintf("%s %s", emoji, status), false),
		slack.MsgOptionAttachments(attachment),
	)
	return err
}

func (s *SlackClient) ListenForMentions() {
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				log.Println("Shutting down Slack listener...")
				return

			case event := <-s.sm.Events:
				switch ev := event.Data.(type) {

				case *slackevents.MessageEvent:
					// Unused unless processing raw events via Events API

				case *slack.MessageEvent:
					if ev.BotID != "" || ev.SubType == "bot_message" {
						continue
					}

					// Check if bot is mentioned
					if strings.Contains(ev.Text, fmt.Sprintf("<@%s>", s.botID)) {
						err := s.SendMessage(ev.Channel, "👋 Hello! I see you mentioned me. How can I help?")
						if err != nil {
							log.Printf("Failed to send response: %v", err)
						}
					}

				case *socketmode.Event:
					if ev.Type == socketmode.EventTypeInteractive {
						s.sm.Ack(*ev.Request) // Acknowledge interactive event if needed
					}
				}
			}
		}
	}()
}

func (s *SlackClient) Stop() {
	log.Println("Stopping Slack client...")
	s.cancel() // This will stop the event listener loop
}
