package accountant

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/heimdalr/dag"

	"github.com/bartossh/Computantis/src/logger"
	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/storage"
	"github.com/bartossh/Computantis/src/transaction"
)

const (
	initialThroughput = 50
	lastVertexHashes  = 100
)

const repiterTick = time.Second * 2

var (
	ErrGenesisRejected                       = errors.New("genesis vertex has been rejected")
	ErrBalanceCaclulationUnexpectedFailure   = errors.New("balance calculation unexpected failure")
	ErrBalanceUnavailable                    = errors.New("balance unavailable")
	ErrLeafBallanceCalculationProcessStopped = errors.New("wallet balance calculation process stopped")
	ErrLeafValidationProcessStopped          = errors.New("leaf validation process stopped")
	ErrNewLeafRejected                       = errors.New("new leaf rejected")
	ErrLeafRejected                          = errors.New("leaf rejected")
	ErrDagIsLoaded                           = errors.New("dag is already loaded")
	ErrDagIsNotLoaded                        = errors.New("dag is not loaded")
	ErrLeafAlreadyExists                     = errors.New("leaf already exists")
	ErrIssuerAddressBalanceNotFound          = errors.New("issuer address balance not found")
	ErrReceiverAddressBalanceNotFound        = errors.New("receiver address balance not found")
	ErrDoubleSpendingOrInsufficinetFounds    = errors.New("double spending or insufficient founds")
	ErrCannotTransferFoundsViaOwnedNode      = errors.New("issuer cannot transfer founds via owned node")
	ErrCannotTransferFoundsFromGenesisWallet = errors.New("issuer cannot be the genesis node")
	ErrVertexHashNotFound                    = errors.New("vertex hash not found")
	ErrVertexAlreadyExists                   = errors.New("vertex already exists")
	ErrTrxInVertexAlreadyExists              = errors.New("transaction in vertex already exists")
	ErrTrxToVertexNotFound                   = errors.New("trx mapping to vertex do not found, transaction doesn't exist")
	ErrUnexpected                            = errors.New("unexpected failure")
	ErrTransferringFoundsFailure             = errors.New("transferring spice failure")
	ErrEntityNotFound                        = errors.New("entity not found")
)

type signatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

// Signer signs the given message and has a public address.
type Signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

// AccountingBook is an entity that represents the accounting process of all received transactions.
type AccountingBook struct {
	repiter              *buffer
	verifier             signatureVerifier
	signer               Signer
	log                  logger.Logger
	dag                  *dag.DAG
	trustedNodesDB       *badger.DB
	trxsToVertxDB        *badger.DB
	verticesDB           *badger.DB
	genesisPublicAddress string
	mux                  sync.RWMutex
	gennessisHash        [32]byte
	weight               atomic.Uint64
	throughput           atomic.Uint64
	dagLoaded            bool
}

// New creates new AccountingBook.
// New AccountingBook will start internally the garbage collection loop, to stop it from running cancel the context.
func NewAccountingBook(ctx context.Context, cfg Config, verifier signatureVerifier, signer Signer, l logger.Logger) (*AccountingBook, error) {
	repi, err := newReplierBuffer(ctx, repiterTick)
	if err != nil {
		return nil, err
	}

	trustedNodesDB, err := storage.CreateBadgerDB(ctx, cfg.TrustedNodesDBPath, l, true)
	if err != nil {
		return nil, err
	}
	trxsToVertxDB, err := storage.CreateBadgerDB(ctx, cfg.TraxsToVerticesMapDBPath, l, true)
	if err != nil {
		return nil, err
	}
	verticesDB, err := storage.CreateBadgerDB(ctx, cfg.VerticesDBPath, l, true)
	if err != nil {
		return nil, err
	}

	ab := &AccountingBook{
		repiter:        repi,
		verifier:       verifier,
		signer:         signer,
		dag:            dag.NewDAG(),
		trustedNodesDB: trustedNodesDB,
		trxsToVertxDB:  trxsToVertxDB,
		verticesDB:     verticesDB,
		mux:            sync.RWMutex{},
		log:            l,
		weight:         atomic.Uint64{},
		throughput:     atomic.Uint64{},
	}

	go ab.runLeafSubscriber(ctx)

	return ab, nil
}

func (ab *AccountingBook) runLeafSubscriber(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-ab.repiter.subscribe():
			if v == nil {
				continue
			}
			ab.AddLeaf(ctx, v)
		}
	}
}

func (ab *AccountingBook) validateLeaf(ctx context.Context, leaf *Vertex) error {
	if leaf == nil {
		return errors.Join(ErrUnexpected, errors.New("leaf to validate is nil"))
	}
	if !ab.isValidWeight(leaf.Weight) {
		return errors.Join(
			ErrLeafRejected,
			fmt.Errorf("leaf doesn't meet condition of minimal weight, throughput: %v current: %v, received: %v", ab.throughput.Load(), ab.weight.Load(), leaf.Weight),
		)
	}

	if err := leaf.verify(ab.verifier); err != nil {
		return errors.Join(ErrLeafRejected, err)
	}
	isRoot, err := ab.dag.IsRoot(string(leaf.Hash[:]))
	if err != nil {
		return errors.Join(ErrUnexpected, err)
	}
	if isRoot {
		return nil
	}
	trusted, err := ab.checkIsTrustedNode(leaf.SignerPublicAddress)
	if err != nil {
		return errors.Join(ErrUnexpected, err)
	}
	if !leaf.Transaction.IsSpiceTransfer() || trusted {
		_, err := ab.dag.GetVertex(string(leaf.RightParentHash[:]))
		if err != nil {
			return errors.Join(ErrLeafRejected, err)
		}

		_, err = ab.dag.GetVertex(string(leaf.LeftParentHash[:]))
		if err != nil {
			return errors.Join(ErrLeafRejected, err)
		}
		return nil
	}

	visited := make(map[string]struct{})
	spiceOut := spice.New(0, 0)
	spiceIn := spice.New(0, 0)
	if err := pourFounds(leaf.Transaction.IssuerAddress, *leaf, &spiceIn, &spiceOut); err != nil {
		return err
	}
	vertices, signal, _ := ab.dag.AncestorsWalker(string(leaf.Hash[:]))
	for ancestorID := range vertices {
		select {
		case <-ctx.Done():
			signal <- true
			return ErrLeafValidationProcessStopped
		default:
		}
		if _, ok := visited[ancestorID]; ok {
			continue
		}
		visited[ancestorID] = struct{}{}

		item, err := ab.dag.GetVertex(ancestorID)
		if err != nil {
			signal <- true
			return errors.Join(ErrUnexpected, err)
		}
		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				return ErrUnexpected
			}
			if vrx.Hash == leaf.LeftParentHash {
				if err := vrx.verify(ab.verifier); err != nil {
					signal <- true
					return errors.Join(ErrLeafRejected, err)
				}
			}
			if vrx.Hash == leaf.RightParentHash {
				if err := vrx.verify(ab.verifier); err != nil {
					signal <- true
					return errors.Join(ErrLeafRejected, err)
				}
			}
			if err := pourFounds(leaf.Transaction.IssuerAddress, *vrx, &spiceIn, &spiceOut); err != nil {
				return errors.Join(ErrTransferringFoundsFailure, err)
			}

		default:
			signal <- true
			return ErrUnexpected
		}
	}

	err = checkHasSufficientFounds(&spiceIn, &spiceOut)
	if err != nil {

		ab.log.Info(
			fmt.Sprintf(
				"No sufficient founds [ in: %s ] [ out: %s ]\n",
				spiceIn, spiceOut,
			),
		)
		return errors.Join(ErrTransferringFoundsFailure, err)
	}
	return nil
}

func (ab *AccountingBook) checkIsTrustedNode(trustedNodePublicAddress string) (bool, error) {
	var ok bool
	err := ab.trustedNodesDB.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(trustedNodePublicAddress))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}
			return err
		}
		ok = true
		return nil
	})
	return ok, err
}

func (ab *AccountingBook) checkTrxInVertexExists(trxHash []byte) (bool, error) {
	err := ab.trxsToVertxDB.View(func(txn *badger.Txn) error {
		_, err := txn.Get(trxHash)
		if err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		return true, nil
	}
	switch err {
	case badger.ErrKeyNotFound:
		return false, nil
	default:
		ab.log.Error(fmt.Sprintf("transaction to vertex mapping for existing trx lookup failed, %s", err))
		return false, ErrUnexpected
	}
}

func (ab *AccountingBook) saveTrxInVertex(trxHash, vrxHash []byte) error {
	return ab.trxsToVertxDB.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(trxHash); err == nil {
			return ErrTrxInVertexAlreadyExists
		}
		return txn.SetEntry(badger.NewEntry(trxHash, vrxHash))
	})
}

func (ab *AccountingBook) removeTrxInVertex(trxHash []byte) error {
	return ab.trxsToVertxDB.Update(func(txn *badger.Txn) error {
		return txn.Delete(trxHash)
	})
}

func (ab *AccountingBook) readTrxVertex(trxHash []byte) (Vertex, error) {
	var vrxHash []byte
	err := ab.trxsToVertxDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(trxHash)
		if err != nil {
			return err
		}
		item.Value(func(v []byte) error {
			vrxHash = v
			return nil
		})
		return nil
	})
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return Vertex{}, ErrTrxToVertexNotFound
		default:
			ab.log.Error(fmt.Sprintf("transaction to vertex mapping failed when looking for transaction hash, %s", err))
			return Vertex{}, ErrUnexpected
		}
	}
	return ab.readVertex(vrxHash)
}

func (ab *AccountingBook) readVertex(vrxHash []byte) (Vertex, error) {
	vrx, err := ab.readVertexFromDAG(vrxHash)
	if err == nil {
		return vrx, nil
	}
	if !errors.Is(err, ErrVertexHashNotFound) {
		return Vertex{}, err
	}
	return ab.readVertexFromStorage(vrxHash)
}

func (ab *AccountingBook) checkVertexExists(vrxHash []byte) (bool, error) {
	_, err := ab.dag.GetVertex(string(vrxHash))
	if err == nil {
		return true, nil
	}
	err = ab.verticesDB.View(func(txn *badger.Txn) error {
		_, err := txn.Get(vrxHash)
		return err
	})
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return false, nil
		default:
			ab.log.Error(fmt.Sprintf("transaction to vertex mapping failed when looking for transaction hash, %s", err))
			return false, ErrUnexpected
		}
	}
	return true, nil
}

func (ab *AccountingBook) readVertexFromDAG(vrxHash []byte) (Vertex, error) {
	item, err := ab.dag.GetVertex(string(vrxHash))
	if err == nil {
		switch v := item.(type) {
		case *Vertex:
			return *v, nil
		default:
			return Vertex{}, ErrUnexpected
		}
	}
	return Vertex{}, ErrVertexHashNotFound
}

func (ab *AccountingBook) readVertexFromStorage(vrxHash []byte) (Vertex, error) {
	var vrx Vertex
	err := ab.verticesDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(vrxHash)
		if err != nil {
			return err
		}
		item.Value(func(v []byte) error {
			vrx, err = decodeVertex(v)
			return err
		})
		return nil
	})
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return vrx, ErrVertexHashNotFound
		default:
			ab.log.Error(fmt.Sprintf("transaction to vertex mapping failed when looking for vertex hash, %s", err))
			return vrx, ErrUnexpected
		}
	}

	return vrx, nil
}

func (ab *AccountingBook) saveVertexToStorage(vrx *Vertex) error {
	buf, err := vrx.Encode()
	if err != nil {
		return err
	}
	return ab.verticesDB.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(vrx.Hash[:]); err == nil {
			return ErrVertexAlreadyExists
		}
		return txn.SetEntry(badger.NewEntry(vrx.Hash[:], buf))
	})
}

func (ab *AccountingBook) updateWaightAndThroughput(weight uint64) {
	if ab.weight.Load() < weight {
		ab.weight.Store(weight)
	}
	leafsCount := uint64(len(ab.dag.GetLeaves()))
	ab.throughput.Store(ab.throughput.Load() + leafsCount + 1)
}

func (ab *AccountingBook) isValidWeight(weight uint64) bool {
	current := ab.weight.Load()
	throughput := ab.throughput.Load()
	if throughput > current {
		return true
	}
	return weight >= current-throughput
}

func (ab *AccountingBook) getValidLeaves(ctx context.Context) (leftLeaf, rightLeaf *Vertex, err error) {
	var i int
	for _, item := range ab.dag.GetLeaves() {
		if i == 2 {
			break
		}

		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				err = errors.Join(ErrUnexpected, errors.New("vertex is nil"))
				return
			}
			err = ab.validateLeaf(ctx, vrx)
			if err != nil {
				ab.dag.DeleteVertex(string(vrx.Hash[:]))
				ab.removeTrxInVertex(vrx.Transaction.Hash[:])
				ab.log.Error(
					fmt.Sprintf("Accounting book rejected leaf hash [ %v ], from [ %v ], %s",
						vrx.Hash, vrx.SignerPublicAddress, err),
				)
				ab.updateWaightAndThroughput(vrx.Weight)
				continue
			}
			switch i {
			case 0:
				leftLeaf = vrx
			case 1:
				rightLeaf = vrx
			}
			i++

		default:
			err = errors.Join(ErrUnexpected, errors.New("cannot match vertex type"))
			return
		}
	}
	return
}

// CreateGenesis creates genesis vertex that will transfer spice to current node as a receiver.
func (ab *AccountingBook) CreateGenesis(subject string, spc spice.Melange, data []byte, reciverPublicAddress string) (Vertex, error) {
	if reciverPublicAddress == ab.signer.Address() {
		return Vertex{}, errors.Join(ErrGenesisRejected, errors.New("receiver and issuer cannot be the same wallet"))
	}

	trx, err := transaction.New(subject, spc, data, reciverPublicAddress, ab.signer)
	if err != nil {
		return Vertex{}, errors.Join(ErrGenesisRejected, err)
	}

	vrx, err := NewVertex(trx, [32]byte{}, [32]byte{}, 0, ab.signer)
	if err != nil {
		return Vertex{}, errors.Join(ErrGenesisRejected, err)
	}

	if err := ab.saveTrxInVertex(trx.Hash[:], vrx.Hash[:]); err != nil {
		return Vertex{}, errors.Join(ErrGenesisRejected, err)
	}

	ab.mux.Lock()
	defer ab.mux.Unlock()

	if err := ab.dag.AddVertexByID(string(vrx.Hash[:]), &vrx); err != nil {
		return Vertex{}, err
	}

	ab.throughput.Store(initialThroughput)
	ab.updateWaightAndThroughput(initialThroughput)

	ab.dagLoaded = true
	ab.genesisPublicAddress = ab.signer.Address()

	return vrx, nil
}

// LoadDag loads stream of Vertices in to the DAG.
func (ab *AccountingBook) LoadDag(ctx context.Context, cancelF context.CancelCauseFunc, cVrx <-chan *Vertex) {
	if ab.DagLoaded() {
		cancelF(ErrDagIsLoaded)
		return
	}

	defer ab.throughput.Store(initialThroughput)
	defer ab.updateWaightAndThroughput(initialThroughput)

	ab.mux.Lock()
	defer ab.mux.Unlock()

VertxLoop:
	for {
		select {
		case <-ctx.Done():
			break VertxLoop
		case vrx := <-cVrx:
			if vrx == nil {
				break VertxLoop
			}
			if err := ab.saveTrxInVertex(vrx.Transaction.Hash[:], vrx.Hash[:]); err != nil {
				cancelF(ErrLeafRejected)
				return
			}

			if err := ab.dag.AddVertexByID(string(vrx.Hash[:]), vrx); err != nil {
				cancelF(err)
				return
			}
		}
	}

	var lastVrx *Vertex
	for _, item := range ab.dag.GetVertices() {
		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				cancelF(ErrUnexpected)
				return
			}
			var addedHash [32]byte
			lastVrx = vrx
		connLoop:
			for _, conn := range [][32]byte{vrx.LeftParentHash, vrx.RightParentHash} {
				if conn == addedHash {
					break connLoop
				}
				if err := ab.dag.AddEdge(string(conn[:]), string(vrx.Hash[:])); err != nil {
					cancelF(err)
					return
				}
				addedHash = conn
			}
		default:
			cancelF(ErrUnexpected)
			return
		}
	}

	ab.dagLoaded = true
	ab.genesisPublicAddress = lastVrx.Transaction.IssuerAddress
}

// DagLoaded returns true if dag is loaded or false otherwise.
func (ab *AccountingBook) DagLoaded() bool {
	return ab.dagLoaded
}

// StreamDAG provides tow channels to subscribe to a stream of vertices.
// First streams verticies and second one streams possible errors.
func (ab *AccountingBook) StreamDAG(ctx context.Context) (<-chan *Vertex, <-chan error) {
	ab.mux.RLock()
	defer ab.mux.RUnlock()

	cVrx := make(chan *Vertex, 100)
	cDone := make(chan error, 1)
	go func(cVrx chan<- *Vertex, cDone chan<- error) {
	VerticesLoop:
		for _, item := range ab.dag.GetVertices() {
			select {
			case <-ctx.Done():
				break VerticesLoop
			default:
			}
			switch vrx := item.(type) {
			case *Vertex:
				cVrx <- vrx
			default:
				cDone <- ErrUnexpected
				break VerticesLoop
			}
		}
		close(cDone)
		close(cVrx)
	}(cVrx, cDone)

	return cVrx, cDone
}

// AddTrustedNode adds trusted node public address to the trusted nodes public address repository.
func (ab *AccountingBook) AddTrustedNode(trustedNodePublicAddress string) error {
	return ab.trustedNodesDB.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(trustedNodePublicAddress), []byte{})
	})
}

// RemoveTrustedNode removes trusted node public address from trusted nodes public address repository.
func (ab *AccountingBook) RemoveTrustedNode(trustedNodePublicAddress string) error {
	return ab.trustedNodesDB.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(trustedNodePublicAddress))
	})
}

// CreateLeaf creates leaf vertex also known as a tip.
// All the graph validations before adding the leaf happens in that function,
// Created leaf will be a subject of validation by another tip.
func (ab *AccountingBook) CreateLeaf(ctx context.Context, trx *transaction.Transaction) (Vertex, error) {
	if !ab.DagLoaded() {
		return Vertex{}, ErrDagIsNotLoaded
	}
	if trx.IssuerAddress == ab.signer.Address() {
		return Vertex{}, ErrCannotTransferFoundsViaOwnedNode
	}
	if trx.IssuerAddress == ab.genesisPublicAddress {
		return Vertex{}, ErrCannotTransferFoundsFromGenesisWallet
	}

	ok, err := ab.checkTrxInVertexExists(trx.Hash[:])
	if err != nil {
		ab.log.Error(fmt.Sprintf(
			"Accounting book creating transaction failed when checking trx to vertex mapping, %s", err,
		),
		)
		return Vertex{}, ErrUnexpected
	}
	if ok {
		return Vertex{}, ErrTrxInVertexAlreadyExists
	}

	ab.mux.Lock()
	defer ab.mux.Unlock()

	leftLeaf, rightLeaf, err := ab.getValidLeaves(ctx)
	if err != nil {
		return Vertex{}, err
	}

	if leftLeaf == nil {
		leftLeaf, rightLeaf, err = ab.getValidLeaves(ctx)
		if err != nil {
			return Vertex{}, err
		}
		if leftLeaf != nil {
			msgErr := errors.Join(ErrUnexpected, errors.New("expected at least one leaf but got zero"))
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s.", msgErr))
			return Vertex{}, msgErr
		}
	}

	if rightLeaf == nil {
		rightLeaf = leftLeaf
	}

	tip, err := NewVertex(
		*trx, leftLeaf.Hash, rightLeaf.Hash,
		calcNewWeight(leftLeaf.Weight, rightLeaf.Weight), ab.signer,
	)
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", tip.Hash, err))
		return Vertex{}, errors.Join(ErrNewLeafRejected, err)
	}
	if err := ab.saveTrxInVertex(trx.Hash[:], tip.Hash[:]); err != nil {
		ab.log.Error(
			fmt.Sprintf(
				"Accounting book vertex create failed saving transaction [ %v ] in tip [ %v ], %s.",
				trx.Hash[:], tip.Hash, err,
			),
		)
		return Vertex{}, ErrUnexpected
	}
	if err := ab.dag.AddVertexByID(string(tip.Hash[:]), &tip); err != nil {
		ab.removeTrxInVertex(trx.Hash[:])
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", tip.Hash, err))
		return Vertex{}, ErrNewLeafRejected
	}

	var addedHash [32]byte
	for _, vrx := range []*Vertex{leftLeaf, rightLeaf} {
		if vrx.Hash == addedHash {
			break
		}
		if err := ab.dag.AddEdge(string(vrx.Hash[:]), string(tip.Hash[:])); err != nil {
			ab.dag.DeleteVertex(string(tip.Hash[:]))
			ab.removeTrxInVertex(trx.Hash[:])
			ab.log.Error(
				fmt.Sprintf(
					"Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when adding an edge, %s,",
					vrx.Hash, vrx.SignerPublicAddress, vrx.LeftParentHash, vrx.RightParentHash, err),
			)
			return Vertex{}, ErrNewLeafRejected
		}
		addedHash = vrx.Hash
	}
	return tip, nil
}

// AddLeaf adds leaf known also as tip to the graph for future validation.
// Added leaf will be a subject of validation by another tip.
func (ab *AccountingBook) AddLeaf(ctx context.Context, leaf *Vertex) error {
	if !ab.DagLoaded() {
		return ErrDagIsNotLoaded
	}
	if leaf == nil {
		return errors.Join(ErrUnexpected, errors.New("leaf is nil"))
	}
	if leaf.Transaction.IssuerAddress == ab.genesisPublicAddress {
		return ErrCannotTransferFoundsFromGenesisWallet
	}

	ok, err := ab.checkVertexExists(leaf.Hash[:])
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book adding leaf failed when checking vertex exists, %s", err))
		return errors.Join(ErrUnexpected, err)
	}
	if ok {
		return ErrLeafAlreadyExists
	}
	ok, err = ab.checkTrxInVertexExists(leaf.Transaction.Hash[:])
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book adding leaf failed when checking if trx to vertex mapping exists, %s", err))
		return errors.Join(ErrUnexpected, err)
	}
	if ok {
		return ErrTrxInVertexAlreadyExists
	}

	if err := leaf.verify(ab.verifier); err != nil {
		ab.log.Error(
			fmt.Sprintf(
				"Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when verifying, %s.",
				leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
		)
		return ErrLeafRejected
	}

	ab.mux.Lock()
	defer ab.mux.Unlock()

	validatedLeafs := make([]*Vertex, 0, 2)

	for _, hash := range [][32]byte{leaf.LeftParentHash, leaf.RightParentHash} {
		item, err := ab.dag.GetVertex(string(hash[:]))
		if err != nil {
			ab.log.Info(
				fmt.Sprintf(
					"Accounting book proceeded with memorizing leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when reading vertex for future validation, %s.",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			if err := ab.repiter.insert(leaf); err != nil {
				ab.log.Error(
					fmt.Sprintf(
						"Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when reading vertex, %s.",
						leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
				)
				return ErrLeafRejected
			}
			return nil
		}
		existringLeaf, ok := item.(*Vertex)
		if !ok {
			return errors.Join(ErrUnexpected, errors.New("wrong leaf type"))
		}
		isLeaf, err := ab.dag.IsLeaf(string(hash[:]))
		if err != nil {
			ab.log.Error(
				fmt.Sprintf(
					"Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when validate is leaf, %s.",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			return ErrLeafRejected
		}
		if isLeaf {
			if err := ab.validateLeaf(ctx, existringLeaf); err != nil {
				ab.dag.DeleteVertex(string(existringLeaf.Hash[:]))
				ab.removeTrxInVertex(existringLeaf.Transaction.Hash[:])
				return errors.Join(ErrLeafRejected, err)
			}
			ab.updateWaightAndThroughput(existringLeaf.Weight)
		}
		validatedLeafs = append(validatedLeafs, existringLeaf)
	}

	if err := ab.saveTrxInVertex(leaf.Transaction.Hash[:], leaf.Hash[:]); err != nil {
		ab.log.Error(
			fmt.Sprintf(
				"Accounting book leaf add failed saving transaction [ %v ] in leaf [ %v ], %s.",
				leaf.Transaction.Hash[:], leaf.Hash, err,
			),
		)
		return errors.Join(ErrUnexpected, err)
	}

	if err := ab.dag.AddVertexByID(string(leaf.Hash[:]), leaf); err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", leaf.Hash, err))
		ab.removeTrxInVertex(leaf.Transaction.Hash[:])
		return ErrLeafRejected
	}

	var addedHash [32]byte
	for _, validVrx := range validatedLeafs {
		if validVrx.Hash == addedHash {
			break
		}
		if err := ab.dag.AddEdge(string(validVrx.Hash[:]), string(leaf.Hash[:])); err != nil {
			ab.dag.DeleteVertex(string(leaf.Hash[:]))
			ab.removeTrxInVertex(leaf.Transaction.Hash[:])
			ab.log.Error(
				fmt.Sprintf(
					"Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when adding edge, %s.",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			return ErrLeafRejected
		}
		addedHash = validVrx.Hash
	}

	return nil
}

// CalculateBalance traverses the graph starting from the recent accepted Vertex,
// and calculates the balance for the given address.
func (ab *AccountingBook) CalculateBalance(ctx context.Context, walletPubAddr string) (Balance, error) {
	ab.mux.RLock()
	defer ab.mux.RUnlock()

	var leaf *Vertex
	var ok bool
	for _, item := range ab.dag.GetLeaves() {
		leaf, ok = item.(*Vertex)
		if !ok {
			return Balance{}, errors.Join(ErrUnexpected, errors.New("calculate balance, cannot cast item to leaf"))
		}
	}
	if leaf == nil {
		return Balance{}, errors.Join(ErrUnexpected, errors.New("calculate balance, cannot read leaf"))
	}

	item, err := ab.dag.GetVertex(string(leaf.Hash[:]))
	if err != nil {
		return Balance{}, errors.Join(ErrUnexpected, err)
	}

	spiceOut := spice.New(0, 0)
	spiceIn := spice.New(0, 0)
	switch vrx := item.(type) {
	case *Vertex:
		if vrx == nil {
			return Balance{}, ErrUnexpected
		}
		if err := pourFounds(walletPubAddr, *vrx, &spiceIn, &spiceOut); err != nil {
			return Balance{}, err
		}
	default:
		return Balance{}, ErrUnexpected

	}
	visited := make(map[string]struct{})
	vertices, signal, err := ab.dag.AncestorsWalker(string(leaf.Hash[:]))
	if err != nil {
		return Balance{}, errors.Join(ErrUnexpected, err)
	}
	for ancestorID := range vertices {
		select {
		case <-ctx.Done():
			signal <- true
			return Balance{}, ErrLeafBallanceCalculationProcessStopped
		default:
		}
		if _, ok := visited[ancestorID]; ok {
			continue
		}
		visited[ancestorID] = struct{}{}

		item, err := ab.dag.GetVertex(ancestorID)
		if err != nil {
			signal <- true
			return Balance{}, errors.Join(ErrUnexpected, err)
		}
		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				return Balance{}, ErrUnexpected
			}
			if err := pourFounds(walletPubAddr, *vrx, &spiceIn, &spiceOut); err != nil {
				return Balance{}, err
			}
		default:
			signal <- true
			return Balance{}, ErrUnexpected
		}
	}

	s := spice.New(0, 0)
	if err := s.Supply(spiceIn); err != nil {
		return Balance{}, errors.Join(ErrBalanceCaclulationUnexpectedFailure, err)
	}

	if err := s.Drain(spiceOut, &spice.Melange{}); err != nil {
		return Balance{}, errors.Join(ErrBalanceCaclulationUnexpectedFailure, err)
	}

	return NewBalance(walletPubAddr, s), nil
}

// ReadTransactionByHash  reads transactions by hashes from DAG and DB.
func (ab *AccountingBook) ReadTransactionByHash(ctx context.Context, hash [32]byte) (transaction.Transaction, error) {
	var vertexHash []byte
	if err := ab.trxsToVertxDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(hash[:])
		if err != nil {
			if !errors.Is(err, badger.ErrKeyNotFound) {
				ab.log.Error(fmt.Sprintf("accountant error with reading transaction to vertex mapping, %s", err))
			}
			return err
		}

		item.Value(func(val []byte) error {
			vertexHash = val
			return nil
		})
		return nil
	}); err != nil {
		return transaction.Transaction{}, err
	}

	ab.mux.RLock()
	defer ab.mux.RUnlock()

	item, err := ab.dag.GetVertex(string(vertexHash))
	switch err {
	case nil:
		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				return transaction.Transaction{}, ErrUnexpected
			}
			return vrx.Transaction, nil // success
		default:
			return transaction.Transaction{}, ErrUnexpected
		}
	default:
		if !errors.Is(err, dag.IDUnknownError{}) {
			ab.log.Error(fmt.Sprintf("accountant error with reading vertex from DAG, %s", err))
		}
	}

	var trx transaction.Transaction
	if err := ab.verticesDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(vertexHash)
		if err != nil {
			if !errors.Is(err, badger.ErrKeyNotFound) {
				ab.log.Error(fmt.Sprintf("accountant error with reading vertex from DB, %s", err))
				return err
			}
			return ErrEntityNotFound
		}

		item.Value(func(val []byte) error {
			vrx, err := decodeVertex(val)
			if err != nil {
				return errors.Join(ErrUnexpected, err)
			}
			trx = vrx.Transaction
			return nil
		})
		return nil
	}); err != nil {
		return trx, err
	}

	return trx, nil // success
}

// Address returns signer public address that is a core cryptographic padlock for the DAG Vertices.
func (ab *AccountingBook) Address() string {
	return ab.signer.Address()
}
