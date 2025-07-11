package vmpoll

import (
	"context"
	"fmt"
	"time"

	"github.com/platform9/pcd-vm-saver/pkg/openstack"
	"go.uber.org/zap"
)

func AutoAwakeVM() (string, error) {
	zap.S().Infof("Triggering auto awake VMs")
	ctx := context.TODO()

	// Fetch all VMs to Awake
	awakeVms := openstack.GetVMsToAwake(ctx)

	if len(awakeVms) == 0 {
		zap.S().Info("No VMs found to awake")
		return "No VMs found to awake", nil
	}

	// Awake by SleepMode UnShelve or Resume
	openstack.AwakeVMs(ctx, awakeVms)

	time.Sleep(25 * time.Second) // Adding a minimum wait time to ensure VMs are awake

	// Build success message with list of awakened VMs
	successMsg := "AutoAwakeVM task completed successfully\n\n"
	successMsg += "List of VMs awakened:\n"

	for _, vm := range awakeVms {
		vmStatus := openstack.GetVMStatus(ctx, vm.ID)
		successMsg += fmt.Sprintf("VM %s (ID: %s) - Current state: %s\n", vm.Name, vm.ID, vmStatus.Status)
	}

	return successMsg, nil
}
