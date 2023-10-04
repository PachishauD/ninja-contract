package chain

import (
	"encoding/hex"
	"sort"
	"testing"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/config"
	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core"
	"github.com/nspcc-dev/neo-go/pkg/core/storage"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/neotest"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	// MaxTraceableBlocks is the default MaxTraceableBlocks setting used for test chains.
	// We don't need a lot of traceable blocks for tests.
	MaxTraceableBlocks = 1000

	// TimePerBlock is the default TimePerBlock setting used for test chains (1s).
	// Usually blocks are created by tests bypassing this setting.
	TimePerBlock = time.Second
)

const singleValidatorWIF = "KxyjQ8eUa4FHt3Gvioyt1Wz29cTUrE4eTqX3yFSk1YFCsPL8uNsY"

// committeeWIFs is a list of unencrypted WIFs sorted by the public key.
var committeeWIFs = []string{
	"KzfPUYDC9n2yf4fK5ro4C8KMcdeXtFuEnStycbZgX3GomiUsvX6W",
	"KzgWE3u3EDp13XPXXuTKZxeJ3Gi8Bsm8f9ijY3ZsCKKRvZUo1Cdn",
	singleValidatorWIF,
	"L2oEXKRAAMiPEZukwR5ho2S6SMeQLhcK9mF71ZnF7GvT8dU4Kkgz",

	// Provide 2 committee extra members so that the committee address differs from
	// the validators one.
	"L1Tr1iq5oz1jaFaMXP21sHDkJYDDkuLtpvQ4wRf1cjKvJYvnvpAb",
	"Kz6XTUrExy78q8f4MjDHnwz8fYYyUE8iPXwPRAkHa3qN2JcHYm7e",
}

var (
	// committeeAcc is an account used to sign a tx as a committee.
	committeeAcc *wallet.Account

	// multiCommitteeAcc contains committee accounts used in a multi-node setup.
	multiCommitteeAcc []*wallet.Account

	// multiValidatorAcc contains validator accounts used in a multi-node setup.
	multiValidatorAcc []*wallet.Account

	// standByCommittee contains a list of committee public keys to use in config.
	standByCommittee []string
)

func init() {
	committeeAcc, _ = wallet.NewAccountFromWIF(singleValidatorWIF)
	pubs := keys.PublicKeys{committeeAcc.PublicKey()}
	err := committeeAcc.ConvertMultisig(1, pubs)
	if err != nil {
		panic(err)
	}

	mc := smartcontract.GetMajorityHonestNodeCount(len(committeeWIFs))
	mv := smartcontract.GetDefaultHonestNodeCount(4)
	accs := make([]*wallet.Account, len(committeeWIFs))
	pubs = make(keys.PublicKeys, len(accs))
	for i := range committeeWIFs {
		accs[i], _ = wallet.NewAccountFromWIF(committeeWIFs[i])
		pubs[i] = accs[i].PublicKey()
	}

	// Config entry must contain validators first in a specific order.
	standByCommittee = make([]string, len(pubs))
	standByCommittee[0] = hex.EncodeToString(pubs[2].Bytes())
	standByCommittee[1] = hex.EncodeToString(pubs[0].Bytes())
	standByCommittee[2] = hex.EncodeToString(pubs[3].Bytes())
	standByCommittee[3] = hex.EncodeToString(pubs[1].Bytes())
	standByCommittee[4] = hex.EncodeToString(pubs[4].Bytes())
	standByCommittee[5] = hex.EncodeToString(pubs[5].Bytes())

	multiValidatorAcc = make([]*wallet.Account, 4)
	sort.Sort(pubs[:4])

	sort.Slice(accs[:4], func(i, j int) bool {
		p1 := accs[i].PublicKey()
		p2 := accs[j].PublicKey()
		return p1.Cmp(p2) == -1
	})
	for i := range multiValidatorAcc {
		multiValidatorAcc[i] = wallet.NewAccountFromPrivateKey(accs[i].PrivateKey())
		err := multiValidatorAcc[i].ConvertMultisig(mv, pubs[:4])
		if err != nil {
			panic(err)
		}
	}

	multiCommitteeAcc = make([]*wallet.Account, len(committeeWIFs))
	sort.Sort(pubs)

	sort.Slice(accs, func(i, j int) bool {
		p1 := accs[i].PublicKey()
		p2 := accs[j].PublicKey()
		return p1.Cmp(p2) == -1
	})
	for i := range multiCommitteeAcc {
		multiCommitteeAcc[i] = wallet.NewAccountFromPrivateKey(accs[i].PrivateKey())
		err := multiCommitteeAcc[i].ConvertMultisig(mc, pubs)
		if err != nil {
			panic(err)
		}
	}
}

// NewSingle creates a new blockchain instance with a single validator and
// setups cleanup functions. The configuration used is with netmode.UnitTestNet
// magic and TimePerBlock/MaxTraceableBlocks options defined by constants in
// this package. MemoryStore is used as the backend storage, so all of the chain
// contents is always in RAM. The Signer returned is the validator (and the committee at
// the same time).
func NewSingle(t testing.TB) (*core.Blockchain, neotest.Signer) {
	return NewSingleWithCustomConfig(t, nil)
}

// NewSingleWithCustomConfig is similar to NewSingle, but allows to override the
// default configuration.
func NewSingleWithCustomConfig(t testing.TB, f func(*config.Blockchain)) (*core.Blockchain, neotest.Signer) {
	return NewSingleWithCustomConfigAndStore(t, f, nil, true)
}

// NewSingleWithCustomConfigAndStore is similar to NewSingleWithCustomConfig, but
// also allows to override backend Store being used. The last parameter controls if
// Run method is called on the Blockchain instance. If not, it is its caller's
// responsibility to do that before using the chain and
// to properly Close the chain when done.
func NewSingleWithCustomConfigAndStore(t testing.TB, f func(cfg *config.Blockchain), st storage.Store, run bool) (*core.Blockchain, neotest.Signer) {
	var cfg = config.Blockchain{
		ProtocolConfiguration: config.ProtocolConfiguration{
			Magic:              netmode.UnitTestNet,
			MaxTraceableBlocks: MaxTraceableBlocks,
			TimePerBlock:       TimePerBlock,
			StandbyCommittee:   []string{hex.EncodeToString(committeeAcc.PublicKey().Bytes())},
			ValidatorsCount:    1,
			VerifyTransactions: true,
		},
	}

	if f != nil {
		f(&cfg)
	}
	if st == nil {
		st = storage.NewMemoryStore()
	}
	log := zaptest.NewLogger(t)
	bc, err := core.NewBlockchain(st, cfg, log)
	require.NoError(t, err)
	if run {
		go bc.Run()
		t.Cleanup(bc.Close)
	}
	return bc, neotest.NewMultiSigner(committeeAcc)
}

// NewMulti creates a new blockchain instance with four validators and six
// committee members. Otherwise, it does not differ much from NewSingle. The
// second value returned contains the validators Signer, the third -- the committee one.
func NewMulti(t testing.TB) (*core.Blockchain, neotest.Signer, neotest.Signer) {
	return NewMultiWithCustomConfig(t, nil)
}

// NewMultiWithCustomConfig is similar to NewMulti, except it allows to override the
// default configuration.
func NewMultiWithCustomConfig(t testing.TB, f func(*config.Blockchain)) (*core.Blockchain, neotest.Signer, neotest.Signer) {
	return NewMultiWithCustomConfigAndStore(t, f, nil, true)
}

// NewMultiWithCustomConfigAndStore is similar to NewMultiWithCustomConfig, but
// also allows to override backend Store being used. The last parameter controls if
// Run method is called on the Blockchain instance. If not, it is its caller's
// responsibility to do that before using the chain and
// to properly Close the chain when done.
func NewMultiWithCustomConfigAndStore(t testing.TB, f func(*config.Blockchain), st storage.Store, run bool) (*core.Blockchain, neotest.Signer, neotest.Signer) {
	bc, validator, committee, err := NewMultiWithCustomConfigAndStoreNoCheck(t, f, st)
	require.NoError(t, err)
	if run {
		go bc.Run()
		t.Cleanup(bc.Close)
	}
	return bc, validator, committee
}

// NewMultiWithCustomConfigAndStoreNoCheck is similar to NewMultiWithCustomConfig,
// but do not perform Blockchain run and do not check Blockchain constructor error.
func NewMultiWithCustomConfigAndStoreNoCheck(t testing.TB, f func(*config.Blockchain), st storage.Store) (*core.Blockchain, neotest.Signer, neotest.Signer, error) {
	var cfg = config.Blockchain{
		ProtocolConfiguration: config.ProtocolConfiguration{
			Magic:              netmode.UnitTestNet,
			MaxTraceableBlocks: MaxTraceableBlocks,
			TimePerBlock:       TimePerBlock,
			StandbyCommittee:   standByCommittee,
			ValidatorsCount:    4,
			VerifyTransactions: true,
		},
	}
	if f != nil {
		f(&cfg)
	}
	if st == nil {
		st = storage.NewMemoryStore()
	}

	log := zaptest.NewLogger(t)
	bc, err := core.NewBlockchain(st, cfg, log)
	return bc, neotest.NewMultiSigner(multiValidatorAcc...), neotest.NewMultiSigner(multiCommitteeAcc...), err
}
