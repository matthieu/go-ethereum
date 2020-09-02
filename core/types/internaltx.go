package types

import (
	"math/big"

	"github.com/matthieu/go-ethereum/common"
)

type InternalTransaction struct {
	*Transaction

	Sender     *common.Address
	ParentHash common.Hash
	Depth      uint64
	Index      uint64
	Note       string
	Rejected   bool
}

type InternalTransactions []*InternalTransaction

func NewInternalTransaction(accountNonce uint64, price *big.Int,
	gasLimit uint64, sender common.Address,
	recipient common.Address, amount *big.Int, payload []byte,
	depth, index uint64, note string) *InternalTransaction {

	tx := NewTransaction(accountNonce, recipient, amount, gasLimit, price, payload)
	var h common.Hash
	return &InternalTransaction{tx, &sender, h, depth, index, note, false}
}

func (self *InternalTransaction) Reject() {
	self.Rejected = true
}

func (tx *InternalTransaction) Hash() common.Hash {
	rej := byte(0)
	if tx.Rejected {
		rej = byte(1)
	}
	data := []interface{}{
		tx.data.AccountNonce,
		tx.ParentHash,
		*tx.Sender,
		*tx.data.Recipient,
		tx.data.Amount,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Payload,
		tx.Note,
		tx.Depth,
		tx.Index,
		rej,
	}
	return rlpHash(data)
}
