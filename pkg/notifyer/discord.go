package notifyer

import (
	"fmt"

	"github.com/gtuk/discordwebhook"
	"github.com/juju/errors"
)

// DiscordClient is complient with the Service interface
type DiscordClient struct {
	Webhook string
}

func (c DiscordClient) Alert(msg AlertMsg) error {
	username := "cosmos-notifyer"
	content := ":rotating_light: " + msg.Msg

	message := discordwebhook.Message{
		Username: &username,
		Content:  &content,
	}

	err := discordwebhook.SendMessage(c.Webhook, message)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c DiscordClient) Recover(msg RecoverMsg) error {
	username := "cosmos-notifyer"
	content := ":ok_hand: " + msg.Msg

	message := discordwebhook.Message{
		Username: &username,
		Content:  &content,
	}

	err := discordwebhook.SendMessage(c.Webhook, message)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c DiscordClient) Delegation(msg DelegationMsg) error {
	username := "cosmos-notifyer"
	content := fmt.Sprintf(":money_mouth: new delegation of %v %s", msg.Amount, msg.Token)

	message := discordwebhook.Message{
		Username: &username,
		Content:  &content,
	}

	err := discordwebhook.SendMessage(c.Webhook, message)
	if err != nil {
		return errors.Trace(err)
	}
	return nil

}

func (c DiscordClient) UnDelegation(msg UnDelegationMsg) error {
	username := "cosmos-notifyer"
	content := fmt.Sprintf(":money_with_wings: lost delegation of %v %s", msg.Amount, msg.Token)

	message := discordwebhook.Message{
		Username: &username,
		Content:  &content,
	}

	err := discordwebhook.SendMessage(c.Webhook, message)
	if err != nil {
		return errors.Trace(err)
	}
	return nil

}
