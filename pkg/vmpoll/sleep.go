package vmpoll

import (
	"context"
	"fmt"
	"time"

	"github.com/platform9/pcd-vm-saver/pkg/openstack"
	"go.uber.org/zap"
)

func AutoSleepVM() (string, error) {
	ctx := context.TODO()
	zap.S().Infof("Triggering auto sleep VMs")

	// 1. Fetch available list of VMs with Default Sleep Filter
	serversInfo := openstack.FetchVMsToSleep(ctx)

	// 2. Fetch current quotas
	currentQuotas := openstack.Quotas(ctx)

	// Build success message
	successMsg := fmt.Sprintf("Quota before sleep operations:\n")
	successMsg += fmt.Sprintf("Cores: %d / %d\n", currentQuotas.VCPUsInUse, currentQuotas.VCPUsLimit)
	successMsg += fmt.Sprintf("RAM: %d / %d\n\n", currentQuotas.RAMInUse, currentQuotas.RAMLimit)
	successMsg += "Servers being shelved/suspended:\n"

	// 3. Parallely Shelve/Suspend all the VMs
	openstack.SleepVMs(ctx, serversInfo)

	// Adding a minimum time wait
	time.Sleep(15 * time.Second)

	// 4. Fetch the status and generate the cumulative shelve VM status
	for _, server := range serversInfo {
		sleepState := openstack.GetVMStaus(ctx, server.ID)
		successMsg += fmt.Sprintf("VM %s (ID: %s) - Current state: %s\n", server.Name, server.ID, sleepState.Status)
	}

	newQuotas := openstack.Quotas(ctx)
	successMsg += fmt.Sprintf("Quota after sleep operations:\n")
	successMsg += fmt.Sprintf("Cores: %d / %d\n", newQuotas.VCPUsInUse, newQuotas.VCPUsLimit)
	successMsg += fmt.Sprintf("RAM: %d / %d\n\n", newQuotas.RAMInUse, newQuotas.RAMLimit)

	return successMsg, nil
}
