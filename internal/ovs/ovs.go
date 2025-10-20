package ovs

import (
	"context"
	"time"

	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/yangjie500/cloud-ovs-agent/pkg/logger"
)

func ConnectOVS(ctx context.Context, endpoint string) (client.Client, error) {
	start := time.Now()
	logger.Infof("[ovs] connecting to OVSDB endpoint=%s", endpoint)

	dbModel, err := model.NewClientDBModel("Open_vSwitch", map[string]model.Model{
		"Bridge":    &Bridge{},
		"Port":      &Port{},
		"Interface": &Interface{},
	})

	if err != nil {
		logger.Errorf("[ovs] build ClientDBModel failed: %v", err)
		return nil, err
	}
	logger.Debugf("[ovs] build ClientDBModel ready (tables: Bridge, Port, Interface)")

	ovs, err := client.NewOVSDBClient(dbModel, client.WithEndpoint(endpoint))
	if err != nil {
		logger.Errorf("[ovs] NewOVSDBClient failed: %v", err)
		return nil, err
	}
	logger.Debugf("[ovs] client constructed")

	if err := ovs.Connect(ctx); err != nil {
		logger.Errorf("[ovs] Connect failed: %v", err)
		return nil, err
	}
	logger.Infof("[ovs] TCP/UNIX session established (elasped=%s)", time.Since(start).Truncate(time.Millisecond))

	if _, err := ovs.MonitorAll(ctx); err != nil {
		logger.Errorf("[ovs] MonitorAll failed: %v", err)
		return nil, err
	}

	logger.Infof("[ovs] MonitorAll started")

	return ovs, nil
}
