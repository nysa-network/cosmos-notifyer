package cosmosblocks

import (
	"regexp"
	"strconv"
)

var (
	regexKeepNumeric = regexp.MustCompile(`[^0-9]+`)
)

type MsgDelegate struct {
	// TODO: In  Events messages, there is a staking
	// From string

	ValoperAddr string
	Amount      string
	Share       string
}

func (msg MsgDelegate) GetAmount() float64 {
	amountFloat, _ := strconv.ParseFloat(msg.Share, 64)
	return amountFloat
}

// MsgUndelegate contain result from "redelegate" and "unbond"
type MsgUndelegate struct {
	ValoperAddr string
	Amount      string
}

func (msg MsgUndelegate) GetAmount() float64 {
	amount := regexKeepNumeric.ReplaceAllString(msg.Amount, "")
	amountFloat, _ := strconv.ParseFloat(amount, 64)
	return amountFloat
}
