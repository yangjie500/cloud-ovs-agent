package ovs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
	"github.com/yangjie500/cloud-ovs-agent/pkg/logger"
)

func EnsureInterfaceOnBridge(ctx context.Context, client client.Client, bridgeName, ifName, logicalPort string) error {
	start := time.Now()
	logger.Infof("[ovs] ensure interface on bridge: br=%s if=%s lp=%s", bridgeName, ifName, logicalPort)

	br, err := findBridgeByName(ctx, client, bridgeName)
	if err != nil {
		logger.Errorf("[ovs] get bridge by name failed: %v", err)
		return err
	}
	logger.Debugf("[ovs] target bridge: name=%s uuid=%s", br.Name, br.UUID)

	iface, _ := findInterfaceByName(ctx, client, ifName)
	if iface != nil {
		logger.Debugf("[ovs] interface exists: %s (uuid=%s)", iface.Name, iface.UUID)
	} else {
		logger.Debugf("[ovs] interface %s missing; will create", ifName)
	}

	port, _ := findPortByName(ctx, client, ifName)
	if port != nil {
		logger.Debugf("[ovs] port exists: %s (uuid=%s)", port.Name, port.UUID)
	} else {
		logger.Debugf("[ovs] interface %s missing; will create", logicalPort)
	}

	ops := make([]ovsdb.Operation, 0, 8)

	var newPortID string
	// Both interface and port does not exists
	if iface == nil && port == nil {
		newIfaceID := uuid.New().String()
		createIfOps, err := buildCreateInterfaceOps(client, newIfaceID, ifName, logicalPort)
		if err != nil {
			return err
		}
		ops = append(ops, createIfOps...)

		newPortID = uuid.New().String()
		portOps, err := buildCreatePortOps(client, newPortID, logicalPort, newIfaceID)
		if err != nil {
			return err
		}
		ops = append(ops, portOps...)

	} else {
		logger.Errorf("Interface %s and Port %s already existed already. Consider deleting it", ifName, logicalPort)
		return fmt.Errorf("Interface %s and Port %s already existed", ifName, logicalPort)
	}

	bridgeOps, err := buildAttachPortToBridgeOps(client, br.UUID, newPortID)
	if err != nil {
		return err
	}
	ops = append(ops, bridgeOps...)

	if len(ops) == 0 {
		logger.Infof("[ovs] no changes needed for if=%s on bridge=%s", ifName, bridgeName)
		return nil
	}
	logger.Debugf("[ovs] transact ops count=%d", len(ops))
	result, err := client.Transact(ctx, ops...)
	if err != nil {
		logger.Errorf("[ovs] transact failed: %v", err)
		return err
	}

	for _, r := range result {
		if r.Error != "" {
			err := fmt.Errorf("ovs error: %s (details: %s)", r.Error, r.Details)
			logger.Errorf("[ovs] ensure interface on bridge error: %v", err)
			return err
		}
	}

	logger.Infof("[ovs] ensured if=%s on bridge=%s in %s", ifName, bridgeName, time.Since(start).Truncate(time.Millisecond))
	return nil
}

func RemoveInterfaceFromBridge(ctx context.Context, client client.Client, bridgeName, ifName, logicalPort string) error {
	start := time.Now()

	br, err := findBridgeByName(ctx, client, bridgeName)
	if err != nil {
		logger.Errorf("[ovs] get bridge by name failed: %v", err)
		return err
	}
	logger.Debugf("[ovs] target bridge: name=%s uuid=%s", br.Name, br.UUID)

	// iface, _ := findInterfaceByName(ctx, client, ifName)
	port, _ := findPortByName(ctx, client, logicalPort)

	ops := make([]ovsdb.Operation, 0, 6)

	if br != nil && port != nil && bridgeHasPort(br, port.UUID) {
		logger.Debugf("[ovs] detaching port %s from bridge %s", logicalPort, bridgeName)
		detachOps, err := buildDetachPortFromBridgeOps(client, br.UUID, port.UUID)
		if err != nil {
			return err
		}
		ops = append(ops, detachOps...)
	}

	if len(ops) == 0 {
		logger.Infof("[ovs] no changes need for if=%s on bridge=%s", ifName, bridgeName)
		return nil
	}
	logger.Debugf("[ovs] transact ops count=%d", len(ops))
	result, err := client.Transact(ctx, ops...)
	if err != nil {
		logger.Errorf("[ovs] transact failed: %v", err)
		return err
	}
	logger.Debugf("Result %+v", result)

	logger.Infof("[ovs] cleanup done for if=%s in %s", ifName, time.Since(start).Truncate(time.Millisecond))
	return nil
}
