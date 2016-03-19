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

	"github.com/ethereum/go-ethereum/common"

	"github.com/matthieu/go-ethereum/core/state"
	"github.com/matthieu/go-ethereum/core/types"
	"github.com/matthieu/go-ethereum/core/vm"
)

type blockGetter interface {
	GetBlock(common.Hash) *types.Block
}

type VMEnv struct {
	state  *state.StateDB
	header *types.Header
	msg    Message
	depth  int
	chain  blockGetter
	typ    vm.Type
	// structured logging
	logs []vm.StructLog

	internalTxs []*types.InternalTransaction
	hash        common.Hash // transaction/message hash originating this env
}

func NewEnv(state *state.StateDB, chain blockGetter, msg Message, header *types.Header, hash common.Hash) *VMEnv {
	return &VMEnv{
		chain:  chain,
		state:  state,
		header: header,
		msg:    msg,
		typ:    vm.StdVmTy,
		hash:   hash,
	}
}

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
func (self *VMEnv) VmType() vm.Type              { return self.typ }
func (self *VMEnv) SetVmType(t vm.Type)          { self.typ = t }
func (self *VMEnv) GetHash(n uint64) common.Hash {
	for block := self.chain.GetBlock(self.header.ParentHash); block != nil; block = self.chain.GetBlock(block.ParentHash()) {
		if block.NumberU64() == n {
			return block.Hash()
		}
	}

	return common.Hash{}
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

func (self *VMEnv) StructLogs() []vm.StructLog {
	return self.logs
}

func (self *VMEnv) AddStructLog(log vm.StructLog) {
	self.logs = append(self.logs, log)
}

func (self *VMEnv) AddInternalTransaction(inttx interface{}) {
	internal := inttx.(*types.InternalTransaction)
	internal.ParentHash = self.hash
	internal.Index = len(self.internalTxs)
	internal.Depth = self.Depth()
	self.internalTxs = append(self.internalTxs, internal)
}

func (self *VMEnv) InternalTransactions() []*types.InternalTransaction {
	return self.internalTxs
}
