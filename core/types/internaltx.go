package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type InternalTransaction struct {
	*Transaction

	Sender     *common.Address
	ParentHash common.Hash
	Depth      int
	Index      int
	Note       string
	Rejected   bool
}

type InternalTransactions []*InternalTransaction

func NewInternalTransaction(accountNonce uint64, price, gasLimit *big.Int, sender common.Address,
	recipient common.Address, amount *big.Int, payload []byte, note string) *InternalTransaction {

	tx := NewTransaction(accountNonce, recipient, amount, gasLimit, price, payload)
	var h common.Hash
	return &InternalTransaction{tx, &sender, h, -1, -1, note, false}
}

func (self *InternalTransaction) Reject() {
	self.Rejected = true
}

func (tx *InternalTransaction) Hash() common.Hash {
	return rlpHash([]interface{}{
		tx.data.AccountNonce,
		tx.ParentHash,
		tx.Sender,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Payload,
		tx.Note,
		tx.Depth,
		tx.Index,
		tx.Rejected,
	})
}
