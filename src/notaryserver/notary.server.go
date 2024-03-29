package notaryserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bartossh/Computantis/src/accountant"
	"github.com/bartossh/Computantis/src/cache"
	"github.com/bartossh/Computantis/src/grpcsecured"
	"github.com/bartossh/Computantis/src/logger"
	"github.com/bartossh/Computantis/src/protobufcompiled"
	"github.com/bartossh/Computantis/src/providers"
	"github.com/bartossh/Computantis/src/transaction"
	"github.com/bartossh/Computantis/src/transformers"
	"github.com/bartossh/Computantis/src/versioning"
)

const (
	proposeTrxTelemetryHistogram  = "propose_trx_request_duration"
	confirmTrxTelemetryHistogram  = "confirm_trx_request_duration"
	rejectTrxTelemetryHistogram   = "reject_trx_request_duration"
	awaitedTrxTelemetryHistogram  = "read_awaited_trx_request_duration"
	readDagTransactionsByAddress  = "read_dag_trx_only"
	approvedTrxTelemetryHistogram = "read_approved_trx_request_duration"
	balanceTelemetryHistogram     = "balance_read_duration"
	dataToSignTelemetryHistogram  = "data_to_sign_request_duration"
)

const (
	checkForRegisteredNodesInterval = 5 * time.Second
	transactionsUpdateTick          = time.Millisecond * 1000
)

const rxNewTrxIssuerAddrBufferSize = 800 // this value shall to be slightly bigger then maximum expected transaction throughput

var (
	ErrWrongPortSpecified = errors.New("port must be between 1 and 65535")
	ErrWrongMessageSize   = errors.New("message size must be between 1024 and 15000000")
	ErrTrxAlreadyExists   = errors.New("transaction already exists")
	ErrRequestIsEmpty     = errors.New("request is empty")
	ErrVerification       = errors.New("verification failed, forbidden")
	ErrThrottle           = errors.New("throttled request, to often reads are forbidden")
	ErrDataEmpty          = errors.New("empty data, invalid contract")
	ErrProcessing         = errors.New("processing request failed")
	ErrNoDataPresent      = errors.New("no entity found")
)

type verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type accounter interface {
	Address() string
	CreateLeaf(ctx context.Context, trx *transaction.Transaction) (accountant.Vertex, error)
	ReadTransactionByHash(ctx context.Context, hashe [32]byte) (transaction.Transaction, error)
	ReadDAGTransactionsByAddress(ctx context.Context, address string) ([]transaction.Transaction, error)
	CalculateBalance(ctx context.Context, walletPubAddr string) (accountant.Balance, error)
}

// RandomDataProvideValidator provides random binary data for signing to prove identity and
// the validator of data being valid and not expired.
type RandomDataProvideValidator interface {
	ProvideData(address string) []byte
	ValidateData(address string, data []byte) bool
}

type nodeNetworkingPublisher interface {
	PublishAddressesAwaitingTrxs(addresses []string, notaryNodeURL string) error
}

type piper interface {
	SendTrx(trx *protobufcompiled.Transaction) bool
	SendVrx(vrx *accountant.Vertex) bool
}

// Config contains configuration of the server.
type Config struct {
	NodePublicURL string `yaml:"public_url"`      // Public URL at which node can be reached.
	Cert          string `yaml:"certificate"`     // PEM certificate.
	Key           string `yaml:"key"`             // PEM key.
	Port          int    `yaml:"port"`            // Port to listen on.
	DataSizeBytes int    `yaml:"data_size_bytes"` // Size of the data to be stored in the transaction.
}

type server struct {
	protobufcompiled.UnimplementedNotaryAPIServer
	pub               nodeNetworkingPublisher
	randDataProv      RandomDataProvideValidator
	tele              providers.HistogramProvider
	log               logger.Logger
	rxNewTrxRecAddrCh chan string
	verifier          verifier
	acc               accounter
	cache             providers.AwaitedTrxCacheProviderBalanceCacher
	flash             providers.FlashbackMemoryAddressProvideRemover
	piper             piper
	nodePublicURL     string
	dataSize          int
}

// Run initializes routing and runs the server. To stop the server cancel the context.
// It blocks until the context is canceled.
func Run(
	ctx context.Context, c Config, pub nodeNetworkingPublisher, pv RandomDataProvideValidator, tele providers.HistogramProvider,
	log logger.Logger, v verifier, acc accounter, cache providers.AwaitedTrxCacheProviderBalanceCacher, flash providers.FlashbackMemoryAddressProvideRemover,
	p piper,
) error {
	var err error
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err = validateConfig(&c); err != nil {
		return err
	}

	var rxNewTrxRecAddrCh chan string
	if pub != nil {
		rxNewTrxRecAddrCh = make(chan string, rxNewTrxIssuerAddrBufferSize)
	}

	s := &server{
		pub:               pub,
		randDataProv:      pv,
		tele:              tele,
		log:               log,
		rxNewTrxRecAddrCh: rxNewTrxRecAddrCh,
		verifier:          v,
		acc:               acc,
		cache:             cache,
		flash:             flash,
		piper:             p,
		nodePublicURL:     c.NodePublicURL,
		dataSize:          c.DataSizeBytes,
	}

	s.tele.CreateUpdateObservableHistogram(proposeTrxTelemetryHistogram, "Propose trx endpoint request duration in [ ms ].")
	s.tele.CreateUpdateObservableHistogram(confirmTrxTelemetryHistogram, "Confirm trx endpoint request duration in [ ms ].")
	s.tele.CreateUpdateObservableHistogram(rejectTrxTelemetryHistogram, "Reject trx endpoint request duration in [ ms ].")
	s.tele.CreateUpdateObservableHistogram(awaitedTrxTelemetryHistogram, "Read awaited / issued trx endpoint request duration in [ ms ].")
	s.tele.CreateUpdateObservableHistogram(approvedTrxTelemetryHistogram, "Read approved trx endpoint request duration in [ ms ].")
	s.tele.CreateUpdateObservableHistogram(dataToSignTelemetryHistogram, "Generate data to sign endpoint request duration in [ ms ].")
	s.tele.CreateUpdateObservableHistogram(balanceTelemetryHistogram, "Calculate balance duration in [ ms ].")
	s.tele.CreateUpdateObservableHistogram(readDagTransactionsByAddress, "Read wallet transactions from DAG in [ ms ].")

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", c.Port))
	if err != nil {
		cancel()
		return err
	}

	grpcServer, err := grpcsecured.NewTLSServer(c.Cert, c.Key)
	if err != nil {
		cancel()
		return err
	}

	protobufcompiled.RegisterNotaryAPIServer(grpcServer, s)

	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			s.log.Fatal(fmt.Sprintf("Node [ %s ] cannot start server on port [ %v ], %s", s.acc.Address(), c.Port, err))
			cancel()
		}
	}()

	time.Sleep(time.Millisecond * 50) // just wait so the server can start

	defer grpcServer.GracefulStop()

	go s.runSubscriber(ctxx)

	<-ctxx.Done()

	return err
}

func validateConfig(c *Config) error {
	if c.Port == 0 || c.Port > 65535 {
		return ErrWrongPortSpecified
	}

	if c.DataSizeBytes < 1024 || c.DataSizeBytes > 15000000 {
		return ErrWrongMessageSize
	}

	return nil
}

func (s *server) runSubscriber(ctx context.Context) {
	if s.pub == nil {
		return
	}
	ticker := time.NewTicker(transactionsUpdateTick)
	defer ticker.Stop()

	receiverAddrSet := make(map[string]struct{}, 100)

	for {
		select {
		case <-ctx.Done():
			return
		case recAddr := <-s.rxNewTrxRecAddrCh:
			receiverAddrSet[recAddr] = struct{}{}
		case <-ticker.C:
			if len(receiverAddrSet) == 0 {
				continue
			}

			addresses := make([]string, 0, len(receiverAddrSet))
			for addr := range receiverAddrSet {
				addresses = append(addresses, addr)
			}

			s.pub.PublishAddressesAwaitingTrxs(addresses, s.nodePublicURL)

			maps.Clear(receiverAddrSet)
		}
	}
}

// Alive returns alive information such as wallet public address API version and API header of running server.
func (s *server) Alive(ctx context.Context, _ *emptypb.Empty) (*protobufcompiled.AliveData, error) {
	return &protobufcompiled.AliveData{
		PublicAddress: s.acc.Address(),
		ApiVersion:    versioning.ApiVersion,
		ApiHeader:     versioning.Header,
	}, nil
}

// Propose validates the transaction and then processes the transaction according to the data in transaction.
func (s *server) Propose(ctx context.Context, in *protobufcompiled.Transaction) (*emptypb.Empty, error) {
	t := time.Now()
	defer func() {
		d := time.Since(t)
		s.tele.RecordHistogramTime(proposeTrxTelemetryHistogram, d)
	}()

	if in == nil {
		return nil, ErrNoDataPresent
	}

	trx, err := transformers.ProtoTrxToTrx(in)
	if err != nil {
		s.log.Error(fmt.Sprintf("propose endpoint message is empty or invalid: %s", err))
		return nil, err
	}

	if err := trx.VerifyIssuer(s.verifier); err != nil {
		s.log.Error(fmt.Sprintf("propose endpoint verification failed: %s", err))
		return nil, ErrVerification
	}

	if trx.IsContract() {
		if len(trx.Data) > s.dataSize {
			s.log.Error(fmt.Sprintf("propose endpoint invalid transaction data size: %d", len(trx.Data)))
			return nil, ErrProcessing
		}
		if err := s.cache.SaveAwaitedTransaction(&trx); err != nil {
			s.log.Error(fmt.Sprintf("propose endpoint saving awaited trx for issuer [ %s ], %s", trx.IssuerAddress, err))
			return nil, ErrProcessing
		}

		go func(tx *protobufcompiled.Transaction) {
			if ok := s.piper.SendTrx(tx); !ok {
				s.log.Error(fmt.Sprintf("sending trx %v to gossiper failed, channel is closed", trx.Hash))
			}
			if s.pub != nil {
				s.rxNewTrxRecAddrCh <- tx.ReceiverAddress
			}
		}(in)

		return &emptypb.Empty{}, nil
	}

	vrx, err := s.acc.CreateLeaf(ctx, &trx)
	if err != nil {
		s.log.Error(fmt.Sprintf("propose endpoint creating leaf: %s", err))
		return nil, ErrProcessing
	}

	if ok := s.piper.SendVrx(&vrx); !ok {
		return nil, ErrProcessing
	}

	go func() {
		if err := s.flash.RemoveAddress(trx.IssuerAddress); err != nil {
			s.log.Error(fmt.Sprintf("confirm endpoint removing issuer address [ %s ] from flash failed: %s", in.IssuerAddress, err))
		}
		if err := s.flash.RemoveAddress(trx.ReceiverAddress); err != nil {
			s.log.Error(fmt.Sprintf("confirm endpoint removing receiver address [ %s ] from flash failed: %s", in.ReceiverAddress, err))
		}
		if err := s.cache.RemoveBalance(trx.IssuerAddress); err != nil {
			s.log.Error(fmt.Sprintf("confirm endpoint removing cached balance for address [ %s ] from flash failed: %s", in.IssuerAddress, err))
		}
		if err := s.cache.RemoveBalance(trx.ReceiverAddress); err != nil {
			s.log.Error(fmt.Sprintf("confirm endpoint removing cached balance for address [ %s ] from flash failed: %s", in.ReceiverAddress, err))
		}
	}()

	return &emptypb.Empty{}, nil
}

// Confirm validates the transaction and processes the confirmation according to the data in transaction.
func (s *server) Confirm(ctx context.Context, in *protobufcompiled.Transaction) (*emptypb.Empty, error) {
	t := time.Now()
	defer func() {
		d := time.Since(t)
		s.tele.RecordHistogramTime(confirmTrxTelemetryHistogram, d)
	}()

	trx, err := transformers.ProtoTrxToTrx(in)
	if err != nil {
		s.log.Error(fmt.Sprintf("confirm endpoint message is empty or invalid: %s", err))
		return nil, err
	}

	if err := trx.VerifyIssuerReceiver(s.verifier); err != nil {
		s.log.Error(fmt.Sprintf(
			"confirm endpoint failed to verify trx hash [ %x ] from receiver [ %s ], %s", trx.Hash, trx.ReceiverAddress, err.Error(),
		))
		return nil, ErrVerification
	}

	_, err = s.cache.RemoveAwaitedTransaction(trx.Hash, trx.ReceiverAddress)
	if err != nil {
		s.log.Error(
			fmt.Sprintf(
				"confirm endpoint failed to remove awaited trx hash [ %x ] from receiver [ %s ] , %s", trx.Hash, trx.ReceiverAddress, err,
			))
		if errors.Is(err, cache.ErrTransactionNotFound) {
			return nil, ErrNoDataPresent
		}
		return nil, ErrProcessing
	}

	vrx, err := s.acc.CreateLeaf(ctx, &trx)
	if err != nil {
		s.log.Error(fmt.Sprintf("confirm endpoint creating leaf for transaction [ %x ] : %s", in.Hash, err))
		return nil, ErrProcessing
	}

	if ok := s.piper.SendVrx(&vrx); !ok {
		return nil, ErrProcessing
	}

	go func() {
		if err := s.flash.RemoveAddress(trx.IssuerAddress); err != nil {
			s.log.Error(fmt.Sprintf("confirm endpoint removing issuer address [ %s ] from flash failed: %s", in.IssuerAddress, err))
		}
		if err := s.flash.RemoveAddress(trx.ReceiverAddress); err != nil {
			s.log.Error(fmt.Sprintf("confirm endpoint removing receiver address [ %s ] from flash failed: %s", in.ReceiverAddress, err))
		}
		if err := s.cache.RemoveBalance(trx.IssuerAddress); err != nil {
			s.log.Error(fmt.Sprintf("confirm endpoint removing cached balance for address [ %s ] from flash failed: %s", in.IssuerAddress, err))
		}
		if err := s.cache.RemoveBalance(trx.ReceiverAddress); err != nil {
			s.log.Error(fmt.Sprintf("confirm endpoint removing cached balance for address [ %s ] from flash failed: %s", in.ReceiverAddress, err))
		}
	}()

	return &emptypb.Empty{}, nil
}

// Reject validates request and then attempts to remove transaction from awaited transactions kept by this node.
func (s *server) Reject(ctx context.Context, in *protobufcompiled.SignedHash) (*emptypb.Empty, error) {
	t := time.Now()
	defer func() {
		d := time.Since(t)
		s.tele.RecordHistogramTime(rejectTrxTelemetryHistogram, d)
	}()

	if in == nil {
		return nil, ErrRequestIsEmpty
	}

	if err := s.verifier.Verify(in.Data, in.Signature, [32]byte(in.Hash), in.Address); err != nil {
		s.log.Error(fmt.Sprintf("reject endpoint failed to verify signature of transaction [ %x ] for address: %s, %s", in.Hash, in.Address, err))
		return nil, ErrProcessing
	}

	trx, err := s.cache.RemoveAwaitedTransaction([32]byte(in.Data), in.Address) // in.Data holds the transaction hash where in.Hash is a message digest
	if err != nil {
		s.log.Error(fmt.Sprintf("reject endpoint failed removing transaction [ %x ] for address [ %s ], %s", in.Hash, in.Address, err))
		if errors.Is(err, cache.ErrTransactionNotFound) {
			return nil, ErrNoDataPresent
		}
		return nil, ErrProcessing
	}

	vrx, err := s.acc.CreateLeaf(ctx, &trx)
	if err != nil {
		s.log.Error(fmt.Sprintf("reject endpoint creating leaf for transaction [ %x ] : %s", in.Hash, err))
		return nil, ErrProcessing
	}

	if ok := s.piper.SendVrx(&vrx); !ok {
		return nil, ErrProcessing
	}

	go func() {
		if err := s.flash.RemoveAddress(trx.IssuerAddress); err != nil {
			s.log.Error(fmt.Sprintf("confirm endpoint removing receiver address [ %s ] from flash failed: %s", in.Address, err))
		}
		if err := s.cache.RemoveBalance(trx.IssuerAddress); err != nil {
			s.log.Error(fmt.Sprintf("confirm endpoint removing cached balance for address [ %s ] from flash failed: %s", in.Address, err))
		}
	}()

	return &emptypb.Empty{}, nil
}

// Waiting endpoint returns all the awaited transactions for given address, those received and issued.
func (s *server) Waiting(ctx context.Context, in *protobufcompiled.SignedHash) (*protobufcompiled.Transactions, error) {
	t := time.Now()
	defer func() {
		d := time.Since(t)
		s.tele.RecordHistogramTime(awaitedTrxTelemetryHistogram, d)
	}()

	if ok := s.randDataProv.ValidateData(in.Address, in.Data); !ok {
		s.log.Error(fmt.Sprintf("waiting transactions endpoint, failed to validate data for address: %s", in.Address))
		return nil, ErrVerification
	}

	if err := s.verifier.Verify(in.Data, in.Signature, [32]byte(in.Hash), in.Address); err != nil {
		s.log.Error(fmt.Sprintf("waiting endpoint failed to verify signature for address: %s, %s", in.Address, err))
		return nil, ErrVerification
	}

	trxs, err := s.cache.ReadTransactions(in.Address)
	if err != nil {
		s.log.Error(fmt.Sprintf("waiting endpoint failed to read awaited transactions for address: %s, %s", in.Address, err))
		return nil, ErrProcessing
	}

	result := &protobufcompiled.Transactions{Array: make([]*protobufcompiled.Transaction, 0, len(trxs)), Len: uint64(len(trxs))}
	for _, trx := range trxs {
		protoTrx, err := transformers.TrxToProtoTrx(trx)
		if err != nil {
			s.log.Warn(fmt.Sprintf("waiting endpoint failed to map trx to protobuf trx for address: %s, %s", in.Address, err))
			continue
		}
		result.Array = append(result.Array, protoTrx)
	}

	return result, nil
}

// Saved returns saved transactions in the graph.
func (s *server) Saved(ctx context.Context, in *protobufcompiled.SignedHash) (*protobufcompiled.Transaction, error) {
	t := time.Now()
	defer func() {
		d := time.Since(t)
		s.tele.RecordHistogramTime(approvedTrxTelemetryHistogram, d)
	}()

	if err := s.verifier.Verify(in.Data, in.Signature, [32]byte(in.Hash), in.Address); err != nil {
		s.log.Error(fmt.Sprintf("waiting endpoint failed to verify signature for address: %s, %s", in.Address, err))
		return nil, ErrVerification
	}

	trx, err := s.acc.ReadTransactionByHash(ctx, [32]byte(in.Data))
	if err != nil {
		s.log.Error(fmt.Sprintf("approved transactions endpoint failed to read hash [ %x ] for address: %s, %s", in.Hash, in.Address, err))
		return nil, ErrProcessing
	}

	if trx.Hash == [32]byte{} {
		s.log.Error(fmt.Sprintf("approved transactions endpoint failed to read hash [ %x ] for address: %s, %s", in.Hash, in.Address, ErrNoDataPresent))
		return nil, ErrNoDataPresent
	}

	protoTrx, err := transformers.TrxToProtoTrx(trx)
	if err != nil {
		return nil, err
	}
	return protoTrx, nil
}

// Data generates temporary data blob for receiver to sign and proof the its identity.
func (s *server) Data(ctx context.Context, in *protobufcompiled.Address) (*protobufcompiled.DataBlob, error) {
	t := time.Now()
	defer func() {
		d := time.Since(t)
		s.tele.RecordHistogramTime(dataToSignTelemetryHistogram, d)
	}()

	if in.Public == "" {
		s.log.Error("empty request on data endpoint")
		return nil, ErrRequestIsEmpty
	}

	d := s.randDataProv.ProvideData(in.Public)

	return &protobufcompiled.DataBlob{Blob: d}, nil
}

// Balance returns balanse for account owner.
// TODO: Find better way of requesting balance - sign blob data!
func (s *server) Balance(ctx context.Context, in *protobufcompiled.SignedHash) (*protobufcompiled.Spice, error) {
	ok, err := s.flash.HasAddress(in.Address)
	if err != nil {
		s.log.Error(fmt.Sprintf("balance endpoint failed to read from flash the address: %s, %s", in.Address, err))
		return nil, ErrProcessing
	}
	if ok {
		return nil, ErrThrottle
	}

	t := time.Now()
	defer func() {
		d := time.Since(t)
		s.tele.RecordHistogramTime(balanceTelemetryHistogram, d)
	}()

	if string(in.Data) != in.Address {
		return nil, ErrVerification
	}

	if err := s.verifier.Verify(in.Data, in.Signature, [32]byte(in.Hash), in.Address); err != nil {
		s.log.Error(fmt.Sprintf("balance endpoint failed to verify signature for address: %s, %s", in.Address, err))
		return nil, ErrVerification
	}

	if s, err := s.cache.ReadBalance(in.Address); err == nil {
		return &protobufcompiled.Spice{
			Currency:              s.Currency,
			SupplementaryCurrency: s.SupplementaryCurrency,
		}, nil
	}

	balance, err := s.acc.CalculateBalance(ctx, in.Address)
	if err != nil {
		s.log.Error(fmt.Sprintf("balance endpoint failed to read balance for address: %s, %s", in.Address, err))
		return nil, ErrProcessing
	}

	go func() {
		if err := s.cache.SaveBalance(in.Address, balance.Spice); err != nil {
			s.log.Error(fmt.Sprintf("balance endpoint failed to cache balance: %s, %s", in.Address, err))
		}
	}()

	return &protobufcompiled.Spice{
		Currency:              balance.Spice.Currency,
		SupplementaryCurrency: balance.Spice.SupplementaryCurrency,
	}, nil
}

// TransactionsInDAG returns all the transactions in a DAG that given address appears in as receiver or issuer.
// Do not reads transactions from vertices storage that contains transactions after the DAG is truncated.
func (s *server) TransactionsInDAG(ctx context.Context, in *protobufcompiled.SignedHash) (*protobufcompiled.Transactions, error) {
	ok, err := s.flash.HasAddress(in.Address)
	if err != nil {
		s.log.Error(fmt.Sprintf("transactions in DAG endpoint failed to read from flash the address: %s, %s", in.Address, err))
		return nil, ErrProcessing
	}
	if ok {
		return nil, ErrThrottle
	}

	t := time.Now()
	defer func() {
		d := time.Since(t)
		s.tele.RecordHistogramTime(readDagTransactionsByAddress, d)
	}()

	if ok := s.randDataProv.ValidateData(in.Address, in.Data); !ok {
		s.log.Error(fmt.Sprintf("waiting transactions endpoint failed to validate data for address: %s", in.Address))
		return nil, ErrVerification
	}

	if err := s.verifier.Verify(in.Data, in.Signature, [32]byte(in.Hash), in.Address); err != nil {
		s.log.Error(fmt.Sprintf("waiting endpoint failed to verify signature for address: %s, %s", in.Address, err))
		return nil, ErrVerification
	}

	trxs, err := s.acc.ReadDAGTransactionsByAddress(ctx, in.Address)
	if err != nil {
		return nil, err
	}

	result := &protobufcompiled.Transactions{Array: make([]*protobufcompiled.Transaction, 0, len(trxs)), Len: uint64(len(trxs))}
	for _, trx := range trxs {
		protoTrx, err := transformers.TrxToProtoTrx(trx)
		if err != nil {
			s.log.Warn(fmt.Sprintf("transaction in dag endpoint failed to map trx to protobuf trx for address: %s, %s", in.Address, err))
			continue
		}
		result.Array = append(result.Array, protoTrx)
	}

	return result, nil
}
