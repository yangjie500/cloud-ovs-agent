package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/yangjie500/cloud-ovs-agent/internal/ovs"
	"github.com/yangjie500/cloud-ovs-agent/internal/sb"
	"github.com/yangjie500/cloud-ovs-agent/pkg/config"
	"github.com/yangjie500/cloud-ovs-agent/pkg/logger"
)

func main() {
	cfg, err := config.LoadAll()
	if err != nil {
		logger.Errorf("Something wrong loading env var: %v", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Connect to local OVSDB (Open_vSwitch)
	ovsCli, err := ovs.ConnectOVS(ctx, "unix:/usr/local/var/run/openvswitch/db.sock")
	if err != nil {
		logger.Errorf("OVS connect failed: %v", err)
		return
	}
	defer ovsCli.Close()

	// Connect to OVN Southbound (central)
	sbCli, err := sb.ConnectSouthBound(ctx, "tcp:"+cfg.SouthboundIp+":"+cfg.SouthboundPort)
	if err != nil {
		logger.Errorf("SB connect failed: %v", err)
	}
	defer sbCli.Close()

	// Tiny cache warm-up so EnsureInterfaceOnBridge's List/Get uses a populated cache
	time.Sleep(200 * time.Millisecond)
	sb.RegisterPBHandler(ctx, sbCli, ovsCli, cfg.HypervisorName, "br-int")

	<-ctx.Done()
	time.Sleep(150 * time.Millisecond)
	logger.Infof("Exiting...")
}
