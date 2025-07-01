package vmpoll

import (
	"context"

	"github.com/platform9/pcd-vm-saver/pkg/openstack"
	"go.uber.org/zap"
)

func AutoAwakeVM() {
	zap.S().Infof("Triggering auto awake VMs")
	ctx := context.TODO()

	// Fetch all VMs to Awake
	awakeVms := openstack.GetVMsToAwake(ctx)

	// Awake by SleepMode UnShelve or Resume
	openstack.AwakeVMs(ctx, awakeVms)

	// TOOD: Integrate list of VMs awakened with Slack notification
}
