package keeper_test

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/CoreumFoundation/coreum-tools/pkg/must"
	"github.com/CoreumFoundation/coreum/x/feemodel/keeper"
	"github.com/CoreumFoundation/coreum/x/feemodel/types"
)

func newParamSubspaceMock() *paramSubspaceMock {
	return &paramSubspaceMock{
		params: map[string][]byte{},
	}
}

type paramSubspaceMock struct {
	params map[string][]byte
}

func (psm *paramSubspaceMock) GetParamSet(ctx sdk.Context, ps paramtypes.ParamSet) {
	for _, pair := range ps.ParamSetPairs() {
		must.OK(json.Unmarshal(psm.params[string(pair.Key)], pair.Value))
	}
}

func (psm *paramSubspaceMock) SetParamSet(ctx sdk.Context, ps paramtypes.ParamSet) {
	for _, pair := range ps.ParamSetPairs() {
		psm.params[string(pair.Key)] = must.Bytes(json.Marshal(pair.Value))
	}
}

func setup() (sdk.Context, keeper.Keeper) {
	key := sdk.NewKVStoreKey(types.StoreKey)
	tKey := sdk.NewTransientStoreKey(types.TransientStoreKey)

	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(key, sdk.StoreTypeIAVL, db)
	cms.MountStoreWithDB(tKey, sdk.StoreTypeTransient, db)
	must.OK(cms.LoadLatestVersion())
	ctx := sdk.NewContext(cms, tmproto.Header{}, false, log.NewNopLogger())

	return ctx, keeper.NewKeeper(newParamSubspaceMock(), key, tKey)
}

func TestTrackGas(t *testing.T) {
	ctx, keeper := setup()

	assert.EqualValues(t, 0, keeper.TrackedGas(ctx))

	keeper.TrackGas(ctx, 10)
	assert.EqualValues(t, 10, keeper.TrackedGas(ctx))

	keeper.TrackGas(ctx, 5)
	assert.EqualValues(t, 15, keeper.TrackedGas(ctx))
}

func TestShortEMAGas(t *testing.T) {
	ctx, keeper := setup()

	assert.EqualValues(t, 0, keeper.GetShortEMAGas(ctx))

	keeper.SetShortEMAGas(ctx, 10)
	assert.EqualValues(t, 10, keeper.GetShortEMAGas(ctx))
}

func TestLongEMAGas(t *testing.T) {
	ctx, keeper := setup()

	assert.EqualValues(t, 0, keeper.GetLongEMAGas(ctx))

	keeper.SetLongEMAGas(ctx, 10)
	assert.EqualValues(t, 10, keeper.GetLongEMAGas(ctx))
}

func TestMinGasPrice(t *testing.T) {
	ctx, keeper := setup()

	keeper.SetMinGasPrice(ctx, sdk.NewDecCoin("coin", sdk.NewInt(10)))
	minGasPrice := keeper.GetMinGasPrice(ctx)
	assert.Equal(t, "10.000000000000000000", minGasPrice.Amount.String())
	assert.Equal(t, "coin", minGasPrice.Denom)

	keeper.SetMinGasPrice(ctx, sdk.NewDecCoin("coin", sdk.NewInt(20)))
	minGasPrice = keeper.GetMinGasPrice(ctx)
	assert.EqualValues(t, "20.000000000000000000", minGasPrice.Amount.String())
	assert.Equal(t, "coin", minGasPrice.Denom)
}

func TestParams(t *testing.T) {
	ctx, keeper := setup()

	defParams := types.DefaultParams()
	keeper.SetParams(ctx, defParams)
	params := keeper.GetParams(ctx)

	assert.Equal(t, defParams.Model.InitialGasPrice.String(), params.Model.InitialGasPrice.String())
	assert.Equal(t, defParams.Model.MaxGasPriceMultiplier.String(), params.Model.MaxGasPriceMultiplier.String())
	assert.Equal(t, defParams.Model.MaxDiscount.String(), params.Model.MaxDiscount.String())
	assert.Equal(t, defParams.Model.EscalationStartFraction.String(), params.Model.EscalationStartFraction.String())
	assert.Equal(t, defParams.Model.MaxBlockGas, params.Model.MaxBlockGas)
	assert.Equal(t, defParams.Model.ShortEmaBlockLength, params.Model.ShortEmaBlockLength)
	assert.Equal(t, defParams.Model.LongEmaBlockLength, params.Model.LongEmaBlockLength)
}
