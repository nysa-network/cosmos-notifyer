package notifyer

import (
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
)

type Service interface {
	Alert(msg AlertMsg) error
	Delegation(msg DelegationMsg) error
	UnDelegation(msg UnDelegationMsg) error
}

// Client is complient with the Service interface
type Client struct {
	Service

	cfg Config

	discordClient *DiscordClient
}

// Config is Client configuration
type Config struct {
	DiscordWebhook string
}

// NewClient return a notifyer.Client compatible with Service interface
func NewClient(cfg Config) *Client {
	c := Client{
		cfg: cfg,
	}

	if cfg.DiscordWebhook != "" {
		c.discordClient = &DiscordClient{
			Webhook: cfg.DiscordWebhook,
		}
	}
	return &c
}

type AlertMsg struct {
	Msg string
}

func (c Client) Alert(msg AlertMsg) error {
	var errs error

	if c.discordClient != nil {
		if err := c.discordClient.Alert(msg); err != nil {
			errs = errors.Wrap(errs, err)
		}
	}
	if errs != nil {
		logrus.WithError(errs).Error()
	}
	return nil
}

type RecoverMsg struct {
	Msg string
}

func (c Client) Recover(msg RecoverMsg) error {
	var errs error

	if c.discordClient != nil {
		if err := c.discordClient.Recover(msg); err != nil {
			errs = errors.Wrap(errs, err)
		}
	}
	if errs != nil {
		logrus.WithError(errs).Error()
	}
	return nil
}

type DelegationMsg struct {
	Amount float64
	Token  string
}

func (c Client) Delegation(msg DelegationMsg) error {
	var errs error

	if c.discordClient != nil {
		if err := c.discordClient.Delegation(msg); err != nil {
			errs = errors.Wrap(errs, err)
		}
	}
	if errs != nil {
		logrus.WithError(errs).Error()
	}
	return nil
}

type UnDelegationMsg struct {
	Amount float64
	Token  string
}

func (c Client) UnDelegation(msg UnDelegationMsg) error {
	var errs error

	if c.discordClient != nil {
		if err := c.discordClient.UnDelegation(msg); err != nil {
			errs = errors.Wrap(errs, err)
		}
	}
	if errs != nil {
		logrus.WithError(errs).Error()
	}
	return nil
}
