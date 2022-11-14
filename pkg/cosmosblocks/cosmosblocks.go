package cosmosblocks

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
	tmjson "github.com/tendermint/tendermint/libs/json"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	libclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Client struct {
	Config

	BlockCh chan *Block

	ws        *libclient.WSClient
	rpcClient *rpcclient.HTTP

	log *logrus.Entry
}

type Config struct {
	RPCEndpoint string

	Logger *logrus.Entry
}

func NewClient(cfg Config) (*Client, error) {
	c := &Client{
		Config:  cfg,
		BlockCh: make(chan *Block, 10),
	}

	c.log = cfg.Logger
	if c.log == nil {
		c.log = logrus.NewEntry(logrus.New())
	}

	var err error
	c.rpcClient, err = rpcclient.New(c.RPCEndpoint, "/websocket")
	if err != nil {
		return nil, errors.Trace(err)
	}
	return c, nil
}

func (c Client) Start(ctx context.Context) error {
	wsClient, err := libclient.NewWS(c.RPCEndpoint, "/websocket",
		libclient.MaxReconnectAttempts(256),
		libclient.ReadWait(120*time.Second),
		libclient.WriteWait(120*time.Second),
		libclient.PingPeriod(5*time.Second),
		libclient.OnReconnect(func() {
			c.log.Warn("reconnecting")
		}))
	if err != nil {
		return errors.Trace(err)
	}

	if err := wsClient.Start(); err != nil {
		return errors.Trace(err)
	}

	defer func() {
		if err := wsClient.Stop(); err != nil {
			c.log.WithError(err).Error()
		}
	}()

	err = wsClient.Subscribe(context.TODO(), `tm.event='NewBlock'`)
	if err != nil {
		return errors.Trace(err)
	}

	latestBlockTime := time.Now()
	checkTime := time.Second * 45
	// go func() {
	// 	for {
	// 		if time.Since(latestBlockTime) > time.Second*45 {
	// 			c.log.Error("No block receive from tendermint since more than 45s")
	// 		}
	// 		time.Sleep(time.Second * 5)
	// 	}
	// }()

	c.log.Info("Start listening blocks...")

	for {
		select {
		case <-wsClient.Quit():
			return errors.New("websocket client: quit")
		case <-ctx.Done():
			return nil
		case block, ok := <-wsClient.ResponsesCh:
			latestBlockTime = time.Now()

			// c.log.Info("receive block from tendermint")
			if !ok {
				c.log.Error("block is not ok")
				continue
			}
			result := ctypes.ResultEvent{}

			if err = tmjson.Unmarshal(block.Result, &result); err != nil {
				c.log.WithError(err).Error()
				continue
			}

			eventBlock, ok := result.Data.(tmtypes.EventDataNewBlock)
			if !ok {
				if result.Data != nil {
					c.log.Error("result is not type: EventDataNewBlock")
				}
				continue
			}

			b := Block{
				Event: &eventBlock,
			}
			b.Logs = make([]*sdk.ABCIMessageLogs, 0, len(eventBlock.Block.Data.Txs))

			l := c.log.WithFields(logrus.Fields{
				"height":  b.GetHeight(),
				"txs.len": len(eventBlock.Block.Data.Txs),
			})
			l.Info()

			for _, blockTX := range eventBlock.Block.Data.Txs {
				time.Sleep(time.Second / 4)

				res, err := c.rpcClient.Tx(context.Background(), blockTX.Hash(), true)
				if err != nil {
					// just wait and retry once
					time.Sleep(time.Second / 4)
					res, err = c.rpcClient.Tx(context.Background(), blockTX.Hash(), true)
					if err != nil {
						l.WithError(err).Errorf("Fail to parse tx: %s", blockTX.String())
						continue
					}
				}

				logs, err := sdk.ParseABCILogs(res.TxResult.Log)
				if err != nil {
					continue
				}
				b.Logs = append(b.Logs, &logs)
			}

			c.BlockCh <- &b

		default:
			if time.Since(latestBlockTime) > checkTime {
				c.log.Errorf("No block receive from tendermint since %s", checkTime.String())
				close(c.BlockCh)
				return nil
			}
		}
	}
}
