package ovs

import (
	"context"
	"fmt"

	"github.com/ovn-kubernetes/libovsdb/client"
)

func findBridgeByName(ctx context.Context, client client.Client, name string) (*Bridge, error) {
	var bridges []Bridge
	if err := client.List(ctx, &bridges); err != nil {
		return nil, fmt.Errorf("list bridges: %w", err)
	}
	for i := range bridges {
		if bridges[i].Name == name {
			return &bridges[i], nil
		}
	}

	return nil, fmt.Errorf("bridge %q not found", name)
}

func findPortByName(ctx context.Context, client client.Client, name string) (*Port, error) {
	var ports []Port
	if err := client.List(ctx, &ports); err != nil {
		return nil, fmt.Errorf("list ports: %w", err)
	}
	for i := range ports {
		if ports[i].Name == name {
			return &ports[i], nil
		}
	}
	return nil, nil
}

func findInterfaceByName(ctx context.Context, client client.Client, name string) (*Interface, error) {
	var ifaces []Interface
	if err := client.List(ctx, &ifaces); err != nil {
		return nil, fmt.Errorf("list ports: %w", err)
	}
	for i := range ifaces {
		if ifaces[i].Name == name {
			return &ifaces[i], nil
		}
	}
	return nil, nil
}

func bridgeHasPort(br *Bridge, portUUID string) bool {
	if br == nil || portUUID == "" || br.Ports == nil {
		return false
	}
	for _, u := range br.Ports {
		if u == portUUID {
			return true
		}
	}
	return false
}
