package util

const (
	Version = "pcd-vm-saver version: v1.0"

	DefaultSleepFilter = "sleep_zone"
	IndiaSleepVal      = "ist"
	USSleepVal         = "us"

	// custom sleep filter needs to have any interger value it will be considered as hours
	CustomSleepFilter = "sleep_time"

	OverrideSleepFilter = "save_sleep"

	SleepModeFilter = "ram_preserve" // Consider Suspend instead of Shelve VM
	AwakeTimeFilter = "awake_time"   // Metadata key to store awake time for the VM
)

/*
1. Create a VM
2. Add a defaultSleepFilter
3. Check current Time IST and if exceeding the DefaultSleepIndiaVMs/DefaultSleepUSVms sleep them.
4. Add timespamp metadata to awake it at the next day.
--
2. Add a customSleepFilter to the VM
3. Check if the creation time stamp is multiples of custom sleep filter time then sleep it.
4. Add timespamp metadata to awake it accordingly.
*/
