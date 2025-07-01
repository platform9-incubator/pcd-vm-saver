package openstack

import (
	"context"
	"os"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/platform9/pcd-vm-saver/pkg/util"
	"go.uber.org/zap"
)

type serverSleepInfo struct {
	Name        string
	ID          string
	SuspendMode bool
	AwakeTime   time.Time
	NewMetadata map[string]string // Metadata to be updated on the server
}

type serverAwakeInfo struct {
	Name        string
	ID          string
	SuspendMode bool
	NewMetadata map[string]string // Metadata to be updated on the server
}

type SleepState struct {
	Name        string
	ID          string
	SleepStatus string
}

type Metrics struct {
	VCPUs    int64
	RAMMB    float64
	DiskGB   float64
	VolumeGB float64
}

func FetchVMsToSleep(ctx context.Context) []serverSleepInfo {

	var sleepVMs []serverSleepInfo

	// TODO: Add this to main init
	// OpenStack authentication credentials
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		// Either of Domain Name or Domain ID is only required not both.
		TenantName: os.Getenv("OS_PROJECT_NAME"),
		TenantID:   os.Getenv("OS_PROJECT_ID"),
	}

	// Authenticate
	provider, err := openstack.AuthenticatedClient(ctx, opts)
	if err != nil {
		zap.S().Errorf("Authentication failed: %v", err)
		return sleepVMs
	}

	// Create compute client
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})

	if err != nil {
		zap.S().Errorf("Failed to create compute client: %v", err)
		return sleepVMs
	}

	// Fetch all servers
	listOpts := servers.ListOpts{
		// TODO: P0 we are only considering active servers
		Status: "ACTIVE", // Only fetch active servers
	}

	allPages, err := servers.List(client, listOpts).AllPages(ctx)
	if err != nil {
		zap.S().Errorf("Failed to list servers: %v", err)
		return sleepVMs
	}

	// Extract server list
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		zap.S().Errorf("Failed to extract servers: %v", err)
		return sleepVMs
	}

	zap.S().Infof("Total servers fetched:", len(serverList))

	// Filter servers by metadata
	for _, server := range serverList {

		// Only check those servers which have metadata
		if len(server.Metadata) > 0 {
			zap.S().Debugf("Checking server:", server.Name, "with ID:", server.ID, "and Metadata:", server.Metadata)

			// Check if OverrideSleepFilter is set to true
			if overrideSleepVal, exists := server.Metadata[util.OverrideSleepFilter]; exists && overrideSleepVal == "true" {
				// If OverrideSleepFilter is set to true, skip this server
				zap.S().Infof("Skipping server %s with ID %s due to OverrideSleepFilter", server.Name, server.ID)
				continue
			}

			// check if metadata contains SleepModeFilter
			var suspendMode bool
			if sleepMode, exists := server.Metadata[util.SleepModeFilter]; exists && sleepMode == "true" {
				// If SleepModeFilter is set to ram_preserve, we need to consider it suspend instead of shelve
				suspendMode = true
			}

			// Check for DefaultSleepFilter or CustomSleepFilter

			// Case 1: Default Sleep Filter i.e Zone based
			if serverVal, exists := server.Metadata[util.DefaultSleepFilter]; exists && (serverVal == util.IndiaSleepVal || serverVal == util.USSleepVal) {

				// Verify the VM needs to seelp now its time i.e 10-10:30 PM
				currentTime := time.Now()

				var sleepTime, awakeTime time.Time

				if serverVal == util.IndiaSleepVal {
					// For India, sleep at 10 PM and awake at 8 AM
					// Change this for testing
					sleepTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 18, 0, 0, 0, currentTime.Location())
					awakeTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day()+1, 8, 0, 0, 0, currentTime.Location())
				}

				if serverVal == util.USSleepVal {
					// For US, sleep at 10 AM and awake at 8 PM
					sleepTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 10, 0, 0, 0, currentTime.Location())
					awakeTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 20, 0, 0, 0, currentTime.Location())
				}

				// Check if current time is between sleep and awake time
				if currentTime.After(sleepTime) && currentTime.Before(awakeTime) {

					// add AwakeTime to existing metadata
					newMetadata := make(map[string]string)
					if server.Metadata != nil {
						newMetadata = server.Metadata
					}

					newMetadata[util.AwakeTimeFilter] = awakeTime.Format(time.RFC3339)

					sleepVMs = append(sleepVMs, serverSleepInfo{
						Name:        server.Name,
						ID:          server.ID,
						SuspendMode: suspendMode,
						AwakeTime:   awakeTime,
						NewMetadata: newMetadata,
					})
				}

				// Case 2: Custom Sleep Filter i.e Zone based
			} else if customSleepVal, exists := server.Metadata[util.CustomSleepFilter]; exists {

				customSleepHours, err := time.ParseDuration(customSleepVal + "h")
				if err != nil {
					zap.S().Errorf("Invalid custom sleep value for server %s with ID %s: %v", server.Name, server.ID, err)
					continue
				}
				currentTime := time.Now()
				creationTime := server.Created

				// Difference
				elapsed := currentTime.Sub(creationTime)

				if elapsed >= customSleepHours && int(elapsed/customSleepHours) >= 0 {

					awakeTime := currentTime.Add(customSleepHours)
					// add AwakeTime to existing metadata
					newMetadata := make(map[string]string)
					if server.Metadata != nil {
						newMetadata = server.Metadata
					}

					newMetadata[util.AwakeTimeFilter] = awakeTime.Format(time.RFC3339)

					// If the elapsed time is a multiple of custom sleep hours, we can consider it for sleep
					zap.S().Infof("Server %s with ID %s is eligible for sleep based on custom sleep filter", server.Name, server.ID)
					sleepVMs = append(sleepVMs, serverSleepInfo{
						Name:        server.Name,
						ID:          server.ID,
						SuspendMode: false,
						AwakeTime:   awakeTime,
						NewMetadata: newMetadata,
					})

				} else {
					// If no default or custom sleep filter, skip this server
					continue
				}
			}
		}
	}
	return sleepVMs
}

func SleepVMs(ctx context.Context, serversInfo []serverSleepInfo) {
	// TODO: Add this to main init
	// OpenStack authentication credentials
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		// Either of Domain Name or Domain ID is only required not both.
		TenantName: os.Getenv("OS_PROJECT_NAME"),
		TenantID:   os.Getenv("OS_PROJECT_ID"),
	}

	// Authenticate
	provider, err := openstack.AuthenticatedClient(ctx, opts)
	if err != nil {
		zap.S().Errorf("Authentication failed: %v", err)
		return
	}

	// Create compute client
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		zap.S().Errorf("Failed to create compute client: %v", err)
		return
	}

	// TODO: Make them parallel
	for _, server := range serversInfo {
		zap.S().Infof("Processing server %s with ID %s for sleep", server.Name, server.ID)
		// NOTE: We need to update the metadata before the VM is suspended or shelved. We can't update it later.

		// Update Server metadata with AwakeTime
		updateOpts := servers.MetadataOpts{}
		for key, value := range server.NewMetadata {
			updateOpts[key] = value
		}

		_, err := servers.UpdateMetadata(ctx, client, server.ID, updateOpts).Extract()
		if err != nil {
			zap.S().Errorf("Failed to update metadata for server %s: %v", server.Name, err)
			//TODO: Add retry logic
			continue
		}

		if server.SuspendMode {
			susRes := servers.Suspend(ctx, client, server.ID)
			if susRes.Err != nil {
				zap.S().Errorf("Failed to suspend server %s: %v", server.Name, susRes.Err)
				continue
			}
		} else {
			shlRes := servers.Shelve(ctx, client, server.ID)
			if shlRes.Err != nil {
				zap.S().Errorf("Failed to shelve server %s: %v", server.Name, shlRes.Err)
				continue
			}
		}

		// So for failed suspend and shelve by metadata is updated, we can handle that case in Awake. Awake if its not Active
		zap.S().Infof("Server %s with ID %s is scheduled to sleep until %s", server.Name, server.ID, server.AwakeTime)
	}
}

func Quotas(ctx context.Context) {
	// Get current openstack vCPU, RAM, Volume Storage
	//TODO Add this logic
}

func GetVMStaus(ctx context.Context, serverId string) *servers.Server {
	// TODO: Add this to main init
	// OpenStack authentication credentials
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		// Either of Domain Name or Domain ID is only required not both.
		TenantName: os.Getenv("OS_PROJECT_NAME"),
		TenantID:   os.Getenv("OS_PROJECT_ID"),
	}

	// Authenticate
	provider, err := openstack.AuthenticatedClient(ctx, opts)
	if err != nil {
		zap.S().Errorf("Authentication failed: %v", err)
	}

	// Create compute client
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		zap.S().Errorf("Failed to create compute client: %v", err)
	}

	// GET server list
	server, err := servers.Get(ctx, client, serverId).Extract()
	if err != nil {
		zap.S().Errorf("Failed to get server %s: %v", serverId, err)
		return nil
	}

	return server
}

func GetVMsToAwake(ctx context.Context) []serverAwakeInfo {

	var awakeVMs []serverAwakeInfo

	// TODO: Add this to main init
	// OpenStack authentication credentials
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		// Either of Domain Name or Domain ID is only required not both.
		TenantName: os.Getenv("OS_PROJECT_NAME"),
		TenantID:   os.Getenv("OS_PROJECT_ID"),
	}

	// Authenticate
	provider, err := openstack.AuthenticatedClient(ctx, opts)
	if err != nil {
		zap.S().Errorf("Authentication failed: %v", err)
		return awakeVMs
	}

	// Create compute client
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})

	if err != nil {
		zap.S().Errorf("Failed to create compute client: %v", err)
		return awakeVMs
	}

	// Fetch all servers
	listOpts := servers.ListOpts{}

	allPages, err := servers.List(client, listOpts).AllPages(ctx)
	if err != nil {
		zap.S().Errorf("Failed to list servers: %v", err)
		return awakeVMs
	}

	// Extract server list
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		zap.S().Errorf("Failed to extract servers: %v", err)
		return awakeVMs
	}

	for _, server := range serverList {

		// Only check those servers which have metadata
		if len(server.Metadata) > 0 {
			// stale awake timestamp
			if server.Status == "ACTIVE" {
				// If the server is already active, we don't need to awake it
				zap.S().Infof("Server %s with ID %s is already active, skipping awake", server.Name, server.ID)
				continue
			}
			zap.S().Debugf("Checking server:", server.Name, "with ID:", server.ID, "and Metadata:", server.Metadata)

			var suspendMode bool
			if sleepMode, exists := server.Metadata[util.SleepModeFilter]; exists && sleepMode == "true" {
				// If SleepModeFilter is set to ram_preserve, we need to consider it suspend instead of shelve
				suspendMode = true
			}

			// Check for AwakeTimeFilter
			if awakeTimeStr, exists := server.Metadata[util.AwakeTimeFilter]; exists {
				awakeTime, err := time.Parse(time.RFC3339, awakeTimeStr)
				if err != nil {
					zap.S().Errorf("Invalid AwakeTime for server %s with ID %s: %v", server.Name, server.ID, err)
					continue
				}

				// remove AwakeTimeFilter from metadata
				metadata := server.Metadata
				delete(metadata, util.AwakeTimeFilter)

				currentTime := time.Now()
				if currentTime.After(awakeTime) {
					// If current time is after AwakeTime, we can consider it for awake
					awakeVMs = append(awakeVMs, serverAwakeInfo{
						Name:        server.Name,
						ID:          server.ID,
						SuspendMode: suspendMode,
						NewMetadata: metadata, // remove AwakeTimeFilter from metadata
					})
				} else {
					zap.S().Infof("Server %s with ID %s is not yet ready to awake, current time: %s, awake time: %s", server.Name, server.ID, currentTime.Format(time.RFC3339), awakeTime.Format(time.RFC3339))
				}
			}
		}
	}
	return awakeVMs
}

func AwakeVMs(ctx context.Context, awakeVMsInfo []serverAwakeInfo) {
	// TODO: Add this to main init
	// OpenStack authentication credentials
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		// Either of Domain Name or Domain ID is only required not both.
		TenantName: os.Getenv("OS_PROJECT_NAME"),
		TenantID:   os.Getenv("OS_PROJECT_ID"),
	}

	// Authenticate
	provider, err := openstack.AuthenticatedClient(ctx, opts)
	if err != nil {
		zap.S().Errorf("Authentication failed: %v", err)
		return
	}

	// Create compute client
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		zap.S().Errorf("Failed to create compute client: %v", err)
		return
	}

	for _, server := range awakeVMsInfo {
		zap.S().Infof("Processing server %s with ID %s to awake", server.Name, server.ID)

		if server.SuspendMode {
			// Resume the server if it was suspended
			resumeRes := servers.Resume(ctx, client, server.ID)
			if resumeRes.Err != nil {
				zap.S().Errorf("Failed to resume server %s: %v", server.Name, resumeRes.Err)
				continue
			}
		} else {
			// Unshelve the server if it was shelved
			unshelveRes := servers.Unshelve(ctx, client, server.ID, servers.UnshelveOpts{})
			if unshelveRes.Err != nil {
				zap.S().Errorf("Failed to unshelve server %s: %v", server.Name, unshelveRes.Err)
				continue
			}
		}

		// TODO: Can't update metadata immediately after the server is resumed/unshelve
		// // Update metadata to remove AwakeTimeFilter
		// updateOpts := servers.MetadataOpts{}
		// for key, value := range server.NewMetadata {
		// 	updateOpts[key] = value
		// }
		// _, err := servers.UpdateMetadata(ctx, client, server.ID, updateOpts).Extract()
		// if err != nil {
		// 	zap.S().Errorf("Failed to update metadata for server %s: %v", server.Name, err)
		// 	//TODO: Add retry logic
		// 	continue
		// }
		zap.S().Infof("Server %s with ID %s is scheduled to awake", server.Name, server.ID)
	}
}
