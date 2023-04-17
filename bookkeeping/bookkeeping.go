package bookkeeping

import (
	"context"
	"errors"
	"time"

	"github.com/bartossh/The-Accountant/block"
	"github.com/bartossh/The-Accountant/transaction"
)

const (
	minDifficulty = 1
	maxDifficulty = 255

	minBlockWriteTimestamp = time.Second
	maxBlockWriteTimestamp = time.Hour * 4

	minBlockTransactionsSize = 1
	maxBlockTransactionsSize = 60000
)

var (
	ErrTrxExistsInTheLadger            = errors.New("transaction is already in the ledger")
	ErrTrxExistsInTheBlockchain        = errors.New("transaction is already in the blockchain")
	ErrAddressNotExists                = errors.New("address does not exist in the addresses repository")
	ErrBlockTxsCorrupted               = errors.New("all transaction failed, block corrupted")
	ErrDifficultyNotInRange            = errors.New("invalid difficulty, difficulty can by in range [1 : 255]")
	ErrBlockWriteTimestampNoInRange    = errors.New("block write timestamp is not in range of [one second : four hours]")
	ErrBlockTransactionsSizeNotInRange = errors.New("block transactions size is not in range of [1 : 60000]")
)

type TrxWriteMover interface {
	WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error
	MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error
}

type BlockReader interface {
	LastBlockHashIndex(ctx context.Context) ([32]byte, uint64, error)
}

type BlockWriter interface {
	WriteBlock(ctx context.Context, block block.Block) error
}

type BlockReadWriter interface {
	BlockReader
	BlockWriter
}

type AddressChecker interface {
	CheckAddressExists(ctx context.Context, address string) (bool, error)
}

type SignatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type BlockFinder interface {
	WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error
	FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
}

type Config struct {
	Difficulty            uint64        `json:"difficulty"              bson:"difficulty"              yaml:"difficulty"`
	BlockWriteTimestamp   time.Duration `json:"block_write_timestamp"   bson:"block_write_timestamp"   yaml:"block_write_timestamp"`
	BlockTransactionsSize int           `json:"block_transactions_size" bson:"block_transactions_size" yaml:"block_transactions_size"`
}

func (c Config) Validate() error {
	if c.Difficulty < minDifficulty || c.Difficulty > maxDifficulty {
		return ErrDifficultyNotInRange
	}

	if c.BlockWriteTimestamp < minBlockWriteTimestamp || c.BlockWriteTimestamp > maxBlockWriteTimestamp {
		return ErrBlockWriteTimestampNoInRange
	}

	if c.BlockTransactionsSize < minBlockTransactionsSize || c.BlockTransactionsSize > maxBlockTransactionsSize {
		return ErrBlockTransactionsSizeNotInRange
	}

	return nil
}

// Ledger is a collection of ledger functionality to perform bookkeeping.
type Ledger struct {
	config Config
	hashC  chan [32]byte
	hashes [][32]byte
	tx     TrxWriteMover
	bc     BlockReadWriter
	ac     AddressChecker
	vr     SignatureVerifier
	tf     BlockFinder
}

// NewLedger creates new Ledger if config is valid or returns error otherwise.
func NewLedger(
	config Config,
	bc BlockReadWriter,
	tx TrxWriteMover,
	ac AddressChecker,
	vr SignatureVerifier,
	tf BlockFinder,
) (*Ledger, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Ledger{
		config: config,
		hashC:  make(chan [32]byte, config.BlockTransactionsSize),
		tx:     tx,
		bc:     bc,
		ac:     ac,
		vr:     vr,
		tf:     tf,
	}, nil
}

// Run runs the Ladger engine that writes blocks to the blockchain repository.
// Run starts a goroutine and can be stopped by cancelling the context.
func (l *Ledger) Run(ctx context.Context) {
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				if len(l.hashes) > 0 {
					l.forge(ctx)
				}
				break
			case h := <-l.hashC:
				l.hashes = append(l.hashes, h)
				if len(l.hashes) == l.config.BlockTransactionsSize {
					l.forge(ctx)
				}
			case <-time.After(l.config.BlockWriteTimestamp):
				if len(l.hashes) > 0 {
					l.forge(ctx)
				}
			}
		}
	}(ctx)
}

// WriteCandidateTransaction validates and writes a transaction to the repository.
// Transaction is not yet a part of the blockchain.
func (l *Ledger) WriteCandidateTransaction(ctx context.Context, tx *transaction.Transaction) error {
	if err := l.validateTx(ctx, tx); err != nil {
		return err
	}
	if err := l.tx.WriteTemporaryTransaction(ctx, tx); err != nil {
		return err
	}

	l.hashC <- tx.Hash

	return nil
}

func (l *Ledger) forge(ctx context.Context) {
	defer l.cleanHashes()
	blcHash, err := l.saveBlock(ctx)
	if err != nil {
		// TODO: log error and all the hashes of unsigned transactions
		return
	}

	if err := l.tf.WriteTransactionsInBlock(ctx, blcHash, l.hashes); err != nil {
		// TODO: log error with writing block hash and trxs hashes in to the search
	}

	if err := l.tx.MoveTransactionsFromTemporaryToPermanent(ctx, l.hashes); err != nil {
		// TODO: log error and all the hashes. This error will cause inconsistency. Decide what to do with that problem.
	}
}

func (l *Ledger) cleanHashes() {
	l.hashes = make([][32]byte, 0, l.config.BlockTransactionsSize)
}

func (l *Ledger) saveBlock(ctx context.Context) ([32]byte, error) {
	h, idx, err := l.bc.LastBlockHashIndex(ctx)
	if err != nil {
		return [32]byte{}, err
	}

	nb := block.NewBlock(l.config.Difficulty, idx, h, l.hashes)

	if err := l.bc.WriteBlock(ctx, nb); err != nil {
		return [32]byte{}, err
	}

	return nb.Hash, nil
}

func (l *Ledger) validateTx(ctx context.Context, tx *transaction.Transaction) error {
	exists, err := l.ac.CheckAddressExists(ctx, tx.IssuerAddress)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAddressNotExists
	}

	exists, err = l.ac.CheckAddressExists(ctx, tx.ReceiverAddress)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAddressNotExists
	}

	if err := l.vr.Verify(tx.Data, tx.IssuerSignature, tx.Hash, tx.IssuerAddress); err != nil {
		return err
	}

	if err := l.vr.Verify(tx.Data, tx.IssuerSignature, tx.Hash, tx.IssuerAddress); err != nil {
		return err
	}

	return nil
}