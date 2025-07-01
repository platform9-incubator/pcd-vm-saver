package vmpoll

import (
	"context"
	"fmt"

	"github.com/platform9/pcd-vm-saver/pkg/openstack"
)

func AutoAwakeVM() {
	fmt.Println("logic of auto awake vm to implement")
	ctx := context.TODO()

	// Fetch all VMs to Awake
	awakeVms := openstack.GetVMsToAwake(ctx)

	// Awake by SleepMode UnShelve or Resume
	openstack.AwakeVMs(ctx, awakeVms)

	// TOOD: Integrate list of VMs awakened with Slack notification
}
