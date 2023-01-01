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

			activeRPC := true

			for {
				rpcs := cosmosblocks.CheckRPCs(chain.RPC)
				rpc := rpcs.GetValidRPCURL()

				if rpc == nil {
					if activeRPC {
						s.notify.Alert(notifyer.AlertMsg{
							Msg: fmt.Sprintf("[%s] No valid RPC (0/%d)", chain.Name, len(rpcs)),
						})
					}
					activeRPC = false
					time.Sleep(time.Second * 5)
					continue
				} else if !activeRPC && rpc != nil {
					s.notify.Recover(notifyer.RecoverMsg{
						Msg: fmt.Sprintf("[%s] RPCs are back up ! ", chain.Name),
					})
				}

				// Start watching
				if err := s.startWatcher(chain, *rpc); err != nil {
					logrus.WithError(err).WithFields(logrus.Fields{
						"chain": chain.Name,
						"rpc":   *rpc,
					}).Error()
				}
			}
		}(chain)
	}
	wg.Wait()
	return nil
}

func (s *service) startWatcher(chain Chain, rpc string) error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	ctx = ctxlogger.WithValue(ctx, "chain", chain.Name)
	ctx = ctxlogger.WithValue(ctx, "rpc", rpc)

	l := ctxlogger.Logger(ctx)
	l.Logger.SetLevel(s.cfg.GetLogLevel())

	c, err := cosmosblocks.NewClient(cosmosblocks.Config{
		RPCEndpoint: rpc,
		Logger:      l,
	})
	if err != nil {
		return errors.Trace(err)
	}

	go func() {
		if err := s.blockHandler(ctx, c, chain); err != nil {
			logrus.WithError(err).Error()
		}
	}()

	if err := c.Start(ctx); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (s *service) blockHandler(ctx context.Context, c *cosmosblocks.Client, chain Chain) error {
	l := ctxlogger.Logger(ctx)

	var (
		latestBlockTime   time.Time = time.Now()
		latestBlockHeight int64     = 0
		missedBlocks      int64     = 0
		missedBlocksAlert int64     = 10

		isJailed bool = false
		isBonded bool = true
	)

START:

	validator, err := c.QueryValidator(chain.ValidatorAddr)
	if err != nil {
		return errors.Errorf("failed to get validator: %s", chain.ValidatorAddr)
	}

	if validator.Validator.IsJailed() {
		if !isJailed {
			isJailed = true
			s.notify.Alert(notifyer.AlertMsg{
				Msg: fmt.Sprintf("[%s] %s is jailed",
					chain.Name, validator.Validator.GetMoniker()),
			})
		}
		time.Sleep(time.Second * 30)
		goto START
	} else if !validator.Validator.IsJailed() && isJailed {
		isJailed = false
		s.notify.Recover(notifyer.RecoverMsg{
			Msg: fmt.Sprintf("[%s] %s is un-jailed",
				chain.Name, validator.Validator.GetMoniker()),
		})
	}

	if !validator.Validator.IsBonded() {
		if isBonded {
			isBonded = false
			s.notify.Alert(notifyer.AlertMsg{
				Msg: fmt.Sprintf("[%s] validator: %s is not in the active set",
					chain.Name, validator.Validator.GetMoniker()),
			})
		}
		time.Sleep(time.Second * 30)
		goto START
	} else if validator.Validator.IsBonded() && !isBonded {
		isBonded = true
		s.notify.Recover(notifyer.RecoverMsg{
			Msg: fmt.Sprintf("[%s] validator: %s is back in the active set",
				chain.Name, validator.Validator.GetMoniker()),
		})
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
				if missedBlocks > missedBlocksAlert {
					missedBlocksAlert += 150
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
				if missedBlocks > 0 && missedBlocks > missedBlocksAlert-150 {
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
