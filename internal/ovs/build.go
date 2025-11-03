package ovs

import (
	"fmt"

	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
	"github.com/yangjie500/cloud-ovs-agent/pkg/logger"
)

func buildCreateInterfaceOps(client client.Client, ifaceUUID, ifName, logicalPort string) ([]ovsdb.Operation, error) {
	ifRow := &Interface{
		UUID: ifaceUUID,
		Name: ifName,
		ExternalIDs: map[string]string{
			"iface-id": logicalPort,
		},
		Type: "system",
	}

	ops, err := client.Create(ifRow)
	if err != nil {
		logger.Errorf("[ovs] build create-if op failed: %v", err)
		return nil, err
	}
	return ops, nil
}

func buildCreatePortOps(client client.Client, portUUID, portName, ifaceRef string) ([]ovsdb.Operation, error) {
	portRow := &Port{
		UUID:       portUUID,
		Name:       portName,
		Interfaces: []string{ifaceRef},
	}
	ops, err := client.Create(portRow)
	if err != nil {
		logger.Errorf("[ovs] build create-port op failed %v", err)
		return nil, err
	}
	return ops, nil
}

func buildAttachPortToBridgeOps(client client.Client, bridgeUUID, portRef string) ([]ovsdb.Operation, error) {
	m := &Bridge{UUID: bridgeUUID}

	ops, err := client.Where(m).Mutate(m, model.Mutation{
		Field:   &m.Ports,
		Mutator: ovsdb.MutateOperationInsert,
		Value:   []string{portRef},
	})
	if err != nil {
		return nil, fmt.Errorf("build insert bridge mutate (attach port) failed: %w", err)
	}
	return ops, nil
}

func buildDetachPortFromBridgeOps(client client.Client, bridgeUUID, portRef string) ([]ovsdb.Operation, error) {
	if bridgeUUID == "" || portRef == "" {
		logger.Errorf("detach: empty bridge or port UUID; bridge=%s port=%s", bridgeUUID, portRef)
		return nil, fmt.Errorf("detach: empty bridge or port UUID")
	}
	m := &Bridge{UUID: bridgeUUID}
	ops, err := client.Where(m).Mutate(m, model.Mutation{
		Field:   &m.Ports,
		Mutator: ovsdb.MutateOperationDelete,
		Value:   []string{portRef},
	})

	if err != nil {
		logger.Errorf("build delete bridge mutate (detach port) failed: %v", err)
		return nil, fmt.Errorf("build delete bridge mutate (detach port) failed: %w", err)
	}

	return ops, nil
}

func buildDeletePortOps(client client.Client, portUUID string) ([]ovsdb.Operation, error) {
	if portUUID == "" {
		return nil, fmt.Errorf("delete port: empty UUID")
	}
	m := &Port{UUID: portUUID}
	ops, err := client.Where(m).Delete()
	if err != nil {
		logger.Errorf("[ovs] build delete port op failed: %v", err)
	}
	return ops, nil
}

// Build ops to ensure external_ids:iface-id equals logicalPort (without overwriting other keys).
func buildEnsureIfaceIdOps(client client.Client, iface *Interface, logicalPort string) ([]ovsdb.Operation, error) {
	cur := ""
	if iface.ExternalIDs != nil {
		if v, ok := iface.ExternalIDs["iface-id"]; ok {
			cur = v
		}
	}
	if cur == logicalPort {
		logger.Debugf("[ovs] iface-id already correct on %s", iface.Name)
		return nil, nil
	}

	m := &Interface{
		UUID: iface.UUID,
	}

	ops := make([]ovsdb.Operation, 0, 2)

	if cur != "" {
		delOps, err := client.Where(m).Mutate(m, model.Mutation{
			Field:   &m.ExternalIDs,
			Mutator: ovsdb.MutateOperationDelete,
			Value:   map[string]string{"iface-id": logicalPort},
		})
		if err != nil {
			return nil, fmt.Errorf("build delete iface-id mutate: %w", err)
		}
		ops = append(ops, delOps...)
	}

	if logicalPort != "" {
		insOps, err := client.Where(m).Mutate(m, model.Mutation{
			Field:   &m.ExternalIDs,
			Mutator: ovsdb.MutateOperationInsert,
			Value:   map[string]string{"iface-id": logicalPort},
		})
		if err != nil {
			logger.Errorf("[ovs] build insert iface-id mutate failed: %v", err)
			return nil, fmt.Errorf("build insert iface-id mutate: %w", err)
		}
		ops = append(ops, insOps...)
	}

	return ops, nil
}

func buildDeleteInterfaceOps(client client.Client, ifaceUUID string) ([]ovsdb.Operation, error) {
	if ifaceUUID == "" {
		return nil, fmt.Errorf("delete iface: emty UUID")
	}

	m := &Interface{UUID: ifaceUUID}
	ops, err := client.Where(m).Delete()
	if err != nil {
		logger.Errorf("[ovs] build delete interface op failed: %v", err)
	}
	return ops, nil
}
