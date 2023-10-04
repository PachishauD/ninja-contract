package consensus

import (
	"errors"

	"github.com/nspcc-dev/dbft/block"
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	coreb "github.com/nspcc-dev/neo-go/pkg/core/block"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

// neoBlock is a wrapper of a core.Block which implements
// methods necessary for dBFT library.
type neoBlock struct {
	coreb.Block

	network   netmode.Magic
	signature []byte
}

var _ block.Block = (*neoBlock)(nil)

// Sign implements the block.Block interface.
func (n *neoBlock) Sign(key crypto.PrivateKey) error {
	k := key.(*privateKey)
	sig := k.PrivateKey.SignHashable(uint32(n.network), &n.Block)
	n.signature = sig
	return nil
}

// Verify implements the block.Block interface.
func (n *neoBlock) Verify(key crypto.PublicKey, sign []byte) error {
	k := key.(*publicKey)
	if k.PublicKey.VerifyHashable(sign, uint32(n.network), &n.Block) {
		return nil
	}
	return errors.New("verification failed")
}

// Transactions implements the block.Block interface.
func (n *neoBlock) Transactions() []block.Transaction {
	txes := make([]block.Transaction, len(n.Block.Transactions))
	for i, tx := range n.Block.Transactions {
		txes[i] = tx
	}

	return txes
}

// SetTransactions implements the block.Block interface.
func (n *neoBlock) SetTransactions(txes []block.Transaction) {
	n.Block.Transactions = make([]*transaction.Transaction, len(txes))
	for i, tx := range txes {
		n.Block.Transactions[i] = tx.(*transaction.Transaction)
	}
}

// Version implements the block.Block interface.
func (n *neoBlock) Version() uint32 { return n.Block.Version }

// PrevHash implements the block.Block interface.
func (n *neoBlock) PrevHash() util.Uint256 { return n.Block.PrevHash }

// MerkleRoot implements the block.Block interface.
func (n *neoBlock) MerkleRoot() util.Uint256 { return n.Block.MerkleRoot }

// Timestamp implements the block.Block interface.
func (n *neoBlock) Timestamp() uint64 { return n.Block.Timestamp * nsInMs }

// Index implements the block.Block interface.
func (n *neoBlock) Index() uint32 { return n.Block.Index }

// ConsensusData implements the block.Block interface.
func (n *neoBlock) ConsensusData() uint64 { return n.Block.Nonce }

// NextConsensus implements the block.Block interface.
func (n *neoBlock) NextConsensus() util.Uint160 { return n.Block.NextConsensus }

// Signature implements the block.Block interface.
func (n *neoBlock) Signature() []byte { return n.signature }
