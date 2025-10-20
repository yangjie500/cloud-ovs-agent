package sb

import (
	"context"

	"github.com/ovn-kubernetes/libovsdb/cache"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/yangjie500/cloud-ovs-agent/internal/netdev"
	"github.com/yangjie500/cloud-ovs-agent/internal/ovs"
	"github.com/yangjie500/cloud-ovs-agent/pkg/logger"
)

type PBWatcher struct {
	Ctx     context.Context
	SbCli   client.Client
	OvsCli  client.Client
	Chassis string // host's chassis/system-id
	Bridge  string // usually "br-int"
}

func (w *PBWatcher) isPB(m model.Model) (*PortBinding, bool) {
	pb, ok := m.(*PortBinding)
	if !ok || pb == nil || pb.LogicalPort == "" {
		return nil, false
	}
	return pb, true
}

func (w *PBWatcher) requestedForThisChassis(pb *PortBinding) bool {
	if pb.Options == nil {
		// no preference → up to your policy; we choose to allow only explicit matches
		return false
	}
	if rc, ok := pb.Options["requested-chassis"]; ok && rc != "" && rc == w.Chassis {
		return true
	}
	return false
}

func RegisterPBHandler(ctx context.Context, sbCli client.Client, ovsCli client.Client, chassis, bridge string) {
	w := &PBWatcher{Ctx: ctx, SbCli: sbCli, OvsCli: ovsCli, Chassis: chassis, Bridge: bridge}
	//DNU
	sbCli.Cache().AddEventHandler(&cache.EventHandlerFuncs{
		AddFunc:    w.onAdd,
		DeleteFunc: w.onDelete,
	})
}

func (w *PBWatcher) onAdd(table string, m model.Model) {
	if table != "Port_Binding" {
		return
	}
	pb, ok := m.(*PortBinding)
	if !ok || pb == nil || pb.LogicalPort == "" {
		return
	}

	// Only realize ports meant for this chassis
	if pb.Options != nil {
		if rc, ok := pb.Options["requested-chassis"]; ok && rc != "" && rc != w.Chassis {
			logger.Debugf("[sb] skip %s: requested-chassis=%s (this=%s)", pb.LogicalPort, rc, w.Chassis)
			return
		}
	}

	// If already bound to another chassis, skip
	if pb.Chassis != nil && *pb.Chassis != "" && *pb.Chassis != w.Chassis {
		logger.Debugf("[sb] skip %s: bound to %s", pb.LogicalPort, *pb.Chassis)
	}

	ifName := "tap-" + pb.LogicalPort
	if _, err := netdev.CreateTap(ifName, 1500, true); err != nil {
		logger.Errorf("[agent] create tap %s failed: %v", ifName, err)
		return
	}

	if err := ovs.EnsureInterfaceOnBridge(w.Ctx, w.OvsCli, w.Bridge, ifName, pb.LogicalPort); err != nil {
		logger.Errorf("[agent] ensure OVS for %s failed: %v", pb.LogicalPort, err)
		return
	}

	if _, err := netdev.SetLinkUp(ifName); err != nil {
		logger.Errorf("[agent] unable to set link %s up: %v", ifName, err)
		return
	}
}

func (w *PBWatcher) onDelete(table string, m model.Model) {
	if table != "Port_Binding" {
		return
	}
	pb, ok := m.(*PortBinding)
	if !ok || pb == nil || pb.LogicalPort == "" {
		return
	}

	ifName := "tap-" + pb.LogicalPort

	if err := ovs.RemoveInterfaceFromBridge(w.Ctx, w.OvsCli, w.Bridge, ifName, pb.LogicalPort); err != nil {
		logger.Errorf("[agent] cleanup %s failed: %v", ifName, err)
	}

	logger.Infof("[agent] cleaned up logical_port=%s if=%s", pb.LogicalPort, ifName)
}
