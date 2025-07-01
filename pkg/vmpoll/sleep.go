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

	if len(serversInfo) == 0 {
		zap.S().Info("No VMs found to sleep")
		return "No VMs found to sleep", nil
	}

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
	time.Sleep(25 * time.Second)

	// 4. Fetch the status and generate the cumulative shelve VM status
	for _, server := range serversInfo {
		sleepState := openstack.GetVMStatus(ctx, server.ID)

		if sleepState.Status != "SHELVED_OFFLOADED" && sleepState.Status != "SUSPENDED" {
			for {
				zap.S().Warnf("VM %s (ID: %s) is not in SHELVED_OFFLOADED/SUSPENDED state, current state: %s", server.Name, server.ID, sleepState.Status)
				time.Sleep(15 * time.Second)                       // Wait before retrying
				sleepState = openstack.GetVMStatus(ctx, server.ID) // Re-fetch the status
				if sleepState.Status == "SHELVED_OFFLOADED" || sleepState.Status == "SUSPENDED" {
					successMsg += fmt.Sprintf("VM %s (ID: %s) - Current state: %s\n", server.Name, server.ID, sleepState.Status)
					break // Exit loop if the VM is shelved/suspended
				}
				zap.S().Infof("Retrying to check VM %s (ID: %s) status", server.Name, server.ID)
			}
		}
	}

	// Adding a minimum time wait
	time.Sleep(25 * time.Second)

	newQuotas := openstack.Quotas(ctx)
	successMsg += fmt.Sprintf("Quota after sleep operations:\n")
	successMsg += fmt.Sprintf("Cores: %d / %d\n", newQuotas.VCPUsInUse, newQuotas.VCPUsLimit)
	successMsg += fmt.Sprintf("RAM: %d / %d\n\n", newQuotas.RAMInUse, newQuotas.RAMLimit)

	return successMsg, nil
}
