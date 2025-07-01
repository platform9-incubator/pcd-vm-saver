package openstack

import (
	"context"
	"fmt"
	"os"
	"sync"
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

func FetchVMsToSleep(ctx context.Context) ([]serverSleepInfo, error) {
	var sleepVMs []serverSleepInfo

	// OpenStack authentication credentials
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		TenantName:       os.Getenv("OS_PROJECT_NAME"),
		TenantID:         os.Getenv("OS_PROJECT_ID"),
	}

	// Validate required environment variables
	if opts.IdentityEndpoint == "" || opts.Username == "" || opts.Password == "" {
		return nil, fmt.Errorf("missing required OpenStack credentials")
	}

	// Authenticate
	provider, err := openstack.AuthenticatedClient(ctx, opts)
	if err != nil {
		zap.S().Errorf("Authentication failed: %v", err)
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	// Create compute client
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		zap.S().Errorf("Failed to create compute client: %v", err)
		return nil, fmt.Errorf("failed to create compute client: %v", err)
	}

	// Fetch all servers
	listOpts := servers.ListOpts{
		Status: "ACTIVE", // Only fetch active servers
	}

	allPages, err := servers.List(client, listOpts).AllPages(ctx)
	if err != nil {
		zap.S().Errorf("Failed to list servers: %v", err)
		return nil, fmt.Errorf("failed to list servers: %v", err)
	}

	// Extract server list
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		zap.S().Errorf("Failed to extract servers: %v", err)
		return nil, fmt.Errorf("failed to extract servers: %v", err)
	}

	zap.S().Infof("Total servers fetched:", len(serverList))

	// Filter servers by metadata
	for _, server := range serverList {
		if len(server.Metadata) > 0 {
			zap.S().Debugf("Checking server:", server.Name, "with ID:", server.ID, "and Metadata:", server.Metadata)

			// Check if OverrideSleepFilter is set to true
			if overrideSleepVal, exists := server.Metadata[util.OverrideSleepFilter]; exists && overrideSleepVal == "true" {
				zap.S().Infof("Skipping server %s with ID %s due to OverrideSleepFilter", server.Name, server.ID)
				continue
			}

			// Check SleepModeFilter
			var suspendMode bool
			if sleepMode, exists := server.Metadata[util.SleepModeFilter]; exists && sleepMode == "true" {
				suspendMode = true
			}

			// Check sleep filters
			if serverVal, exists := server.Metadata[util.DefaultSleepFilter]; exists && (serverVal == util.IndiaSleepVal || serverVal == util.USSleepVal) {
				currentTime := time.Now()
				var sleepTime, awakeTime time.Time

				switch serverVal {
				case util.IndiaSleepVal:
					sleepTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 18, 0, 0, 0, currentTime.Location())
					awakeTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day()+1, 8, 0, 0, 0, currentTime.Location())
				case util.USSleepVal:
					sleepTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 10, 0, 0, 0, currentTime.Location())
					awakeTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 20, 0, 0, 0, currentTime.Location())
				}

				if currentTime.After(sleepTime) && currentTime.Before(awakeTime) {
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
			} else if customSleepVal, exists := server.Metadata[util.CustomSleepFilter]; exists {
				customSleepHours, err := time.ParseDuration(customSleepVal + "h")
				if err != nil {
					zap.S().Errorf("Invalid custom sleep value for server %s with ID %s: %v", server.Name, server.ID, err)
					continue
				}

				currentTime := time.Now()
				creationTime := server.Created
				elapsed := currentTime.Sub(creationTime)

				if elapsed >= customSleepHours {
					awakeTime := currentTime.Add(customSleepHours)
					newMetadata := make(map[string]string)
					if server.Metadata != nil {
						newMetadata = server.Metadata
					}

					newMetadata[util.AwakeTimeFilter] = awakeTime.Format(time.RFC3339)
					sleepVMs = append(sleepVMs, serverSleepInfo{
						Name:        server.Name,
						ID:          server.ID,
						SuspendMode: false,
						AwakeTime:   awakeTime,
						NewMetadata: newMetadata,
					})
				}
			}
		}
	}

	return sleepVMs, nil
}

func SleepVMs(ctx context.Context, serversInfo []serverSleepInfo) error {
	// OpenStack authentication credentials
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		TenantName:       os.Getenv("OS_PROJECT_NAME"),
		TenantID:         os.Getenv("OS_PROJECT_ID"),
	}

	// Validate required environment variables
	if opts.IdentityEndpoint == "" || opts.Username == "" || opts.Password == "" {
		return fmt.Errorf("missing required OpenStack credentials")
	}

	// Authenticate
	provider, err := openstack.AuthenticatedClient(ctx, opts)
	if err != nil {
		zap.S().Errorf("Authentication failed: %v", err)
		return fmt.Errorf("authentication failed: %v", err)
	}

	// Create compute client
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		zap.S().Errorf("Failed to create compute client: %v", err)
		return fmt.Errorf("failed to create compute client: %v", err)
	}

	// Process servers in parallel
	var wg sync.WaitGroup
	for _, server := range serversInfo {
		wg.Add(1)
		go func(server serverSleepInfo) {
			defer wg.Done()

			zap.S().Infof("Processing server %s with ID %s for sleep", server.Name, server.ID)

			// Update Server metadata with AwakeTime
			updateOpts := servers.MetadataOpts{}
			for key, value := range server.NewMetadata {
				updateOpts[key] = value
			}

			_, err := servers.UpdateMetadata(ctx, client, server.ID, updateOpts).Extract()
			if err != nil {
				zap.S().Errorf("Failed to update metadata for server %s: %v", server.Name, err)
				// TODO: Add retry logic
				return
			}

			// Suspend or Shelve based on SuspendMode
			if server.SuspendMode {
				susRes := servers.Suspend(ctx, client, server.ID)
				if susRes.Err != nil {
					zap.S().Errorf("Failed to suspend server %s: %v", server.Name, susRes.Err)
					return
				}
			} else {
				shlRes := servers.Shelve(ctx, client, server.ID)
				if shlRes.Err != nil {
					zap.S().Errorf("Failed to shelve server %s: %v", server.Name, shlRes.Err)
					return
				}
			}
		}(server)
	}

	wg.Wait()
	return nil
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
