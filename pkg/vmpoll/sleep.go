package vmpoll

import (
	"context"
	"fmt"

	"github.com/platform9/pcd-vm-saver/pkg/openstack"
)

func AutoSleepVM() (string, error) {
	ctx := context.TODO()
	fmt.Println("Starting AutoSleepVM process")

	// Fetch all VMs to Sleep
	serversInfo, err := openstack.FetchVMsToSleep(ctx)
	if err != nil {
		return "", err
	}

	// Sleep VMs
	err = openstack.SleepVMs(ctx, serversInfo)
	if err != nil {
		return "", err
	}

	// Create success message
	successMsg := fmt.Sprintf("AutoSleepVM task completed successfully")
	return successMsg, nil
}
