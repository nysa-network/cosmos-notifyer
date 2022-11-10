package main

type Config struct {
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

func (c Chain) GetTokenCoefficient() int {
	if c.Token.Coefficient == 0 {
		return 1000000
	}
	return c.Token.Coefficient
}
