package main

import (
	"context"
	"sync"
	"time"

	"nysa-network/internal/ctxlogger"
	"nysa-network/pkg/cosmosblocks"
	"nysa-network/pkg/notifyer"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func (s *service) Start(cctx *cli.Context) error {
	s.notify = notifyer.NewClient(notifyer.Config{
		DiscordWebhook: s.cfg.Notifications.Discord.Webhook,
	})

	wg := sync.WaitGroup{}

	for _, chain := range s.cfg.Chains {
		wg.Add(1)

		go func(chain Chain) {
			defer wg.Done()

			s.startWatcher(chain)
		}(chain)
	}
	wg.Wait()
	return nil
}

func (s *service) startWatcher(chain Chain) {
	for {
		for _, rpc := range chain.RPC {
			ctx, cancelFunc := context.WithCancel(context.Background())

			ctx = ctxlogger.WithValue(ctx, "chain", chain.Name)
			ctx = ctxlogger.WithValue(ctx, "rpc", rpc)

			l := ctxlogger.Logger(ctx)
			l.Logger.SetLevel(s.cfg.GetLogLevel())

			c := cosmosblocks.NewClient(cosmosblocks.Config{
				RPCEndpoint: rpc,
				Logger:      l,
			})

			go s.blockHandler(ctx, c, chain)

			if err := c.Start(ctx); err != nil {
				l.WithError(err).Error()
			}
			cancelFunc()
		}
	}

}

func (s *service) blockHandler(ctx context.Context, c *cosmosblocks.Client, chain Chain) {
	l := ctxlogger.Logger(ctx)

	latestBlockTime := time.Now()
	var latestBlockHeight int64 = 0

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if len(c.BlockCh) == cap(c.BlockCh) {
					l.Error("Block channel is full !!")
				}
				if time.Since(latestBlockTime) > time.Second*30 {
					l.Error("no block received since 30s")
				}
				time.Sleep(time.Second * 10)
			}
		}
	}(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case block, ok := <-c.BlockCh:
			if !ok {
				continue
			}
			latestBlockTime = time.Now()
			if latestBlockHeight != 0 && (block.GetHeight()-latestBlockHeight) > 1 {
				l.Errorf("missed block from %d to %d", latestBlockHeight, block.GetHeight())
			}
			latestBlockHeight = block.GetHeight()

			// Check delegations messages
			for _, msg := range block.GetMsgDelegate() {
				if chain.ValidatorAddr == msg.ValoperAddr {
					amount := msg.GetAmount() / float64(chain.GetTokenCoefficient())
					if amount > chain.Notification.MinimumDelegation {
						err := s.notify.Delegation(notifyer.DelegationMsg{
							Amount: amount,
							Token:  chain.Token.Label,
						})
						if err != nil {
							l.WithError(err).WithFields(logrus.Fields{
								"msg":    msg,
								"amount": amount,
								"token":  chain.Token.Label,
							}).Error("Failed to send delegation message")
						}
					}
				}
			}

			// Check for undelegations
			for _, msg := range block.GetMsgUndelegate() {
				if chain.ValidatorAddr == msg.ValoperAddr {
					amount := msg.GetAmount() / float64(chain.GetTokenCoefficient())
					if amount > chain.Notification.MinimumDelegation {
						err := s.notify.UnDelegation(notifyer.UnDelegationMsg{
							Amount: amount,
							Token:  chain.Token.Label,
						})
						if err != nil {
							l.WithError(err).WithFields(logrus.Fields{
								"msg":    msg,
								"amount": amount,
								"token":  chain.Token.Label,
							}).Error("Failed to send undelegation message")
						}
					}
				}
			}
		}
	}
}
