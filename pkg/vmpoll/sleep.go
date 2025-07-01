package vmpoll

import (
	"context"
	"fmt"
	"time"

	"github.com/platform9/pcd-vm-saver/pkg/openstack"
)

func AutoSleepVM() {
	ctx := context.TODO()
	fmt.Println("logic of auto sleep vm to implement")

	// 1. Fetch available list of VMs with Default Sleep Filter
	// Testing - Done
	serversInfo := openstack.FetchVMsToSleep(ctx)

	// 2. Fetch current quotas
	// TODO: Add this - P1 cc: @shweta

	// 3. Parallely Shelve/Suspend all the VMs
	// Testing - Done
	openstack.SleepVMs(ctx, serversInfo)

	// Adding a minimum time wait
	time.Sleep(15 * time.Second)

	// 4. Fetch the status and generate the cumulative shelve VM status
	// TODO: We need to wait for the status. I think we need to wait till all VMs are hibernated to get full freeup resources
	// Testing - Done
	for _, server := range serversInfo {
		sleepState := openstack.GetVMStaus(ctx, server.ID)
		fmt.Printf("VM %s with ID %s is in %s state\n", server.Name, server.ID, sleepState.Status)
		// TODO: Use this info for cumulative shelve VM status
	}

	// 5. Fetch current quotas post all shelve status cc: @shweta
	// NOTE: Active --> Shelving Image Pending Upload --> Shelving Image Uploading --> Shelved Offloaded

	// 6. Slack the success or failure events of cumulative shelve VM status. cc: @shweta
	// Info of Quotas before Sleep operation
	// Info of Quotas after Sleep operation
	// No of VMs Hibernated Triggered
	// No of VMs failed to hibernate
	// VM Name - User - Tenant - Shelve Status - Current VM Status
}
