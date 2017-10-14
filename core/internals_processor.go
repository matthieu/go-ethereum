package core

import (
	"math/big"

	"github.com/matthieu/go-ethereum/common"
	"github.com/matthieu/go-ethereum/core/types"
)

// Implementation of evm.InternalTxListener

type InternalTxWatcher struct {
	internals types.InternalTransactions
}

func NewInternalTxWatcher() *InternalTxWatcher {
	return &InternalTxWatcher{
		internals: make(types.InternalTransactions, 0),
	}
}

func (self *IntenalTxWatcher) InternalTransactions() types.InternalTransactions {
	return self.internals
}

func (self *InternalTxWatcher) RegisterCall(nonce uint64, gasPrice, gas *big.Int, srcAddr, dstAddr common.Address, value *big.Int, data []byte) {
}

func (self *InternalTxWatcher) RegisterCallCode(nonce uint64, gasPrice, gas *big.Int, contractAddr common.Address, value *big.Int, data []byte) {
}

func (self *InternalTxWatcher) RegisterCreate(nonce uint64, gasPrice, gas *big.Int, srcAddr, newContractAddr common.Address, value *big.Int, data []byte) {
}

func (self *InternalTxWatcher) RegisterDelegateCall(nonce uint64, gasPrice, gas *big.Int, callerAddr common.Address, value *big.Int, data []byte) {
}

func (self *InternalTxWatcher) RegisterSuicide(nonce uint64, gasPrice, gas *big.Int, contractAddr, creatorAddr common.Address, remainingValue *big.Int) {
}
