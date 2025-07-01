package vmpoll

import (
	"context"
	"fmt"

	"github.com/platform9/pcd-vm-saver/pkg/openstack"
)

func AutoAwakeVM() (string, error) {
	ctx := context.TODO()
	fmt.Println("Starting AutoAwakeVM process")

	// Fetch all VMs to Awake
	awakeVms := openstack.GetVMsToAwake(ctx)

	// Awake by SleepMode UnShelve or Resume
	openstack.AwakeVMs(ctx, awakeVms)

	// Create success message
	successMsg := fmt.Sprintf("AutoAwakeVM task completed successfully")
	return successMsg, nil
}
