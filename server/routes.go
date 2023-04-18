package server

import (
	"github.com/bartossh/The-Accountant/transaction"
	"github.com/gofiber/fiber/v2"
)

func (s *server) alive(c *fiber.Ctx) error {
	return c.JSON(map[string]interface{}{"alive": true})
}

// SearchAddressRquest is a request to search for address.
type SearchAddressRquest struct {
	Address string `json:"address"`
}

// SearchAddressResponse is a response for address search.
type SearchAddressResponse struct {
	Addresses []string `json:"addresses"`
}

func (s *server) address(c *fiber.Ctx) error {
	var req SearchAddressRquest

	if err := c.BodyParser(&req); err != nil {
		// TODO: log err
		return fiber.ErrBadRequest
	}
	results, err := s.repo.FindAddress(c.Context(), req.Address, queryLimit)
	if err != nil {
		// TODO: log error
		return fiber.ErrNotFound
	}

	return c.JSON(SearchAddressResponse{
		Addresses: results,
	})
}

// SearchBlockRequest is a request to search for block.
type SearchBlockRequest struct {
	RawTrxHash [32]byte `json:"raw_trx_hash"`
}

// searchBlockResponse is a response for block search.
type SearchBlockResponse struct {
	RawBlockHash [32]byte `json:"raw_block_hash"`
}

func (s *server) trxInBlock(c *fiber.Ctx) error {
	var req SearchBlockRequest

	if err := c.BodyParser(&req); err != nil {
		// TODO: log err
		return fiber.ErrBadRequest
	}

	res, err := s.repo.FindTransactionInBlockHash(c.Context(), req.RawTrxHash)
	if err != nil {
		// TODO: log error
		return fiber.ErrNotFound
	}

	return c.JSON(SearchBlockResponse{
		RawBlockHash: res,
	})
}

// TransactionProposeRequest is a request to propose a transaction.
type TransactionProposeRequest struct {
	ReceiverAddr string                  `json:"receiver_addr"`
	Transaction  transaction.Transaction `json:"transaction"`
}

// TransactionConfirmProposeResponse is a response for transaction propose.
type TransactionConfirmProposeResponse struct {
	Succes  bool     `json:"success"`
	TrxHash [32]byte `json:"trx_hash"`
}

func (s *server) propose(c *fiber.Ctx) error {
	var req TransactionProposeRequest
	if err := c.BodyParser(&req); err != nil {
		// TODO log err
		return fiber.ErrBadRequest
	}

	if err := s.bookkeeping.WriteIssuerSignedTransactionForReceiver(c.Context(), req.ReceiverAddr, &req.Transaction); err != nil {
		// TODO log error
		return c.JSON(TransactionConfirmProposeResponse{
			Succes:  false,
			TrxHash: req.Transaction.Hash,
		})
	}

	return c.JSON(TransactionConfirmProposeResponse{
		Succes:  true,
		TrxHash: req.Transaction.Hash,
	})
}

func (s *server) confirm(c *fiber.Ctx) error {
	var trx transaction.Transaction
	if err := c.BodyParser(&trx); err != nil {
		// TODO: log err
		return fiber.ErrBadRequest
	}

	if err := s.bookkeeping.WriteCandidateTransaction(c.Context(), &trx); err != nil {
		// TODO: log err
		return c.JSON(TransactionConfirmProposeResponse{
			Succes:  false,
			TrxHash: trx.Hash,
		})
	}

	return c.JSON(TransactionConfirmProposeResponse{
		Succes:  true,
		TrxHash: trx.Hash,
	})
}

// AwaitedTransactionRequest is a request to get awaited transactions for given address.
// Request contains of Address for which Awaited Transacttions are requested, Data in binary format,
// Hash of Data ad Signature of the Data to prove that entity doing the request is an Address owner.
type AwaitedTransactionRequest struct {
	Address   string   `json:"address"`
	Data      []byte   `json:"data"`
	Hash      [32]byte `json:"hash"`
	Signature []byte   `json:"signature"`
}

// AwaitedTransactionResponse is a response for awaited transactions request.
type AwaitedTransactionResponse struct {
	Success             bool                      `json:"success"`
	AwaitedTransactions []transaction.Transaction `json:"awaited_transactions"`
}

func (s *server) awaited(c *fiber.Ctx) error {
	var req AwaitedTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}

	if ok := s.pv.ValidateData(req.Address, req.Data); !ok {
		return fiber.ErrForbidden
	}

	trxs, err := s.bookkeeping.ReadAwaitedTransactionsForAddress(c.Context(), req.Data, req.Signature, req.Hash, req.Address)
	if err != nil {
		// TODO log error
		return c.JSON(AwaitedTransactionResponse{
			Success:             false,
			AwaitedTransactions: nil,
		})
	}

	return c.JSON(AwaitedTransactionResponse{
		Success:             true,
		AwaitedTransactions: trxs,
	})
}

// DataToSignRequest is a request to get data to sign for proving identity.
type DataToSignRequest struct {
	Address string `json:"address"`
}

// DataToSignRequest is a response containing data to sign for proving identity.
type DataToSignResponse struct {
	Data []byte `json:"message"`
}

func (s *server) data(c *fiber.Ctx) error {
	var req DataToSignRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	d := s.pv.ProvideData(req.Address)
	return c.JSON(DataToSignResponse{Data: d})
}