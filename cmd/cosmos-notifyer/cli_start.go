package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"nysa-network/internal/ctxlogger"
	"nysa-network/pkg/cosmosblocks"
	"nysa-network/pkg/notifyer"

	"github.com/juju/errors"
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
	var (
		startTime           time.Time
		lastRPCAlertFiredAt time.Time
	)

	for {
		startTime = time.Now()

		// Loop over all RPCs
		for _, rpc := range chain.RPC {
			ctx, cancelFunc := context.WithCancel(context.Background())

			ctx = ctxlogger.WithValue(ctx, "chain", chain.Name)
			ctx = ctxlogger.WithValue(ctx, "rpc", rpc)

			l := ctxlogger.Logger(ctx)
			l.Logger.SetLevel(s.cfg.GetLogLevel())

			c, err := cosmosblocks.NewClient(cosmosblocks.Config{
				RPCEndpoint: rpc,
				Logger:      l,
			})
			if err != nil {
				l.WithError(err).Error()
				cancelFunc()
				continue
			}

			status, err := c.Status(ctx)
			if err != nil {
				l.WithError(err).Error()
				cancelFunc()
				continue
			} else if status.SyncInfo.CatchingUp {
				l.Error("node is catching-up...")
				cancelFunc()
				continue
			}

			go func() {
				if err := s.blockHandler(ctx, c, chain); err != nil {
					logrus.WithError(err).Error()
				}
			}()

			if err := c.Start(ctx); err != nil {
				l.WithError(err).Error()
			}
			cancelFunc()
		}

		// If we're here, a RPC has failed
		if time.Since(startTime) < time.Second*60 && time.Since(lastRPCAlertFiredAt) > time.Hour*4 {
			lastRPCAlertFiredAt = time.Now()

			err := s.notify.Alert(notifyer.AlertMsg{
				Msg: fmt.Sprintf("[%s] RPCs are down", chain.Name),
			})
			if err != nil {
				logrus.WithError(err).Error("Failed to send alert message: RPC is down")
			}

			go func() {
				for {
					if time.Since(startTime) > time.Second*60*5 {
						err := s.notify.Recover(notifyer.RecoverMsg{
							Msg: fmt.Sprintf("[%s] RPCs are back up!", chain.Name),
						})
						if err != nil {
							logrus.WithError(err).Error("Failed to send recover message: RPC is up")
						}
						return
					}
					time.Sleep(time.Second * 5)
				}

			}()
		}
	}

}

func (s *service) blockHandler(ctx context.Context, c *cosmosblocks.Client, chain Chain) error {
	l := ctxlogger.Logger(ctx)

	var (
		latestBlockTime   time.Time = time.Now()
		latestBlockHeight int64     = 0
		missedBlocks      int64     = 0
		missedBlocksAlert int64     = 10
	)

	validator, err := c.QueryValidator(chain.ValidatorAddr)
	if err != nil {
		return errors.Errorf("failed to get validator: %s", chain.ValidatorAddr)
	}

	validatorAddr, err := validator.GetAddress()
	if err != nil {
		return errors.Errorf("failed to parse validator address: %s", chain.ValidatorAddr)
	}

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
			if latestBlockHeight == 0 {
				return errors.Errorf("no blocks received")
			}
			return nil
		case block, ok := <-c.BlockCh:
			if !ok {
				continue
			}
			latestBlockTime = time.Now()
			if latestBlockHeight != 0 && (block.GetHeight()-latestBlockHeight) > 1 {
				l.Errorf("missed block from %d to %d", latestBlockHeight, block.GetHeight())
			}
			latestBlockHeight = block.GetHeight()

			// Check validator signed block
			if !block.IsValidatorSigned(validatorAddr) {
				l.Error("Validator didn't signed block")
				missedBlocks += 1
				missedBlocksAlert += 150
				if missedBlocks > missedBlocksAlert {
					err := s.notify.Alert(notifyer.AlertMsg{
						Msg: fmt.Sprintf("[%s] %s missed %d blocks",
							chain.Name, validator.Validator.GetMoniker(), missedBlocks),
					})
					if err != nil {
						l.WithError(err).WithFields(logrus.Fields{
							"token": chain.Token.Label,
						}).Error("Failed to send alert message")
					}
				}
			} else {
				if missedBlocks != 0 && missedBlocks < missedBlocksAlert {
					s.notify.Recover(notifyer.RecoverMsg{
						Msg: fmt.Sprintf("[%s] Signing block again, missed blocks: %d", chain.Name, missedBlocks),
					})
				}
				missedBlocks = 0
				missedBlocksAlert = 10
			}

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
