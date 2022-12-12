package cosmosblocks

import (
	"context"
)

type RPCStatus struct {
	URL string

	CatchingUp bool

	Err error
}

type RPCStatuses []RPCStatus

func (rpcs RPCStatuses) GetValidRPCURL() *string {
	for _, rpc := range rpcs {
		if rpc.Err == nil && !rpc.CatchingUp {
			return &rpc.URL
		}
	}
	return nil
}

func CheckRPCs(rpcAddrs []string) RPCStatuses {
	statuses := make([]RPCStatus, 0, len(rpcAddrs))

	for _, rpc := range rpcAddrs {
		status := RPCStatus{
			URL: rpc,
		}

		c, err := NewClient(Config{
			RPCEndpoint: rpc,
		})
		if err != nil {
			status.Err = err
			continue
		}
		// TODO: Check chain-id is correct

		rpcStatus, err := c.Status(context.Background())
		if err != nil {
			status.Err = err
			continue
		} else if rpcStatus.SyncInfo.CatchingUp {
			status.CatchingUp = true
		}

		statuses = append(statuses, status)
	}

	return statuses
}
