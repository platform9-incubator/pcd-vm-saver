package vmpoll

import (
	"fmt"
)

func AutoAwakeVM() {
	fmt.Println("logic of auto awake vm to implement")
	//ctx := context.TODO()
	// Fetch all VMs
	//awakeVms := openstack.GetVMsToAwake(ctx)

	// Metadata AwakeTimestamp
	// If timestamp is before the current time
	// Awake by SleepMode UnShelve or Resume

}
