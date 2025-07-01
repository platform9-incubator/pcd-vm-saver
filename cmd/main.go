package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/platform9/pcd-vm-saver/pkg/log"
	"github.com/platform9/pcd-vm-saver/pkg/util"
	"github.com/platform9/pcd-vm-saver/pkg/vmpoll"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type CronSkipperLogger struct{}

func (c *CronSkipperLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	zap.S().Error(err)
	zap.S().Errorw(msg, keysAndValues...)
}

func (c *CronSkipperLogger) Info(msg string, keysAndValues ...interface{}) {
	zap.S().Infow(msg, keysAndValues...)
}

func run(*cobra.Command, []string) {

	zap.S().Info("Starting pcd-vm-saver...")
	zap.S().Infof("Version of pcd-vm-saver being used is: %s", util.Version)
	zap.S().Info("starting scheduled tasks")
	vmpoll.AutoSleepVM()
	vmpoll.AutoAwakeVM()

	schedule := cron.New(cron.WithChain(cron.SkipIfStillRunning(&CronSkipperLogger{})))

	schedule.AddFunc("@every 15m", vmpoll.AutoSleepVM) // will hibernate/sleep clusters if there are adequate ready clusters in pool
	schedule.AddFunc("@every 15m", vmpoll.AutoAwakeVM)
	schedule.Start()
	zap.S().Info("cron jobs scheduled")

	zap.S().Info("pcd-vm-saver is running")
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		zap.S().Info("server stopping...")
		schedule.Stop()
	}
}

func main() {
	cmd := buildCmds()
	cmd.Execute()
}

func buildCmds() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pcd-vm-saver",
		Short: "pcd-vm-saver helps handling VMs efficiently by hibernating and awaking them",
		Long:  "pcd-vm-saver helps handling VMs efficiently by hibernating and awaking them",
		Run:   run,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Current version of pcd-vm-saver being used",
		Long:  "Current version of pcd-vm-saver being used",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(util.Version)
		},
	}

	rootCmd.AddCommand(versionCmd)
	return rootCmd
}

func init() {
	err := log.Logger()
	if err != nil {
		fmt.Printf("Failed to initiate logger, Error is: %s", err)
	}
}
