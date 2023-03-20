package modules

// crust build/integration-tests
// ./coreum-modules -chain-id=coreum-mainnet-1 -cored-address=full-node-curium.mainnet-1.coreum.dev:9090 -test.run=TestValidatorGrant

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	integrationtests "github.com/CoreumFoundation/coreum/integration-tests"
	"github.com/CoreumFoundation/coreum/pkg/client"
)

func TestValidatorGrants(t *testing.T) {
	const amount = 20_300_000_000
	recipients := []string{
		"core1x20lytyf6zkcrv5edpkfkn8sz578qg5sxnjcgv",
		"core1up3dsjekups4y2rf376a24l2ljtkc8t9t3f4f4",
		"core1ll9gdh5ur6gyv6swgshcm4zkkw4ttakt4ukjma",
		"core1vsdgrlmq87qax45k32ghp94rcl3j03fw6826vt",
		"core1k3wy8ztt2e0uq3j5deukjxu2um4a4z5tku03lr",
		"core1py9v5f7e4aaka55lwtthk30scxhnqfa68ksrte",
		"core1fq63eks78npnsgtja8m8nvvv5qmahqxa5klcfu",
		"core1pe5d6fkmghjrpgk7qps3nqmsnwth7ejucqkuzj",
		"core1zv3rz855lan7mznzlmzycf0a3tslar4lf4wrk5",
		"core19kg8c9k6quujkh0sflp78n4jsp3vzdy8y2t2m9",
		"core1q6mdzggk7feskx3uy90su8sqernswjpcrechnl",
		"core16h0h2cjul7qa3np664ae9gqzrd0apcp8e7wgfw",
		"core1sg0gwgwumymhp0acldjpz7s3k9trgcnndqw593",
		"core1f83v95nmgr5zujarwrf3shen4fydgraahrgwq3",
		"core1flaz3hzgg3tjszl372lu2zz5jsmxd8pv7npmgk",
		"core1f3v67lpkha4hhruncehqzjmt6f6t2fdnxryxvm",
		"core1kepnaw38rymdvq5sstnnytdqqkpd0xxwz289jg",
		"core1wj9qy0fvjawl9agz0ef3e9euw9mjfp0lnjxa5m",
	}

	requireT := require.New(t)

	ctx, chain := integrationtests.NewTestingContext(t)
	codec := chain.ClientContext.Codec()
	keyring := chain.ClientContext.Keyring()

	pubKey1 := &secp256k1.PubKey{}
	pubKey2 := &secp256k1.PubKey{}
	requireT.NoError(codec.UnmarshalJSON([]byte(`{"key":"A2MidxM8OUyemp7UycIVNDR2YxEomyfEYtndydqIuBsV"}`), pubKey1))
	requireT.NoError(codec.UnmarshalJSON([]byte(`{"key":"A3sqCkVWPIHZ64JWwd3rM7Qxj2vjsDfoOJ+JLn7bP4ge"}`), pubKey2))

	multisigKey := multisig.NewLegacyAminoPubKey(
		2,
		[]types.PubKey{pubKey1, pubKey2},
	)
	multisigInfo, err := keyring.SaveMultisig("coreum-foundation-0", multisigKey)
	requireT.NoError(err)

	multisendMsg := &banktypes.MsgMultiSend{
		Inputs: []banktypes.Input{
			{
				Address: multisigInfo.GetAddress().String(),
				Coins:   sdk.NewCoins(chain.NewCoin(sdk.NewIntFromUint64(amount).MulRaw(int64(len(recipients))))),
			},
		},
	}
	for _, r := range recipients {
		multisendMsg.Outputs = append(multisendMsg.Outputs, banktypes.Output{
			Address: r,
			Coins:   sdk.NewCoins(chain.NewCoin(sdk.NewIntFromUint64(amount))),
		})
	}

	acc, err := client.GetAccountInfo(ctx, chain.ClientContext, multisigInfo.GetAddress())
	requireT.NoError(err)

	txBuilder, err := chain.TxFactory().
		WithAccountNumber(acc.GetAccountNumber()).
		WithSequence(acc.GetSequence()).
		WithGas(chain.GasLimitByMsgs(multisendMsg)).
		BuildUnsignedTx(multisendMsg)
	requireT.NoError(err)

	json, err := chain.ClientContext.TxConfig().TxJSONEncoder()(txBuilder.GetTx())
	requireT.NoError(err)

	requireT.NoError(chain.ClientContext.PrintString(fmt.Sprintf("%s\n", json)))
}
