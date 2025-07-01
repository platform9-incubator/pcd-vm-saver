package vmpoll

import (
	"context"
	"fmt"

	"github.com/platform9/pcd-vm-saver/pkg/openstack"
	"go.uber.org/zap"
)

func AutoAwakeVM() (string, error) {
	zap.S().Infof("Triggering auto awake VMs")
	ctx := context.TODO()

	// Fetch all VMs to Awake
	awakeVms := openstack.GetVMsToAwake(ctx)

	// Awake by SleepMode UnShelve or Resume
	openstack.AwakeVMs(ctx, awakeVms)

	// Build success message with list of awakened VMs
	successMsg := "AutoAwakeVM task completed successfully\n\n"
	successMsg += "List of VMs awakened:\n"

	for _, vm := range awakeVms {
		vmStatus := openstack.GetVMStatus(ctx, vm.ID)
		successMsg += fmt.Sprintf("VM %s (ID: %s) - Current state: %s\n", vm.Name, vm.ID, vmStatus.Status)
	}

	return successMsg, nil
}
