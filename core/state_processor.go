// Copyright 2015 The go-ethereum Authors
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

	"github.com/blockcypher/prpl/dain/util"

	"github.com/matthieu/go-ethereum/core/state"
	"github.com/matthieu/go-ethereum/core/types"
	"github.com/matthieu/go-ethereum/core/vm"
	"github.com/matthieu/go-ethereum/crypto"
	"github.com/matthieu/go-ethereum/logger"
	"github.com/matthieu/go-ethereum/logger/glog"
	"github.com/matthieu/go-ethereum/params"
)

var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

type TxExecReport struct {
	Transaction *types.Transaction
	Internals   types.InternalTransactions
	Receipt     *types.Receipt
	Errored     string
	GasUsed     *big.Int
	GasLeftover *big.Int
	GasRefund   *big.Int
}

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig
	bc     blockGetter
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc blockGetter) *StateProcessor {

	return &StateProcessor{
		config: config,
		bc:     bc,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, vm.Logs, *big.Int, []*TxExecReport, error) {
	var (
		receipts     types.Receipts
		totalUsedGas = big.NewInt(0)
		err          error
		header       = block.Header()
		allLogs      vm.Logs
		allReports   []*TxExecReport
		gp           = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		ApplyDAOHardFork(statedb)
	}
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		statedb.StartRecord(tx.Hash(), block.Hash(), i)
		receipt, logs, txg, report, err := ApplyTransaction(p.config, p.bc, gp, statedb, header, tx, totalUsedGas, cfg)
		util.LogNotice("TX", i, "used", txg)
		if err != nil {
			return nil, nil, totalUsedGas, nil, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, logs...)
		allReports = append(allReports, report)
	}
	AccumulateRewards(statedb, header, block.Uncles())

	return receipts, allLogs, totalUsedGas, allReports, err
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment.
//
// ApplyTransactions returns the generated receipts and vm logs during the
// execution of the state transition phase.
func ApplyTransaction(config *params.ChainConfig, bc blockGetter, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *big.Int, cfg vm.Config) (*types.Receipt, vm.Logs, *big.Int, *TxExecReport, error) {

	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	report := &TxExecReport{Transaction: tx}
	env := NewEnv(statedb, config, bc, msg, header, tx.Hash(), cfg)
	_, gas, err := ApplyMessage(env, msg, gp, report)
	util.LogNotice("Gas from ApplyMessage return:", gas, "from report:", report.GasUsed)

	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Update the state with pending changes
	usedGas.Add(usedGas, report.GasUsed)
	receipt := types.NewReceipt(statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes(), usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = new(big.Int).Set(report.GasUsed)
	if MessageCreatesContract(msg) {
		receipt.ContractAddress = crypto.CreateAddress(msg.From(), tx.Nonce())
	}

	logs := statedb.GetLogs(tx.Hash())
	receipt.Logs = logs
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	glog.V(logger.Debug).Infoln(receipt)
	report.Internals = env.InternalTransactions()
	report.Receipt = receipt

	return receipt, logs, gas, report, err
}

// AccumulateRewards credits the coinbase of the given block with the
// mining reward. The total reward consists of the static block reward
// and rewards for included uncles. The coinbase of each uncle block is
// also rewarded.
func AccumulateRewards(statedb *state.StateDB, header *types.Header, uncles []*types.Header) {
	reward := new(big.Int).Set(BlockReward)
	r := new(big.Int)
	for _, uncle := range uncles {
		r.Add(uncle.Number, big8)
		r.Sub(r, header.Number)
		r.Mul(r, BlockReward)
		r.Div(r, big8)
		statedb.AddBalance(uncle.Coinbase, r)

		r.Div(BlockReward, big32)
		reward.Add(reward, r)
	}
	statedb.AddBalance(header.Coinbase, reward)
}
