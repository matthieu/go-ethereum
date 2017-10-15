package core

import (
	"math/big"

	"github.com/matthieu/go-ethereum/common"
	"github.com/matthieu/go-ethereum/core/types"
	"github.com/matthieu/go-ethereum/core/vm"
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

func (self *InternalTxWatcher) InternalTransactions() types.InternalTransactions {
	return self.internals
}

func (self *InternalTxWatcher) RegisterCall(nonce uint64, gasPrice *big.Int, gas uint64, srcAddr, dstAddr common.Address, value *big.Int, data []byte) {
	self.internals = append(self.internals,
		types.NewInternalTransaction(nonce, gasPrice, toBigInt(gas),
			srcAddr, dstAddr, value, data, "call"))
}

func (self *InternalTxWatcher) RegisterStaticCall(nonce uint64, gasPrice *big.Int, gas uint64, srcAddr, dstAddr common.Address, data []byte) {

	self.internals = append(self.internals,
		types.NewInternalTransaction(nonce, gasPrice, toBigInt(gas),
			srcAddr, dstAddr, big.NewInt(0), data, "staticcall"))
}

func (self *InternalTxWatcher) RegisterCallCode(nonce uint64, gasPrice *big.Int, gas uint64, contractAddr common.Address, value *big.Int, data []byte) {
	self.internals = append(self.internals,
		types.NewInternalTransaction(nonce, gasPrice, toBigInt(gas),
			contractAddr, contractAddr, value, data, "call"))
}

func (self *InternalTxWatcher) RegisterCreate(nonce uint64, gasPrice *big.Int, gas uint64, srcAddr, newContractAddr common.Address, value *big.Int, data []byte) {
	self.internals = append(self.internals,
		types.NewInternalTransaction(nonce, gasPrice, toBigInt(gas),
			srcAddr, newContractAddr, value, data, "create"))
}

func (self *InternalTxWatcher) RegisterDelegateCall(nonce uint64, gasPrice *big.Int, gas uint64, callerAddr common.Address, value *big.Int, data []byte) {
	self.internals = append(self.internals,
		types.NewInternalTransaction(nonce, gasPrice, toBigInt(gas),
			callerAddr, callerAddr, value, data, "call"))
}

func (self *InternalTxWatcher) RegisterSuicide(nonce uint64, gasPrice *big.Int, gas uint64, contractAddr, creatorAddr common.Address, remainingValue *big.Int) {
	self.internals = append(self.internals,
		types.NewInternalTransaction(nonce, gasPrice, toBigInt(gas),
			contractAddr, creatorAddr, remainingValue,
			append([]byte{vm.SELFDESTRUCT}, creatorAddr[:]...), "suicide"))
}

func toBigInt(g uint64) *big.Int {
	return big.NewInt(0).SetUint64(g)
}
