package main

import (
	"context"
	"flag"
	"fmt"

	test_infra "github.com/pingcap/tipocket/pkg/test-infra"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	crossregion "github.com/pingcap/tipocket/testcase/cross-region"
)

var (
	tsoRequests    = flag.Int("tso-request-count", 2000, "tso requests count for each allocator")
	pdConfTemplate = `
enable-local-tso = true
[labels]
zone = '%v'
`
	plaintextProtocolHeader = "plaintext://"
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}
	fixture.Context.Namespace = "cross-region"
	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: crossregion.ClientCreator{
			Cfg: &crossregion.Config{
				DBName:      "test",
				TSORequests: *tsoRequests,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: provideCrossRegionCluster(),
	}
	suit.Run(context.Background())
}

func provideCrossRegionCluster() cluster.Cluster {
	namespace := "cross-region"
	names := []string{
		"dc-1",
		"dc-2",
		"dc-3",
	}
	confs := []fixture.TiDBClusterConfig{
		provideConf(2, 1, 1, nil, "dc-1"),
		provideConf(2, 1, 1, &fixture.ClusterRef{
			Name:      "dc-1",
			Namespace: namespace,
		}, "dc-2"),
		provideConf(2, 1, 1, &fixture.ClusterRef{
			Name:      "dc-1",
			Namespace: namespace,
		}, "dc-3"),
	}
	return test_infra.NewCrossRegionTestCluster(namespace, names, confs)
}

func provideConf(pdReplicas, kvReplicas, dbReplicas int, ref *fixture.ClusterRef, dcLocation string) fixture.TiDBClusterConfig {
	cloned := fixture.Context.TiDBClusterConfig
	cloned.PDReplicas = pdReplicas
	cloned.TiKVReplicas = kvReplicas
	cloned.TiDBReplicas = dbReplicas
	cloned.PDImage = "hub.pingcap.net/gaosong/pd:c436e988"
	cloned.PDStorageClassName = "shared-nvme-disks"
	cloned.TiKVStorageClassName = "nvme-disks"
	cloned.LogStorageClassName = "shared-sas-disks"
	cloned.Ref = ref
	cloned.PDConfig = fmt.Sprintf("%s%s", plaintextProtocolHeader, fmt.Sprintf(pdConfTemplate, dcLocation))
	return cloned
}
