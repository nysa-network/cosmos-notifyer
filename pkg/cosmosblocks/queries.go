package cosmosblocks

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/juju/errors"
	"github.com/tendermint/tendermint/libs/bytes"
)

type Validator staking.QueryValidatorResponse

func (v Validator) GetAddress() (bytes.HexBytes, error) {
	pk := ed25519.PubKey{}
	err := pk.Unmarshal(v.Validator.ConsensusPubkey.Value)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return pk.Address(), nil
}

func (c *Client) QueryValidator(valoper string) (*Validator, error) {
	q := staking.QueryValidatorRequest{
		ValidatorAddr: valoper,
	}
	b, err := q.Marshal()
	if err != nil {
		return nil, errors.Trace(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.rpcClient.ABCIQuery(ctx, "/cosmos.staking.v1beta1.Query/Validator", b)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if resp.Response.Value == nil {
		return nil, errors.Errorf("Validator (%s) not found", valoper)
	}

	valResp := staking.QueryValidatorResponse{}
	err = valResp.Unmarshal(resp.Response.Value)
	if err != nil {
		return nil, errors.Trace(err)
	}

	val := Validator(valResp)
	return &val, nil
}
