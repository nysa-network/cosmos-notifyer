package main

import "github.com/sirupsen/logrus"

type Config struct {
	LogLevel string `yaml:"log_level"`

	Chains []Chain `yaml:"chains"`

	Notifications struct {
		Discord *struct {
			Webhook string `yaml:"webhook"`
		} `yaml:"discord"`
	} `yaml:"notifications"`
}

type Chain struct {
	Name          string   `yaml:"name"`
	ValidatorAddr string   `yaml:"validator_address"`
	RPC           []string `yaml:"rpc"`

	Token struct {
		Label       string `yaml:"label"`
		Coefficient int    `yaml:"coefficient"`
	} `yaml:"token"`

	Notification struct {
		MinimumDelegation float64 `yaml:"minimum_delegation"`
	} `yaml:"notification"`
}

func (cfg Config) GetLogLevel() logrus.Level {
	if cfg.LogLevel == "DEBUG" {
		return logrus.DebugLevel
	} else if cfg.LogLevel == "INFO" {
		return logrus.InfoLevel
	} else if cfg.LogLevel == "WARN" {
		return logrus.WarnLevel
	} else if cfg.LogLevel == "ERROR" {
		return logrus.ErrorLevel
	}
	return logrus.InfoLevel
}

func (c Chain) GetTokenCoefficient() int {
	if c.Token.Coefficient == 0 {
		return 1000000
	}
	return c.Token.Coefficient
}
