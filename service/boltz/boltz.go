package boltz

import (
	"context"
	"encoding/json"
	"github.com/opendexnetwork/opendex-docker-api/config"
	"github.com/opendexnetwork/opendex-docker-api/service/core"
	docker "github.com/docker/docker/client"
)

type Service struct {
	*core.SingleContainerService
	*RpcClient
}

type Node string

const (
	BTC Node = "btc"
	LTC Node = "ltc"
)

func New(
	name string,
	services map[string]core.Service,
	containerName string,
	dockerClient *docker.Client,
	rpcConfig config.RpcConfig,
) *Service {
	base := core.NewSingleContainerService(name, services, containerName, dockerClient)

	return &Service{
		SingleContainerService: base,
		RpcClient:              NewRpcClient(rpcConfig, base),
	}
}

// {
//  "symbol": "BTC",
//  "lnd_pubkey": "02c882fbd75ba7c0e3175a0b86037b4d056599a694fcfad56589fc05d081b62774",
//  "block_height": 1835961
// }

func (t *Service) GetInfo(node Node) (map[string]interface{}, error) {
	output, err := t.Exec1([]string{"wrapper", string(node), "getinfo"})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type NodeStatus struct {
	Status string
	IsUp   bool
}

func (t *Service) checkNode(node Node) NodeStatus {
	_, err := t.GetInfo(node)
	if err == nil {
		return NodeStatus{Status: string(node) + " up", IsUp: true}
	} else {
		t.logger.Errorf("get info: %s", err)
		return NodeStatus{Status: string(node) + " down", IsUp: false}
	}
}

func (t *Service) GetStatus(ctx context.Context) string {
	status := t.SingleContainerService.GetStatus(ctx)
	if status == "Disabled" {
		return status
	}
	if status != "Container running" {
		if ctx.Value("LauncherState") == "setup" {
			return "Waiting for sync"
		}
		return status
	}

	// container is running

	btcStatus := t.checkNode(BTC)
	ltcStatus := t.checkNode(LTC)

	if btcStatus.IsUp && ltcStatus.IsUp {
		return "Ready"
	} else {
		return btcStatus.Status + "; " + ltcStatus.Status
	}
}

func (t *Service) Close() error {
	err := t.RpcClient.Close()
	if err != nil {
		t.GetLogger().Errorf("Failed to close RPC client: %s", err)
	}
	return nil
}
