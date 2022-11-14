package cosmosblocks

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Block struct {
	Event *tmtypes.EventDataNewBlock
	Logs  []*sdk.ABCIMessageLogs
}

func (b Block) GetHeight() int64 {
	return b.Event.Block.Height
}

func (b Block) IsValidatorSigned(valconsAddr []byte) bool {
	for _, sig := range b.Event.Block.LastCommit.Signatures {
		if bytes.Equal(sig.ValidatorAddress.Bytes(), valconsAddr) {
			return true
		}
	}
	return false
}

func (b Block) GetValidators() ([]*tmtypes.Address, error) {
	addrs := make([]*tmtypes.Address, 0, len(b.Event.Block.LastCommit.Signatures))

	for _, sig := range b.Event.Block.LastCommit.Signatures {
		addrs = append(addrs, &sig.ValidatorAddress)
	}
	return addrs, nil
}

func (b Block) GetMsgDelegate() []*MsgDelegate {
	ret := make([]*MsgDelegate, 0)

	for _, log0 := range b.Logs {
		if log0 == nil {
			continue
		}
		for _, log := range *log0 {
			for _, e := range log.Events {
				if e.Type == "delegate" {
					attributes := e.GetAttributes()

					for i := 0; i < len(attributes); {
						msg := &MsgDelegate{
							ValoperAddr: attributes[i].Value,
							Amount:      attributes[i+1].Value,
							Share:       attributes[i+2].Value,
						}
						ret = append(ret, msg)

						for i+1 < len(attributes) && attributes[i].Key != attributes[i+1].Key {
							i += 1
						}
						i++
					}
				}
			}
		}
	}
	return ret
}

func (b Block) GetMsgUndelegate() []*MsgUndelegate {
	ret := make([]*MsgUndelegate, 0, 10)

	for _, log0 := range b.Logs {
		if log0 == nil {
			continue
		}
		for _, log := range *log0 {
			for _, e := range log.Events {

				if e.Type == "redelegate" {
					attributes := e.GetAttributes()

					for i := 0; i < len(attributes); {
						msg := MsgUndelegate{
							ValoperAddr: attributes[i].Value,
							Amount:      attributes[i+2].Value,
						}
						ret = append(ret, &msg)

						for i+1 < len(attributes) && attributes[i].Key != attributes[i+1].Key {
							i += 1
						}
						i++
					}
				} else if e.Type == "unbond" {
					attributes := e.GetAttributes()

					for i := 0; i < len(attributes); {
						msg := &MsgUndelegate{
							ValoperAddr: attributes[i].Value,
							Amount:      attributes[i+1].Value,
						}
						ret = append(ret, msg)

						for i+1 < len(attributes) && attributes[i].Key != attributes[i+1].Key {
							i += 1
						}
						i++

					}
				}
			}
		}
	}

	return ret
}
