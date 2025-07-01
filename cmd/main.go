package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/platform9/pcd-vm-saver/pkg/log"
	"github.com/platform9/pcd-vm-saver/pkg/slack"
	"github.com/platform9/pcd-vm-saver/pkg/util"
	"github.com/platform9/pcd-vm-saver/pkg/vmpoll"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type CronSkipperLogger struct{}

func (c *CronSkipperLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	zap.S().Error(err)
	zap.S().Errorw(msg, keysAndValues...)
}

func (c *CronSkipperLogger) Info(msg string, keysAndValues ...interface{}) {
	zap.S().Infow(msg, keysAndValues...)
}

func run(cmd *cobra.Command, args []string) {
	zap.S().Info("Starting pcd-vm-saver...")
	zap.S().Infof("Version of pcd-vm-saver being used is: %s", util.Version)
	zap.S().Info("starting scheduled tasks")

	// Initialize Slack client
	appToken := os.Getenv("SLACK_APP_TOKEN")
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if appToken == "" || botToken == "" {
		zap.S().Warn("Slack tokens not found in environment variables. Slack integration will be disabled.")
	} else {
		client, err := slack.NewSlackClient(appToken, botToken)
		if err != nil {
			zap.S().Errorf("Failed to initialize Slack client: %v", err)
		} else {
			client.Start()
			client.ListenForMentions()
			
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			cmd.SetContext(context.WithValue(ctx, "slackClient", client))
		}
	}

	// Create schedule
	schedule := cron.New(cron.WithChain(cron.SkipIfStillRunning(&CronSkipperLogger{})))
	schedule.AddFunc("@every 1m", func() {
		ctx := cmd.Context()
		if ctx == nil {
			zap.S().Warn("No context available, skipping Slack notification")
			return
		}

		client, ok := ctx.Value("slackClient").(*slack.SlackClient)
		if !ok {
			zap.S().Warn("Slack client not found in context, skipping notification")
			return
		}

		channelID := os.Getenv("SLACK_CHANNEL_ID")
		if channelID == "" {
			zap.S().Warn("SLACK_CHANNEL_ID not set, skipping notification")
			return
		}

		client.SendNotification(channelID, "info", "Starting AutoSleepVM task...")
		
		if err := vmpoll.AutoSleepVM(); err != nil {
			client.SendNotification(channelID, "failure", "AutoSleepVM task failed: "+err.Error())
			zap.S().Errorf("AutoSleepVM failed: %v", err)
		} else {
			client.SendNotification(channelID, "success", "AutoSleepVM task completed successfully")
			zap.S().Info("AutoSleepVM completed successfully")
		}
	})
	schedule.AddFunc("@every 2m", func() {
		ctx := cmd.Context()
		if ctx == nil {
			zap.S().Warn("No context available, skipping Slack notification")
			return
		}

		client, ok := ctx.Value("slackClient").(*slack.SlackClient)
		if !ok {
			zap.S().Warn("Slack client not found in context, skipping notification")
			return
		}

		channelID := os.Getenv("SLACK_CHANNEL_ID")
		if channelID == "" {
			zap.S().Warn("SLACK_CHANNEL_ID not set, skipping notification")
			return
		}

		client.SendNotification(channelID, "info", "Starting AutoAwakeVM task...")
		
		if err := vmpoll.AutoAwakeVM(); err != nil {
			client.SendNotification(channelID, "failure", "AutoAwakeVM task failed: "+err.Error())
			zap.S().Errorf("AutoAwakeVM failed: %v", err)
		} else {
			client.SendNotification(channelID, "success", "AutoAwakeVM task completed successfully")
			zap.S().Info("AutoAwakeVM completed successfully")
		}
	})
	schedule.Start()
	zap.S().Info("cron jobs scheduled")

	// Wrap the scheduled functions with notification handling
	schedule.AddFunc("@every 1m", func() {
		ctx := cmd.Context()
		if ctx == nil {
			zap.S().Warn("No context available, skipping Slack notification")
			return
		}

		client, ok := ctx.Value("slackClient").(*slack.SlackClient)
		if !ok {
			zap.S().Warn("Slack client not found in context, skipping notification")
			return
		}

		channelID := os.Getenv("SLACK_CHANNEL_ID")
		if channelID == "" {
			zap.S().Warn("SLACK_CHANNEL_ID not set, skipping notification")
			return
		}

		client.SendNotification(channelID, "info", "Starting AutoSleepVM task...")
		
		if err := vmpoll.AutoSleepVM(); err != nil {
			client.SendNotification(channelID, "failure", "AutoSleepVM task failed: "+err.Error())
			zap.S().Errorf("AutoSleepVM failed: %v", err)
		} else {
			client.SendNotification(channelID, "success", "AutoSleepVM task completed successfully")
			zap.S().Info("AutoSleepVM completed successfully")
		}
	})

	schedule.AddFunc("@every 2m", func() {
		ctx := cmd.Context()
		if ctx == nil {
			zap.S().Warn("No context available, skipping Slack notification")
			return
		}

		client, ok := ctx.Value("slackClient").(*slack.SlackClient)
		if !ok {
			zap.S().Warn("Slack client not found in context, skipping notification")
			return
		}

		channelID := os.Getenv("SLACK_CHANNEL_ID")
		if channelID == "" {
			zap.S().Warn("SLACK_CHANNEL_ID not set, skipping notification")
			return
		}

		client.SendNotification(channelID, "info", "Starting AutoAwakeVM task...")
		
		if err := vmpoll.AutoAwakeVM(); err != nil {
			client.SendNotification(channelID, "failure", "AutoAwakeVM task failed: "+err.Error())
			zap.S().Errorf("AutoAwakeVM failed: %v", err)
		} else {
			client.SendNotification(channelID, "success", "AutoAwakeVM task completed successfully")
			zap.S().Info("AutoAwakeVM completed successfully")
		}
	})

	zap.S().Info("pcd-vm-saver is running")
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		zap.S().Info("server stopping...")
		schedule.Stop()
	}
}

func main() {
	cmd := buildCmds()
	cmd.Execute()
}

func buildCmds() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pcd-vm-saver",
		Short: "pcd-vm-saver helps handling VMs efficiently by hibernating and awaking them",
		Long:  "pcd-vm-saver helps handling VMs efficiently by hibernating and awaking them",
		Run:   run,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Current version of pcd-vm-saver being used",
		Long:  "Current version of pcd-vm-saver being used",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(util.Version)
		},
	}

	rootCmd.AddCommand(versionCmd)
	return rootCmd
}

func init() {
	err := log.Logger()
	if err != nil {
		fmt.Printf("Failed to initiate logger, Error is: %s", err)
	}
}
