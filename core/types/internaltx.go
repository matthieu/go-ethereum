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
}

type InternalTransactions []*InternalTransaction

func NewInternalTransaction(accountNonce uint64, price, gasLimit *big.Int, sender common.Address,
	recipient common.Address, amount *big.Int, payload []byte, parentHash common.Hash,
	depth int) *InternalTransaction {

	tx := NewTransaction(accountNonce, recipient, amount, gasLimit, price, payload)
	return &InternalTransaction{tx, &sender, parentHash, depth}
}
