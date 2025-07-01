# pcd-vm-saver

`pcd-vm-saver` is a tool designed to efficiently manage virtual machines (VMs) by automating their hibernation and awakening processes. It helps optimize resource usage by suspending or shelving VMs during idle periods and waking them up when needed.

## 📊 Features
- 🚀  **Automatic VM Sleep aka Hibernate**: Automatically hibernates VMs based on predefined filters such as time zones or custom sleep durations (metadata).

- 🚀  **Automatic VM Awake**: Wakes up VMs based on scheduled awake time in metadata configurations.

- **Customizable Filters**: 
    * Supports default sleep filters (e.g. time zone-based, `sleep_zone=ist`) 
    * Custom sleep filters (e.g. sleep duration in hours `sleep_time=8`) 
    * Sleep Mode filter (optional) (e.g. `ram_preserve=true`)

- 📣  **Slack Notifications** for VM sleep, awake actions and quota metrics.

- 📂 **Logging**: Provides detailed logs for debugging and monitoring VM operations.


## Pre-requisites
To start the pcd-vm-saver service, pre-requisites are:

1. Linux/Mac (Preferred)
2. 🔐 Openstack credentials 

Set the required OpenStack environment variables in the:

* OS_AUTH_URL
* OS_USERNAME
* OS_PASSWORD
* OS_USER_DOMAIN_NAME
* OS_PROJECT_NAME
* OS_PROJECT_ID
* OS_REGION_NAME
* SLACK_CHANNEL_ID
* SLACK_APP_TOKEN
* SLACK_BOT_TOKEN

## 🛠 Build pcd-vm-saver 

Clone the repository, navigate to the cloned repository and download the dependencies using `go mod download`. Before building, ensure the required pre-requisites are met.

To build the pcd-vm-saver binary, use the below command, pcd-vm-saver binary built using make is placed in `bin` directory.

```sh
# Using make, prefered for linux OS.
make build-linux

# Using make, for mac OS
make build-mac

# Using go build and run.
sudo go run cmd/main.go
```

## Run pcd-vm-saver

`pcd-vm-saver` can be run using binary and as a system service.

### Using binary
To run pcd-vm-saver through binary, follow the below command:
```sh
# Start the pcd-vm-saver.
./bin/pcd-vm-saver
```

### Using system service file
To run pcd-vm-saver as a system service, service file [pcd-vm-saver.service](pcd-vm-saver.service) should be placed at `/etc/systemd/system/` directory and pcd-vm-saver binary at `/usr/bin/pcd-vm-saver/` directory. To start the service follow the below commands:

```sh
# Start the pcd-vm-saver service.
sudo systemctl start pcd-vm-saver.service
```

`pcd-vm-saver service` will be now up and running, to check the latest status of service:

```sh
# Check the status of pcd-vm-saver
sudo systemctl status pcd-vm-saver.service

* Logs for pcd-vm-saver can be found at `$HOME/pcd-vm-saver-logs/vm-saver.log`
```

### Using Teamcity Cron Job
We can host and integrate this repo with teamcity cron job similar to our existing resource-cleanup task. Thus allowing us to host it at central location and periodically monitor VM resources `hibernate, awake`.