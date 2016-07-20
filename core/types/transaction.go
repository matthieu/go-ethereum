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

package types

import (
	"container/heap"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"
	"sync/atomic"

	"github.com/matthieu/go-ethereum/common"
	"github.com/matthieu/go-ethereum/crypto"
	"github.com/matthieu/go-ethereum/logger"
	"github.com/matthieu/go-ethereum/logger/glog"
	"github.com/matthieu/go-ethereum/rlp"
)

var ErrInvalidSig = errors.New("invalid v, r, s values")

type Transaction struct {
	Dat TxData
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type TxData struct {
	AccountNonce    uint64
	Price, GasLimit *big.Int
	Recipient       *common.Address `rlp:"nil"` // nil means contract creation
	Amount          *big.Int
	Payload         []byte
	V               byte     // signature
	R, S            *big.Int // signature
}

func NewContractCreation(nonce uint64, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	return &Transaction{Dat: TxData{
		AccountNonce: nonce,
		Recipient:    nil,
		Amount:       new(big.Int).Set(amount),
		GasLimit:     new(big.Int).Set(gasLimit),
		Price:        new(big.Int).Set(gasPrice),
		Payload:      data,
		R:            new(big.Int),
		S:            new(big.Int),
	}}
}

func NewTransaction(nonce uint64, to common.Address, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := TxData{
		AccountNonce: nonce,
		Recipient:    &to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     new(big.Int),
		Price:        new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasLimit != nil {
		d.GasLimit.Set(gasLimit)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}
	return &Transaction{Dat: d}
}

func (tx *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.Dat)
}

func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.Dat)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}
	return err
}

func (tx *Transaction) Data() []byte       { return common.CopyBytes(tx.Dat.Payload) }
func (tx *Transaction) Gas() *big.Int      { return new(big.Int).Set(tx.Dat.GasLimit) }
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.Dat.Price) }
func (tx *Transaction) Value() *big.Int    { return new(big.Int).Set(tx.Dat.Amount) }
func (tx *Transaction) Nonce() uint64      { return tx.Dat.AccountNonce }

func (tx *Transaction) To() *common.Address {
	if tx.Dat.Recipient == nil {
		return nil
	} else {
		to := *tx.Dat.Recipient
		return &to
	}
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// SigHash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (tx *Transaction) SigHash() common.Hash {
	return rlpHash([]interface{}{
		tx.Dat.AccountNonce,
		tx.Dat.Price,
		tx.Dat.GasLimit,
		tx.Dat.Recipient,
		tx.Dat.Amount,
		tx.Dat.Payload,
	})
}

func (tx *Transaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.Dat)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// From returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// From Uses the homestead consensus rules to determine whether the signature is
// valid.
//
// From caches the address, allowing it to be used regardless of
// Frontier / Homestead. however, the first time called it runs
// signature validations, so we need two versions. This makes it
// easier to ensure backwards compatibility of things like package rpc
// where eth_getblockbynumber uses tx.From() and needs to work for
// both txs before and after the first homestead block. Signatures
// valid in homestead are a subset of valid ones in Frontier)
func (tx *Transaction) From() (common.Address, error) {
	return doFrom(tx, true)
}

// FromFrontier returns the address derived from the signature (V, R, S) using
// secp256k1 elliptic curve and an error if it failed deriving or upon an
// incorrect signature.
//
// FromFrantier uses the frontier consensus rules to determine whether the
// signature is valid.
//
// FromFrontier caches the address, allowing it to be used regardless of
// Frontier / Homestead. however, the first time called it runs
// signature validations, so we need two versions. This makes it
// easier to ensure backwards compatibility of things like package rpc
// where eth_getblockbynumber uses tx.From() and needs to work for
// both txs before and after the first homestead block. Signatures
// valid in homestead are a subset of valid ones in Frontier)
func (tx *Transaction) FromFrontier() (common.Address, error) {
	return doFrom(tx, false)
}

func doFrom(tx *Transaction, homestead bool) (common.Address, error) {
	if from := tx.from.Load(); from != nil {
		return from.(common.Address), nil
	}
	pubkey, err := tx.publicKey(homestead)
	if err != nil {
		return common.Address{}, err
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pubkey[1:])[12:])
	tx.from.Store(addr)
	return addr, nil
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.Dat.Price, tx.Dat.GasLimit)
	total.Add(total, tx.Dat.Amount)
	return total
}

func (tx *Transaction) SignatureValues() (v byte, r *big.Int, s *big.Int) {
	return tx.Dat.V, new(big.Int).Set(tx.Dat.R), new(big.Int).Set(tx.Dat.S)
}

func (tx *Transaction) publicKey(homestead bool) ([]byte, error) {
	if !crypto.ValidateSignatureValues(tx.Dat.V, tx.Dat.R, tx.Dat.S, homestead) {
		return nil, ErrInvalidSig
	}

	// encode the signature in uncompressed format
	r, s := tx.Dat.R.Bytes(), tx.Dat.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = tx.Dat.V - 27

	// recover the public key from the signature
	hash := tx.SigHash()
	pub, err := crypto.Ecrecover(hash[:], sig)
	if err != nil {
		glog.V(logger.Error).Infof("Could not get pubkey from signature: ", err)
		return nil, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return nil, errors.New("invalid public key")
	}
	return pub, nil
}

func (tx *Transaction) WithSignature(sig []byte) (*Transaction, error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	cpy := &Transaction{Dat: tx.Dat}
	cpy.Dat.R = new(big.Int).SetBytes(sig[:32])
	cpy.Dat.S = new(big.Int).SetBytes(sig[32:64])
	cpy.Dat.V = sig[64] + 27
	return cpy, nil
}

func (tx *Transaction) SignECDSA(prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := tx.SigHash()
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(sig)
}

func (tx *Transaction) String() string {
	var from, to string
	if f, err := tx.From(); err != nil {
		from = "[invalid sender]"
	} else {
		from = fmt.Sprintf("%x", f[:])
	}
	if tx.Dat.Recipient == nil {
		to = "[contract creation]"
	} else {
		to = fmt.Sprintf("%x", tx.Dat.Recipient[:])
	}
	enc, _ := rlp.EncodeToBytes(&tx.Dat)
	return fmt.Sprintf(`
	TX(%x)
	Contract: %v
	From:     %s
	To:       %s
	Nonce:    %v
	GasPrice: %v
	GasLimit  %v
	Value:    %v
	Data:     0x%x
	V:        0x%x
	R:        0x%x
	S:        0x%x
	Hex:      %x
`,
		tx.Hash(),
		len(tx.Dat.Recipient) == 0,
		from,
		to,
		tx.Dat.AccountNonce,
		tx.Dat.Price,
		tx.Dat.GasLimit,
		tx.Dat.Amount,
		tx.Dat.Payload,
		tx.Dat.V,
		tx.Dat.R,
		tx.Dat.S,
		enc,
	)
}

// Transaction slice type for basic sorting.
type Transactions []*Transaction

// Len returns the length of s
func (s Transactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp
func (s Transactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

// Returns a new set t which is the difference between a to b
func TxDifference(a, b Transactions) (keep Transactions) {
	keep = make(Transactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// TxByNonce implements the sort interface to allow sorting a list of transactions
// by their nonces. This is usually only useful for sorting transactions from a
// single account, otherwise a nonce comparison doesn't make much sense.
type TxByNonce Transactions

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].Dat.AccountNonce < s[j].Dat.AccountNonce }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// TxByPrice implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type TxByPrice Transactions

func (s TxByPrice) Len() int           { return len(s) }
func (s TxByPrice) Less(i, j int) bool { return s[i].Dat.Price.Cmp(s[j].Dat.Price) > 0 }
func (s TxByPrice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s *TxByPrice) Push(x interface{}) {
	*s = append(*s, x.(*Transaction))
}

func (s *TxByPrice) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

// SortByPriceAndNonce sorts the transactions by price in such a way that the
// nonce orderings within a single account are maintained.
//
// Note, this is not as trivial as it seems from the first look as there are three
// different criteria that need to be taken into account (price, nonce, account
// match), which cannot be done with any plain sorting method, as certain items
// cannot be compared without context.
//
// This method first sorts the separates the list of transactions into individual
// sender accounts and sorts them by nonce. After the account nonce ordering is
// satisfied, the results are merged back together by price, always comparing only
// the head transaction from each account. This is done via a heap to keep it fast.
func SortByPriceAndNonce(txs []*Transaction) {
	// Separate the transactions by account and sort by nonce
	byNonce := make(map[common.Address][]*Transaction)
	for _, tx := range txs {
		acc, _ := tx.From() // we only sort valid txs so this cannot fail
		byNonce[acc] = append(byNonce[acc], tx)
	}
	for _, accTxs := range byNonce {
		sort.Sort(TxByNonce(accTxs))
	}
	// Initialize a price based heap with the head transactions
	byPrice := make(TxByPrice, 0, len(byNonce))
	for acc, accTxs := range byNonce {
		byPrice = append(byPrice, accTxs[0])
		byNonce[acc] = accTxs[1:]
	}
	heap.Init(&byPrice)

	// Merge by replacing the best with the next from the same account
	txs = txs[:0]
	for len(byPrice) > 0 {
		// Retrieve the next best transaction by price
		best := heap.Pop(&byPrice).(*Transaction)

		// Push in its place the next transaction from the same account
		acc, _ := best.From() // we only sort valid txs so this cannot fail
		if accTxs, ok := byNonce[acc]; ok && len(accTxs) > 0 {
			heap.Push(&byPrice, accTxs[0])
			byNonce[acc] = accTxs[1:]
		}
		// Accumulate the best priced transaction
		txs = append(txs, best)
	}
}
