package MuSeal

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"sync"
	"io"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/consensus"
	"math/big"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/crypto/sha3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"encoding/binary"
	"math/rand"
	// "fmt"
	// "strconv"
	// "io/ioutil"
	// "os"
	// // "path/filepath"
	// "encoding/json"
	// "github.com/Knetic/govaluate"
	"github.com/pkg/errors"
)

func New(config *params.MuSealConfig, db ethdb.Database) *MuSeal {
	conf := *config
	// if conf.Epoch == 0 {
	// 	conf.Epoch = epochLength
	// }
	// Allocate the snapshot caches and create the engine

	return &MuSeal{
		config:     &conf,
		db:         db,
	}
}

type MuSeal struct {
	config *params.MuSealConfig
	db     ethdb.Database    
	lock   sync.RWMutex
}

func (MuSeal *MuSeal) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

func (MuSeal *MuSeal) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	log.Info("will verfiyHeader")
	return nil
}

func (MuSeal *MuSeal) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error){
	log.Info("will verfiyHeaders")
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for _, header := range headers {
			err := MuSeal.VerifyHeader(chain, header)

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

func (MuSeal *MuSeal) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	log.Info("will verfiy uncles")
	return nil
}

func (MuSeal *MuSeal)  VerifySeal(chain consensus.ChainHeaderReader, header *types.Header) error{
	log.Info("will verfiy VerifySeal")
	return nil
}

func (MuSeal *MuSeal) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error{
	log.Info("will prepare the block")
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Difficulty = MuSeal.CalcDifficulty(chain, header.Time, parent)
	return nil
}

func (MuSeal *MuSeal) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return calcDifficultyHomestead(time, parent)
}

var (
	expDiffPeriod = big.NewInt(100000)
	big1          = big.NewInt(1)
	big2          = big.NewInt(2)
	big9          = big.NewInt(9)
	big10         = big.NewInt(10)
	bigMinus99    = big.NewInt(-99)
	big2999999    = big.NewInt(2999999)
)

// calcDifficultyHomestead is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses the Homestead rules.
func calcDifficultyHomestead(time uint64, parent *types.Header) *big.Int {
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)

	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).SetUint64(parent.Time)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// 1 - (block_timestamp - parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big10)
	x.Sub(big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, params.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}
	// for the exponential factor
	periodCount := new(big.Int).Add(parent.Number, big1)
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, commonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(big1) > 0 {
		y.Sub(periodCount, big2)
		y.Exp(big2, y, nil)
		x.Add(x, y)
	}
	return x
}

// Finalize runs any post-transaction state modifications (e.g. block rewards)
// and assembles the final block.
// Note: The block header and state database might be updated to reflect any
// consensus rules that happen at finalization (e.g. block rewards).
// func (MuSeal *MuSeal) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
// 	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error){
// 	log.Info("will Finalize the block")
// 	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
// 	b := types.NewBlock(header, txs, uncles, receipts, trie.NewStackTrie(nil))

// 	return b, nil
// }

func (c *MuSeal) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, withdrawals []*types.Withdrawal) {
	log.Info("will Finalize the block")
	// header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	// b := types.NewBlock(header, txs, uncles, receipts, trie.NewStackTrie(nil))

	// return b, nil
}

func (c *MuSeal) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt, withdrawals []*types.Withdrawal) (*types.Block, error) {
	if len(withdrawals) > 0 {
		return nil, errors.New("MuSeal does not support withdrawals")
	}
	// Finalize block
	c.Finalize(chain, header, state, txs, uncles, nil)

	// Assign the final state root to header.
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// Assemble and return the final block for sealing.
	return types.NewBlock(header, txs, nil, receipts, trie.NewStackTrie(nil)), nil
}

func encodeSigHeader(w io.Writer, header *types.Header) {
	enc := []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-crypto.SignatureLength], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	if header.WithdrawalsHash != nil {
		panic("unexpected withdrawal hash value in clique")
	}
	if header.ExcessBlobGas != nil {
		panic("unexpected excess blob gas value in clique")
	}
	if header.BlobGasUsed != nil {
		panic("unexpected blob gas used value in clique")
	}
	if header.ParentBeaconRoot != nil {
		panic("unexpected parent beacon root value in clique")
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}


func SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	// encodeSigHeader(hasher, header)
	hasher.(crypto.KeccakState).Read(hash[:])
	return hash
}

func (c *MuSeal) SealHash(header *types.Header) common.Hash {
	return SealHash(header)
}

// Seal generates a new block for the given input block with the local miner's
// seal place on top.
func (MuSeal *MuSeal) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	log.Info("will Seal the block")
	//time.Sleep(15 * time.Second)
	header := block.Header()
	/*
	runes := []rune(header.ParentHash.String())
	index_in_hash := string(runes[0:3])
	index_in_decimal, _ := strconv.ParseInt(index_in_hash , 0, 64)
	index_in_decimal = index_in_decimal % 10
	*/
	// header.Nonce, header.MixDigest = getRequiredHeader(result_in_float)
	go func() {
		select {
		case results <- block.WithSeal(header):
		default:
			log.Warn("Sealing result is not read by miner", "sealhash", SealHash(header))
		}
	}()
	return nil
}

func getNonce(result float64) (types.BlockNonce) {
	var i uint64 = uint64(result)
	var n types.BlockNonce

	binary.BigEndian.PutUint64(n[:], i)
	return n
}

func rangeIn(low, hi int) int {
	return low + rand.Intn(hi-low)
}

// APIs returns the RPC APIs this consensus engine provides.
func (MuSeal *MuSeal) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "MuSeal",
		Version:   "1.0",
		Service:   &API{chain: chain, MuSeal: MuSeal},
		Public:    false,
	}}
}

// Close implements consensus.Engine. It's a noop for clique as there are no background threads.
func (c *MuSeal) Close() error {
	return nil
}