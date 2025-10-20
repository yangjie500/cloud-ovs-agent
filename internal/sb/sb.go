package sb

import (
	"context"
	"time"

	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/yangjie500/cloud-ovs-agent/pkg/logger"
)

func ConnectSouthBound(ctx context.Context, endpoint string) (client.Client, error) {
	start := time.Now()
	logger.Infof("[sb] connecting to OVN_Southbound endpoint=%s", endpoint)

	dbModel, err := model.NewClientDBModel("OVN_Southbound", map[string]model.Model{
		"Port_Binding": &PortBinding{},
	})
	if err != nil {
		logger.Errorf("[sb] build ClientDBModel failed: %v", err)
		return nil, err
	}

	logger.Debugf("[sb] ClientDBModel ready (tables: Port_Binding)")

	sb, err := client.NewOVSDBClient(dbModel, client.WithEndpoint(endpoint))
	if err != nil {
		logger.Errorf("[sb] NewOVSDBClient failed: %v", err)
		return nil, err
	}
	logger.Debugf("[sb] client constructed")

	if err := sb.Connect(ctx); err != nil {
		logger.Errorf("[sb] Connect failed: %v", err)
		return nil, err
	}
	logger.Infof("[sb] session established (elapsed=%s)", time.Since(start).Truncate(time.Millisecond))

	// sch := sb.Schema()
	// if tbl, ok := sch.Tables["Port_Binding"]; ok {
	// 	for col := range tbl.Columns {
	// 		fmt.Println(col)
	// 	}
	// }

	if _, err := sb.MonitorAll(ctx); err != nil {
		logger.Errorf("[sb] MonitorAll failed: %v", err)
	}

	logger.Infof("[sb] MonitorAll started")

	return sb, nil
}
