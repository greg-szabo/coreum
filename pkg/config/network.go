package config

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcosmostypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/pkg/errors"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/CoreumFoundation/coreum/pkg/config/constant"
	feemodeltypes "github.com/CoreumFoundation/coreum/x/feemodel/types"
)

var (
	//go:embed genesis/genesis.tmpl.json
	genesisTemplate string

	//go:embed genesis/gentx/coreum-mainnet-1
	mainGenTxsFS embed.FS

	//go:embed genesis/gentx/coreum-testnet-1
	testGenTxsFS embed.FS

	//go:embed genesis/gentx/coreum-devnet-1
	devGenTxsFS embed.FS

	networkConfigs map[constant.ChainID]NetworkConfig
)

//nolint:funlen
func init() {
	// common vars
	var (
		feeConfig = FeeConfig{
			FeeModel: feemodeltypes.DefaultModel(),
		}

		govConfig = GovConfig{
			ProposalConfig: GovProposalConfig{
				MinDepositAmount: "4000000000", // 4,000 CORE
				VotingPeriod:     "4h",         // 4 hours
			},
		}

		stakingConfig = StakingConfig{
			UnbondingTime: "168h", // 7 days
			MaxValidators: 32,
		}

		customParamsConfig = CustomParamsConfig{
			Staking: CustomParamsStakingConfig{
				MinSelfDelegation: sdk.NewInt(20_000_000_000), // 20k core
			},
		}

		assetFTConfig = AssetFTConfig{
			IssueFee: sdk.NewIntFromUint64(10_000_000),
		}

		assetNFTConfig = AssetNFTConfig{
			MintFee: sdk.ZeroInt(),
		}
	)

	// mainnet vars

	// CORE allocation:
	// 500M = (5 * 25_000 + 700_000 + 99_175_000) + 4 * 100_000_000
	// In total 11 wallets will be created in genesis:
	// 5 * 25_000 is a balance of each of 4 wallets used to create genesis validators & 1 wallet to join network as validator after launch.
	// 700_000 is a balance of bridge wallet.
	// 99_175_000 is a balance of foundation-0 wallet (99_175_000 = 100_000_000 - (5 * 25_000 + 700_000)).
	// 4 * 100_000_000 is a balance of each of remaining 4 foundation wallets.
	mainGenesisValidatorCreatorBalance := sdk.NewCoins(sdk.NewCoin(constant.DenomMain, sdk.NewInt(25_000_000_000)))

	mainBridgeBalance := sdk.NewCoins(sdk.NewCoin(constant.DenomMain, sdk.NewInt(700_000_000_000)))

	mainFoundationZeroInitialBalance := sdk.NewCoins(sdk.NewCoin(constant.DenomMain, sdk.NewInt(99_175_000_000_000)))
	mainFoundationOtherInitialBalance := sdk.NewCoins(sdk.NewCoin(constant.DenomMain, sdk.NewInt(100_000_000_000_000)))

	// testnet vars

	// 500M = 4 * (124_950_000 + 50_000)
	// where 124_950_000 is a balance of each of 4 initial foundation wallets.
	// where 50_000 is balances of each of 4 initial validator stakers.
	testFoundationInitialBalance := sdk.NewCoins(sdk.NewCoin(constant.DenomTest, sdk.NewInt(124_950_000_000_000)))
	testStakerValidatorBalance := sdk.NewCoins(sdk.NewCoin(constant.DenomTest, sdk.NewInt(50_000_000_000)))

	testGovConfig := govConfig
	testGovConfig.ProposalConfig.VotingPeriod = "24h"

	// devnet vars

	// 10m delegated and 1m extra to the txs
	devStakerValidatorBalance := sdk.NewCoins(sdk.NewCoin(constant.DenomDev, sdk.NewInt(11_000_000_000_000)))

	devGovConfig := govConfig
	devGovConfig.ProposalConfig.VotingPeriod = "24h"

	// configs
	networkConfigs = map[constant.ChainID]NetworkConfig{
		constant.ChainIDMain: {
			ChainID:              constant.ChainIDMain,
			GenesisTime:          time.Date(2023, 3, 12, 0, 0, 0, 0, time.UTC),
			AddressPrefix:        constant.AddressPrefixMain,
			MetadataDisplayDenom: constant.DenomMainDisplay,
			Denom:                constant.DenomMain,
			Fee:                  feeConfig,
			NodeConfig: NodeConfig{
				SeedPeers: []string{
					"0df493af80fbaad41b9b26d6f4520b39ceb1d210@34.171.208.193:26656", // seed-iron
					"cba16f4f32707d70a2a2d10861fac897f1e9aaa1@34.72.150.107:26656",  // seed-nickle
				},
			},
			GovConfig:          govConfig,
			StakingConfig:      stakingConfig,
			CustomParamsConfig: customParamsConfig,
			AssetFTConfig:      assetFTConfig,
			AssetNFTConfig:     assetNFTConfig,
			FundedAccounts: []FundedAccount{
				// coreum-krypton genesis-validators-creator: 25k
				{
					Address:  "core1d5wqdp322zn5jyn5mszrrstg2xuq35xyrhsc9f",
					Balances: mainGenesisValidatorCreatorBalance,
				},
				// coreum-neon genesis-validators-creator: 25k
				{
					Address:  "core15cpygjlf7pgfnqlc8uz9eryspwd0pwk3xrup8h",
					Balances: mainGenesisValidatorCreatorBalance,
				},
				// coreum-radon genesis-validators-creator: 25k
				{
					Address:  "core1zmhfe2hh4qmg54gpsyw8n35gayx3a85mqlfzgk",
					Balances: mainGenesisValidatorCreatorBalance,
				},
				// coreum-xenon genesis-validators-creator: 25k
				{
					Address:  "core1hsmhywnkehyyv8muzswhdumzztae4hq4k3dj8p",
					Balances: mainGenesisValidatorCreatorBalance,
				},
				// coreum-argon validators-creator: 25k
				{
					Address:  "core1nc84mnnqshaln65vsykr63m605sc4kvdwnkgg9",
					Balances: mainGenesisValidatorCreatorBalance,
				},
				// bridge: 700k
				{
					Address:  "core1ssh2d2ft6hzrgn9z6k7mmsamy2hfpxl9y8re5x",
					Balances: mainBridgeBalance,
				},
				// coreum-foundation-0: 99_175_000
				{
					Address:  "core13xmyzhvl02xpz0pu8v9mqalsvpyy7wvs9q5f90",
					Balances: mainFoundationZeroInitialBalance,
				},
				// coreum-foundation-1: 100M
				{
					Address:  "core14g6wpzdx8g9txvxxu3fl7fplal9y5ztx34ac5p",
					Balances: mainFoundationOtherInitialBalance,
				},
				// coreum-foundation-2: 100M
				{
					Address:  "core1zn2ns3ls68jlsv5dgkuz0rxsxt5fhk7n9cfl23",
					Balances: mainFoundationOtherInitialBalance,
				},
				// coreum-foundation-3: 100M
				{
					Address:  "core1p4gsfkmqm0uxua65phteqwnmu39fwjvtspfkcj",
					Balances: mainFoundationOtherInitialBalance,
				},
				// coreum-foundation-4: 100M
				{
					Address:  "core1rddqzjzy4f5frxkhds3sux0m03encqtla3ayu9",
					Balances: mainFoundationOtherInitialBalance,
				},
			},
			GenTxs: readGenTxs(mainGenTxsFS),
		},
		constant.ChainIDTest: {
			ChainID:              constant.ChainIDTest,
			GenesisTime:          time.Date(2023, 1, 16, 12, 0, 0, 0, time.UTC),
			AddressPrefix:        constant.AddressPrefixTest,
			MetadataDisplayDenom: constant.DenomTestDisplay,
			Denom:                constant.DenomTest,
			Fee:                  feeConfig,
			NodeConfig: NodeConfig{
				SeedPeers: []string{
					"64391878009b8804d90fda13805e45041f492155@35.232.157.206:26656", // seed-sirius
					"53f2367d8f8291af8e3b6ca60efded0675ff6314@34.29.15.170:26656",   // seed-antares
				},
			},
			GovConfig:          testGovConfig,
			StakingConfig:      stakingConfig,
			CustomParamsConfig: customParamsConfig,
			AssetFTConfig:      assetFTConfig,
			AssetNFTConfig:     assetNFTConfig,
			FundedAccounts: []FundedAccount{
				// Accounts used to create initial validators to bootstrap chain.
				// validator-1-creator
				{
					Address:  "testcore1wf67lcjnp7mhr3ntjct7fdsex3e4h6jaxxp5aa",
					Balances: testStakerValidatorBalance,
				},
				// validator-2-creator
				{
					Address:  "testcore1snrwnty4h4rrnghd4s9m2xklrk7qu42haygce6",
					Balances: testStakerValidatorBalance,
				},
				// validator-3-creator
				{
					Address:  "testcore14x4ux30sadvg90k2xd8fte5vnhhh0uvkxrrn9j",
					Balances: testStakerValidatorBalance,
				},
				// validator-4-creator
				{
					Address:  "testcore1adst6w4e79tddzhcgaru2l2gms8jjep6a4caa7",
					Balances: testStakerValidatorBalance,
				},

				// Accounts storing remaining total supply of the chain. Created as single signature accounts initially and will be
				// transferred to management after chain start.
				// foundation-initial-1
				{
					Address:  "testcore1efkcsd94u0vrx8rgq9cktjgq7fgwrjap3qu289",
					Balances: testFoundationInitialBalance,
				},
				// foundation-initial-2
				{
					Address:  "testcore18nfwg708vu74e6mrcu6yjdzcdq5608rmvavt05",
					Balances: testFoundationInitialBalance,
				},
				// foundation-initial-3
				{
					Address:  "testcore1qrqhjrc2jl36l4vuvhvjlt6kg6d0xqazzlxek7",
					Balances: testFoundationInitialBalance,
				},
				// foundation-initial-4
				{
					Address:  "testcore12guwnjehw06c9r40knd0js5dn6g924p7xxg48h",
					Balances: testFoundationInitialBalance,
				},
			},
			GenTxs: readGenTxs(testGenTxsFS),
		},
		constant.ChainIDDev: {
			ChainID:              constant.ChainIDDev,
			GenesisTime:          time.Date(2022, 6, 27, 12, 0, 0, 0, time.UTC),
			AddressPrefix:        constant.AddressPrefixDev,
			MetadataDisplayDenom: constant.DenomDevDisplay,
			Denom:                constant.DenomDev,
			Fee:                  feeConfig,
			NodeConfig: NodeConfig{
				SeedPeers: []string{
					"602df7489bd45626af5c9a4ea7f700ceb2222b19@34.135.242.117:26656",
					"88d1266e086bfe33589886cc10d4c58e85a69d14@34.135.191.69:26656",
				},
			},
			GovConfig:          devGovConfig,
			StakingConfig:      stakingConfig,
			CustomParamsConfig: customParamsConfig,
			AssetFTConfig:      assetFTConfig,
			AssetNFTConfig:     assetNFTConfig,
			FundedAccounts: []FundedAccount{
				// Staker of validator Mercury
				{
					Address:  "devcore15eqsya33vx9p5zt7ad8fg3k674tlsllk3pvqp6",
					Balances: devStakerValidatorBalance,
				},
				// Staker of validator Venus
				{
					Address:  "devcore105ct3vl89ar53jrj23zl6e09cmqwym2ua5hegf",
					Balances: devStakerValidatorBalance,
				},
				// Staker of validator Earth
				{
					Address:  "devcore14x46r5eflga696sd5my900euvlplu2prhny5ae",
					Balances: devStakerValidatorBalance,
				},
				// Faucet's account storing the rest of total supply
				{
					Address:  "devcore1ckuncyw0hftdq5qfjs6ee2v6z73sq0urd390cd",
					Balances: sdk.NewCoins(sdk.NewCoin(constant.DenomDev, sdk.NewInt(100_000_000_000_000))), // 100m faucet
				},
			},
			GenTxs: readGenTxs(devGenTxsFS),
		},
	}
}

func readGenTxs(genTxsFs fs.FS) []json.RawMessage {
	genTxs := make([]json.RawMessage, 0)
	err := fs.WalkDir(genTxsFs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			panic("can't open GenTxs FS")
		}
		if d.IsDir() {
			return nil
		}

		file, err := genTxsFs.Open(path)
		if err != nil {
			panic(fmt.Sprintf("can't open file %q from GenTxs FS", path))
		}
		defer file.Close()
		txBytes, err := io.ReadAll(file)
		if err != nil {
			panic(fmt.Sprintf("can't read file %+v from GenTxs FS", file))
		}
		genTxs = append(genTxs, txBytes)
		return nil
	})
	if err != nil {
		panic("can't read files from GenTxs FS")
	}

	return genTxs
}

// Networks returns slice of available networks.
func Networks() []Network {
	networks := make([]Network, 0, len(networkConfigs))
	for _, nc := range networkConfigs {
		networks = append(networks, NewNetwork(nc))
	}
	return networks
}

// FeeConfig is the part of network config defining parameters of our fee model.
type FeeConfig struct {
	FeeModel feemodeltypes.Model
}

// GovConfig contains gov module configs.
type GovConfig struct {
	ProposalConfig GovProposalConfig
}

// GovProposalConfig contains gov module proposal-related configuration options.
type GovProposalConfig struct {
	// MinDepositAmount is the minimum amount needed to create a proposal. Basically anti-spam policy.
	MinDepositAmount string

	// VotingPeriod is the proposal voting period duration.
	VotingPeriod string
}

// StakingConfig contains staking module configuration.
type StakingConfig struct {
	// UnbondingTime is the time duration after which bonded coins will become to be released
	UnbondingTime string

	// MaxValidators is the maximum number of validators that could be created
	MaxValidators int
}

// CustomParamsStakingConfig contains custom params for the staking module configuration.
type CustomParamsStakingConfig struct {
	// MinSelfDelegation is the minimum allowed amount of the stake coin for the validator to be created.
	MinSelfDelegation sdk.Int
}

// CustomParamsConfig contains custom params module configuration.
type CustomParamsConfig struct {
	Staking CustomParamsStakingConfig
}

// AssetFTConfig is the part of network config defining parameters of ft assets.
type AssetFTConfig struct {
	IssueFee sdk.Int
}

// AssetNFTConfig is the part of network config defining parameters of nft assets.
type AssetNFTConfig struct {
	MintFee sdk.Int
}

// NetworkConfig helps initialize Network instance.
type NetworkConfig struct {
	ChainID              constant.ChainID
	GenesisTime          time.Time
	AddressPrefix        string
	MetadataDisplayDenom string
	Denom                string
	Fee                  FeeConfig
	FundedAccounts       []FundedAccount
	GenTxs               []json.RawMessage
	NodeConfig           NodeConfig
	GovConfig            GovConfig
	StakingConfig        StakingConfig
	CustomParamsConfig   CustomParamsConfig
	AssetFTConfig        AssetFTConfig
	AssetNFTConfig       AssetNFTConfig
}

// Network holds all the configuration for different predefined networks.
type Network struct {
	chainID              constant.ChainID
	genesisTime          time.Time
	addressPrefix        string
	metadataDisplayDenom string
	denom                string
	fee                  FeeConfig
	nodeConfig           NodeConfig
	gov                  GovConfig
	staking              StakingConfig
	customParams         CustomParamsConfig
	assetFT              AssetFTConfig
	assetNFT             AssetNFTConfig

	mu             *sync.Mutex
	fundedAccounts []FundedAccount
	genTxs         []json.RawMessage
}

// NewNetwork returns a new instance of Network.
func NewNetwork(c NetworkConfig) Network {
	n := Network{
		genesisTime:          c.GenesisTime,
		chainID:              c.ChainID,
		addressPrefix:        c.AddressPrefix,
		metadataDisplayDenom: c.MetadataDisplayDenom,
		denom:                c.Denom,
		nodeConfig:           c.NodeConfig.Clone(),
		fee:                  c.Fee,
		gov:                  c.GovConfig,
		staking:              c.StakingConfig,
		customParams:         c.CustomParamsConfig,
		assetFT:              c.AssetFTConfig,
		assetNFT:             c.AssetNFTConfig,
		mu:                   &sync.Mutex{},
		fundedAccounts:       append([]FundedAccount{}, c.FundedAccounts...),
		genTxs:               append([]json.RawMessage{}, c.GenTxs...),
	}

	return n
}

// FundedAccount is used to provide information about prefunded
// accounts in network config.
type FundedAccount struct {
	// we can't use the sdk.AccAddress because of configurable prefixes
	Address  string
	Balances sdk.Coins
}

func validateNoDuplicateFundedAccounts(accounts []FundedAccount) error {
	accountsIndexMap := map[string]interface{}{}
	for _, fundEntry := range accounts {
		fundEntry := fundEntry
		_, exists := accountsIndexMap[fundEntry.Address]
		if exists {
			return errors.New("duplicate funded account is not allowed")
		}
		accountsIndexMap[fundEntry.Address] = true
	}

	return nil
}

// FundAccount funds address with balances at genesis.
func (n *Network) FundAccount(accAddress sdk.AccAddress, balances sdk.Coins) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.fundedAccounts = append(n.fundedAccounts, FundedAccount{
		Address:  accAddress.String(),
		Balances: balances,
	})
	return nil
}

// NodeConfig returns NodeConfig.
func (n *Network) NodeConfig() *NodeConfig {
	nodeConfig := n.nodeConfig.Clone()
	return &nodeConfig
}

// AddGenesisTx adds transaction to the genesis file.
func (n *Network) AddGenesisTx(signedTx json.RawMessage) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.genTxs = append(n.genTxs, signedTx)
}

func applyFundedAccountToGenesis(
	fa FundedAccount,
	accountState authcosmostypes.GenesisAccounts,
	bankState *banktypes.GenesisState,
) authcosmostypes.GenesisAccounts {
	accountAddress := sdk.MustAccAddressFromBech32(fa.Address)
	accountState = append(accountState, authcosmostypes.NewBaseAccount(accountAddress, nil, 0, 0))
	coins := fa.Balances
	bankState.Balances = append(
		bankState.Balances,
		banktypes.Balance{Address: accountAddress.String(), Coins: coins},
	)
	bankState.Supply = bankState.Supply.Add(coins...)

	return accountState
}

// GenesisDoc returns the genesis doc of the network.
func (n Network) GenesisDoc() (*tmtypes.GenesisDoc, error) {
	codec := NewEncodingConfig(module.NewBasicManager(
		auth.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
	)).Codec

	genesisJSON, err := genesisByTemplate(n)
	if err != nil {
		return nil, errors.Wrap(err, "not able get genesis")
	}

	genesisDoc, err := tmtypes.GenesisDocFromJSON(genesisJSON)
	if err != nil {
		return nil, errors.Wrap(err, "not able to parse genesis json bytes")
	}
	var appState map[string]json.RawMessage

	if err = json.Unmarshal(genesisDoc.AppState, &appState); err != nil {
		return nil, errors.Wrap(err, "not able to parse genesis app state")
	}

	authState := authcosmostypes.GetGenesisStateFromAppState(codec, appState)
	accountState, err := authcosmostypes.UnpackAccounts(authState.Accounts)
	if err != nil {
		return nil, errors.Wrap(err, "not able to unpack auth accounts")
	}

	genutilState := genutiltypes.GetGenesisStateFromAppState(codec, appState)
	bankState := banktypes.GetGenesisStateFromAppState(codec, appState)

	n.mu.Lock()
	defer n.mu.Unlock()

	if err := validateNoDuplicateFundedAccounts(n.fundedAccounts); err != nil {
		return nil, err
	}

	for _, fundedAcc := range n.fundedAccounts {
		accountState = applyFundedAccountToGenesis(fundedAcc, accountState, bankState)
	}

	genutilState.GenTxs = append(genutilState.GenTxs, n.genTxs...)

	genutiltypes.SetGenesisStateInAppState(codec, appState, genutilState)
	authState.Accounts, err = authcosmostypes.PackAccounts(authcosmostypes.SanitizeGenesisAccounts(accountState))
	if err != nil {
		return nil, errors.Wrap(err, "not able to sanitize and pack accounts")
	}
	appState[authcosmostypes.ModuleName] = codec.MustMarshalJSON(&authState)

	bankState.Balances = banktypes.SanitizeGenesisBalances(bankState.Balances)
	appState[banktypes.ModuleName] = codec.MustMarshalJSON(bankState)

	genesisDoc.AppState, err = json.MarshalIndent(appState, "", "  ")
	if err != nil {
		return nil, err
	}

	return genesisDoc, nil
}

// EncodeGenesis returns the json encoded representation of the genesis file.
func (n Network) EncodeGenesis() ([]byte, error) {
	genesisDoc, err := n.GenesisDoc()
	if err != nil {
		return nil, errors.Wrap(err, "not able to get genesis doc")
	}

	bs, err := tmjson.MarshalIndent(genesisDoc, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "not able to marshal genesis doc")
	}

	return bs, nil
}

// SaveGenesis saves json encoded representation of the genesis config into file.
func (n Network) SaveGenesis(homeDir string) error {
	genDocBytes, err := n.EncodeGenesis()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(homeDir+"/config", 0o700); err != nil {
		return errors.Wrap(err, "unable to make config directory")
	}

	err = os.WriteFile(homeDir+"/config/genesis.json", genDocBytes, 0644)
	return errors.Wrap(err, "unable to write genesis bytes to file")
}

// SetSDKConfig sets global SDK config to some network-specific values.
// In typical applications this func should be called right after network initialization.
func (n Network) SetSDKConfig() {
	config := sdk.GetConfig()

	// Set address & public key prefixes
	config.SetBech32PrefixForAccount(n.addressPrefix, n.addressPrefix+"pub")
	config.SetBech32PrefixForValidator(n.addressPrefix+"valoper", n.addressPrefix+"valoperpub")
	config.SetBech32PrefixForConsensusNode(n.addressPrefix+"valcons", n.addressPrefix+"valconspub")

	// Set BIP44 coin type corresponding to CORE
	config.SetCoinType(constant.CoinType)

	config.Seal()
}

// AddressPrefix returns the address prefix to be used in network config.
func (n Network) AddressPrefix() string {
	return n.addressPrefix
}

// ChainID returns the chain ID used in network config.
func (n Network) ChainID() constant.ChainID {
	return n.chainID
}

// GenesisTime returns the genesis time of the network.
func (n Network) GenesisTime() time.Time {
	return n.genesisTime
}

// FundedAccounts returns the funded accounts.
func (n Network) FundedAccounts() []FundedAccount {
	n.mu.Lock()
	defer n.mu.Unlock()

	fundedAccounts := make([]FundedAccount, len(n.fundedAccounts))
	copy(fundedAccounts, n.fundedAccounts)
	return fundedAccounts
}

// GenTxs returns the genesis transactions.
func (n Network) GenTxs() []json.RawMessage {
	n.mu.Lock()
	defer n.mu.Unlock()

	genTxs := make([]json.RawMessage, len(n.genTxs))
	copy(genTxs, n.genTxs)
	return genTxs
}

// Denom returns the base chain denom. This is different
// for each network(i.e. mainnet, testnet, etc).
func (n Network) Denom() string {
	return n.denom
}

// FeeModel returns fee model configuration.
func (n Network) FeeModel() feemodeltypes.Model {
	return n.fee.FeeModel
}

// NetworkConfigByChainID returns predefined NetworkConfig for a ChainID.
func NetworkConfigByChainID(id constant.ChainID) (NetworkConfig, error) {
	nc, found := networkConfigs[id]
	if !found {
		return NetworkConfig{}, errors.Errorf("chainID %s not found", id)
	}

	return nc, nil
}

// NetworkByChainID returns predefined Network for a ChainID.
func NetworkByChainID(id constant.ChainID) (Network, error) {
	nc, err := NetworkConfigByChainID(id)
	if err != nil {
		return Network{}, err
	}

	return NewNetwork(nc), nil
}

// genesisByTemplate returns the genesis formatted by the input template.
func genesisByTemplate(n Network) ([]byte, error) {
	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
	}

	genesisBuf := new(bytes.Buffer)
	err := template.Must(template.New("genesis").Funcs(funcMap).Parse(genesisTemplate)).Execute(genesisBuf, struct {
		GenesisTimeUTC       string
		ChainID              constant.ChainID
		MetadataDisplayDenom string
		Denom                string
		FeeModelParams       feemodeltypes.ModelParams
		Gov                  GovConfig
		Staking              StakingConfig
		CustomParamsConfig   CustomParamsConfig
		AssetFTConfig        AssetFTConfig
		AssetNFTConfig       AssetNFTConfig
	}{
		GenesisTimeUTC:       n.genesisTime.UTC().Format(time.RFC3339),
		ChainID:              n.chainID,
		MetadataDisplayDenom: n.metadataDisplayDenom,
		Denom:                n.denom,
		FeeModelParams:       n.FeeModel().Params(),
		Gov:                  n.gov,
		Staking:              n.staking,
		CustomParamsConfig:   n.customParams,
		AssetFTConfig:        n.assetFT,
		AssetNFTConfig:       n.assetNFT,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to template genesis file")
	}

	return genesisBuf.Bytes(), nil
}
