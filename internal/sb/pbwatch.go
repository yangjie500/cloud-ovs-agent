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

func (w *PBWatcher) checkIsPB(m model.Model) (*PortBinding, bool) {
	pb, ok := m.(*PortBinding)
	if !ok || pb == nil || pb.LogicalPort == "" {
		logger.Infof("[sb] Port Binding does not conform to Port Binding Format")
		return nil, false
	}
	return pb, true
}

func (w *PBWatcher) requestedForThisChassis(pb *PortBinding) bool {
	if pb.Options == nil {
		// no preference → up to your policy; we choose to allow only explicit matches
		logger.Debugf("[sb] Port Binding has no requested chassis")
		return false
	}
	if rc, ok := pb.Options["requested-chassis"]; ok && rc != "" && rc == w.Chassis {
		return true
	}
	logger.Debugf("[sb] Port Binding is not for this requested chassis, Chassis: %s; PortBinding Chassis: %s", w.Chassis, pb.Options["requested-chassis"])
	return false
}

func RegisterPBHandler(ctx context.Context, sbCli client.Client, ovsCli client.Client, chassis, bridge string) {
	w := &PBWatcher{Ctx: ctx, SbCli: sbCli, OvsCli: ovsCli, Chassis: chassis, Bridge: bridge}
	sbCli.Cache().AddEventHandler(&cache.EventHandlerFuncs{
		AddFunc:    w.onAdd,
		DeleteFunc: w.onDelete,
	})
}

func (w *PBWatcher) onAdd(table string, m model.Model) {
	if table != "Port_Binding" {
		logger.Infof("Table is not Port_Binding")
		return
	}
	pb, isPb := w.checkIsPB(m)
	if !isPb {
		return
	}

	logger.Infof("UUID: %s; logicalPort: %s; type: %s; datapath: %s, tunnelKey: %d, chassis: %s, up: %t; options: %+v",
		pb.UUID,
		pb.LogicalPort,
		pb.Type,
		pb.Datapath,
		pb.TunnelKey,
		valOrNil(pb.Chassis),
		valOrNil(pb.Up),
		pb.Options)

	if pb.Type == "patch" {
		logger.Debugf("[agent] Ignoring router portl; router: %s", pb.LogicalPort)
		return
	}

	isForThisChassis := w.requestedForThisChassis(pb)
	if !isForThisChassis {
		return
	}

	ifName := pb.LogicalPort
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

	logger.Infof("[agent] created and link up logical_port=%s if=%s", pb.LogicalPort, ifName)
}

func (w *PBWatcher) onDelete(table string, m model.Model) {
	if table != "Port_Binding" {
		logger.Infof("Table is not Port_Binding")
		return
	}

	pb, isPb := w.checkIsPB(m)
	if !isPb {
		return
	}

	logger.Infof("UUID: %s; logicalPort: %s; type: %s; datapath: %s, tunnelKey: %d, chassis: %s, up: %t; options: %+v",
		pb.UUID,
		pb.LogicalPort,
		pb.Type,
		pb.Datapath,
		pb.TunnelKey,
		valOrNil(pb.Chassis),
		valOrNil(pb.Up),
		pb.Options)

	if pb.Type == "patch" {
		logger.Debugf("[agent] Ignoring router portl; router: %s", pb.LogicalPort)
		return
	}

	ifName := pb.LogicalPort

	if err := ovs.RemoveInterfaceFromBridge(w.Ctx, w.OvsCli, w.Bridge, ifName, pb.LogicalPort); err != nil {
		logger.Errorf("[agent] cleanup %s failed: %v", ifName, err)
	}

	if err := netdev.SetLinkDown(ifName); err != nil {
		logger.Errorf("[agent] unable to set link %s up: %v", ifName, err)
		return
	}
	if err := netdev.DeleteLink(ifName); err != nil {
		logger.Warnf("[agent] delete link %s: %v", ifName, err)
	}

	logger.Infof("[agent] cleaned up logical_port=%s if=%s", pb.LogicalPort, ifName)
}

func valOrNil[T any](p *T) any {
	if p == nil {
		return "<nil>"
	}
	return *p
}
