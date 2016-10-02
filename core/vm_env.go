// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"

	"github.com/matthieu/go-ethereum/common"

	"github.com/matthieu/go-ethereum/core/state"
	"github.com/matthieu/go-ethereum/core/types"
	"github.com/matthieu/go-ethereum/core/vm"
)

type blockGetter interface {
	GetBlock(common.Hash) *types.Block
}

// GetHashFn returns a function for which the VM env can query block hashes through
// up to the limit defined by the Yellow Paper and uses the given block chain
// to query for information.
func GetHashFn(ref common.Hash, chain blockGetter) func(n uint64) common.Hash {
	return func(n uint64) common.Hash {
		for block := chain.GetBlock(ref); block != nil; block = chain.GetBlock(block.ParentHash()) {
			if block.NumberU64() == n {
				return block.Hash()
			}
		}

		return common.Hash{}
	}
}

type VMEnv struct {
	chainConfig *ChainConfig   // Chain configuration
	state       *state.StateDB // State to use for executing
	evm         *vm.EVM        // The Ethereum Virtual Machine
	depth       int            // Current execution depth
	msg         Message        // Message appliod

	header    *types.Header            // Header information
	chain     blockGetter              // Blockchain handle
	logs      []vm.StructLog           // Logs for the custom structured logger
	getHashFn func(uint64) common.Hash // getHashFn callback is used to retrieve block hashes

	internalTxs []*types.InternalTransaction
	hash        common.Hash // transaction/message hash originating this env
}

func NewEnv(state *state.StateDB, chainConfig *ChainConfig, chain blockGetter, msg Message, header *types.Header, hash common.Hash, cfg vm.Config) *VMEnv {
	env := &VMEnv{
		chainConfig: chainConfig,
		chain:       chain,
		state:       state,
		header:      header,
		msg:         msg,
		getHashFn:   GetHashFn(header.ParentHash, chain),
		hash:        hash,
	}

	// if no log collector is present set self as the collector
	if cfg.Logger.Collector == nil {
		cfg.Logger.Collector = env
	}

	env.evm = vm.New(env, cfg)
	return env
}

func (self *VMEnv) RuleSet() vm.RuleSet          { return self.chainConfig }
func (self *VMEnv) Vm() vm.Vm                    { return self.evm }
func (self *VMEnv) Origin() common.Address       { f, _ := self.msg.From(); return f }
func (self *VMEnv) OriginationHash() common.Hash { return self.hash }
func (self *VMEnv) BlockNumber() *big.Int        { return self.header.Number }
func (self *VMEnv) Coinbase() common.Address     { return self.header.Coinbase }
func (self *VMEnv) Time() *big.Int               { return self.header.Time }
func (self *VMEnv) Difficulty() *big.Int         { return self.header.Difficulty }
func (self *VMEnv) GasLimit() *big.Int           { return self.header.GasLimit }
func (self *VMEnv) Value() *big.Int              { return self.msg.Value() }
func (self *VMEnv) Db() vm.Database              { return self.state }
func (self *VMEnv) Depth() int                   { return self.depth }
func (self *VMEnv) SetDepth(i int)               { self.depth = i }
func (self *VMEnv) GetHash(n uint64) common.Hash {
	return self.getHashFn(n)
}

func (self *VMEnv) AddLog(log *vm.Log) {
	self.state.AddLog(log)
}
func (self *VMEnv) CanTransfer(from common.Address, balance *big.Int) bool {
	return self.state.GetBalance(from).Cmp(balance) >= 0
}

func (self *VMEnv) MakeSnapshot() vm.Database {
	return self.state.Copy()
}

func (self *VMEnv) SetSnapshot(copy vm.Database) {
	self.state.Set(copy.(*state.StateDB))
}

func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) {
	Transfer(from, to, amount)
}

func (self *VMEnv) Call(me vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return Call(self, me, addr, data, gas, price, value)
}
func (self *VMEnv) CallCode(me vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return CallCode(self, me, addr, data, gas, price, value)
}

func (self *VMEnv) DelegateCall(me vm.ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	return DelegateCall(self, me, addr, data, gas, price)
}

func (self *VMEnv) Create(me vm.ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return Create(self, me, data, gas, price, value)
}

func (self *VMEnv) Suicide(me vm.ContractRef, origin common.Address) error {
	return Suicide(self, me, origin)
}

func (self *VMEnv) StructLogs() []vm.StructLog {
	return self.logs
}

func (self *VMEnv) AddStructLog(log vm.StructLog) {
	self.logs = append(self.logs, log)
}

func (self *VMEnv) AddInternalTransaction(inttx interface{}) {
	if len(self.internalTxs) > 500 {
		return
	}
	internal := inttx.(*types.InternalTransaction)
	internal.ParentHash = self.hash
	internal.Index = uint64(len(self.internalTxs))
	internal.Depth = uint64(self.Depth())
	self.internalTxs = append(self.internalTxs, internal)
}

func (self *VMEnv) InternalTransactions() []*types.InternalTransaction {
	return self.internalTxs
}
