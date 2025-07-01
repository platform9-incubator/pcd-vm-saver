package vmpoll

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platform9/pcd-vm-saver/pkg/openstack"
	"github.com/platform9/pcd-vm-saver/pkg/slack"
	"go.uber.org/zap"
)

func AutoSleepVM() error {
	ctx := context.TODO()
	fmt.Println("Starting AutoSleepVM process")

	// Get Slack client from context
	client, ok := ctx.Value("slackClient").(*slack.SlackClient)
	if !ok {
		zap.S().Warn("Slack client not found in context, skipping notification")
	}

	// 1. Fetch available list of VMs with Default Sleep Filter
	// Testing - Done
	serversInfo, err := openstack.FetchVMsToSleep(ctx)
	if err != nil {
		return err
	}

	// 2. Fetch current quotas
	// TODO: Add this - P1 cc: @shweta

	// 3. Parallely Shelve/Suspend all the VMs
	// Testing - Done
	err = openstack.SleepVMs(ctx, serversInfo)
	if err != nil {
		return err
	}

	// Adding a minimum time wait
	time.Sleep(15 * time.Second)

	// 4. Fetch the status and generate the cumulative shelve VM status
	var successVMs []string
	var failedVMs []string
	var statusMessage strings.Builder
	
	for _, server := range serversInfo {
		sleepState := openstack.GetVMStaus(ctx, server.ID)
		statusMessage.WriteString(fmt.Sprintf("*VM:* %s\n", server.Name))
		statusMessage.WriteString(fmt.Sprintf("*Suspend Mode:* %t\n", server.SuspendMode))
		statusMessage.WriteString(fmt.Sprintf("*Current Status:* %s\n\n", sleepState.Status))
		
		if sleepState.Status == "SHELVED" || sleepState.Status == "SUSPENDED" {
			successVMs = append(successVMs, server.Name)
		} else {
			failedVMs = append(failedVMs, server.Name)
		}
	}

	// Send Slack notifications
	channelID := os.Getenv("SLACK_CHANNEL_ID")
	if channelID == "" {
		zap.S().Error("SLACK_CHANNEL_ID not set")
		return fmt.Errorf("SLACK_CHANNEL_ID not set")
	}

	// Send success notification
	successMsg := fmt.Sprintf("*VM Sleep Operation Summary*\n\n")
	successMsg += fmt.Sprintf("*Total VMs:* %d\n", len(serversInfo))
	successMsg += fmt.Sprintf("*Successfully Hibernated:* %d\n", len(successVMs))
	successMsg += fmt.Sprintf("*Failed:* %d\n", len(failedVMs))

	if len(successVMs) > 0 {
		successMsg += "\n*Successfully Hibernated VMs:*\n"
		successMsg += strings.Join(successVMs, "\n")
	}

	if len(failedVMs) > 0 {
		successMsg += "\n*Failed VMs:*\n"
		successMsg += strings.Join(failedVMs, "\n")
	}

	if client != nil {
		err = client.SendNotification(channelID, "done", successMsg)
		if err != nil {
			zap.S().Errorf("Failed to send Slack notification: %v", err)
			return fmt.Errorf("failed to send Slack notification: %v", err)
		}
	}

	// Send detailed status notification
	detailsMsg := statusMessage.String()
	if client != nil {
		err = client.SendNotification(channelID, "done", detailsMsg)
		if err != nil {
			zap.S().Errorf("Failed to send detailed Slack notification: %v", err)
			return fmt.Errorf("failed to send detailed Slack notification: %v", err)
		}
	}

	return nil
}
