package transaction

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/src/spice"
	msgpackv2 "github.com/shamaton/msgpack/v2"
	"github.com/vmihailenco/msgpack"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	minAddressLength     = 49
	ExpirationTimeInDays = 7 // transaction validity expiration time in days. TODO: move to config
)

var (
	ErrTransactionHasAFutureTime        = errors.New("transaction has a future time")
	ErrExpiredTransaction               = errors.New("transaction has expired")
	ErrTransactionHashIsInvalid         = errors.New("transaction hash is invalid")
	ErrSignatureNotValidOrDataCorrupted = errors.New("signature not valid or data are corrupted")
	ErrSubjectIsEmpty                   = errors.New("subject cannot be empty")
	ErrAddressIsInvalid                 = errors.New("address is invalid")
	ErrNilTransaction                   = errors.New("nil transaction")
)

// TrxAddressesSubscriberCallback is a method or function performing compotation on the transactions addresses.
type TrxAddressesSubscriberCallback func(addresses []string, storageNodeURL string)

// Signer provides signing and address methods.
type Signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

// Verifier provides signature verification method.
type Verifier interface {
	Verify(message, signature []byte, hash [32]byte, issuer string) error
}

// Transaction contains transaction information, subject type, subject data, signatures and public keys.
// Transaction is valid for a week from being issued.
type Transaction struct {
	ID                any           `json:"-"                  bson:"_id"                db:"id"                    msgpack:"-"`
	CreatedAt         time.Time     `json:"created_at"         bson:"created_at"         db:"created_at"            msgpack:"created_at"`
	IssuerAddress     string        `json:"issuer_address"     bson:"issuer_address"     db:"issuer_address"        msgpack:"issuer_address"`
	ReceiverAddress   string        `json:"receiver_address"   bson:"receiver_address"   db:"receiver_address"      msgpack:"receiver_address"`
	Subject           string        `json:"subject"            bson:"subject"            db:"subject"               msgpack:"subject"`
	Data              []byte        `json:"data"               bson:"data"               db:"data"                  msgpack:"data"`
	IssuerSignature   []byte        `json:"issuer_signature"   bson:"issuer_signature"   db:"issuer_signature"      msgpack:"issuer_signature"`
	ReceiverSignature []byte        `json:"receiver_signature" bson:"receiver_signature" db:"receiver_signature"    msgpack:"receiver_signature"`
	Hash              [32]byte      `json:"hash"               bson:"hash"               db:"hash"                  msgpack:"hash"`
	Spice             spice.Melange `json:"spice"              bson:"spice"              db:"spice"                 msgpack:"spice"`
}

// New creates new transaction signed by the issuer.
func New(subject string, spice spice.Melange, data []byte, receiverAddress string, issuer Signer) (Transaction, error) {
	if len(subject) == 0 {
		return Transaction{}, ErrSubjectIsEmpty
	}

	if len(receiverAddress) < minAddressLength {
		return Transaction{}, ErrAddressIsInvalid
	}
	// TODO: create and use address verifier

	createdAt := time.Now()

	msgLen := len(subject) + len(data) + len(issuer.Address()) + len(receiverAddress) + 24
	message := make([]byte, msgLen)
	n := copy(message, []byte(subject))
	n += copy(message[n:], data)
	n += copy(message[n:], []byte(issuer.Address()))
	n += copy(message[n:], []byte(receiverAddress))
	b0 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b0, uint64(createdAt.UnixNano()))
	n += copy(message[n:], b0)
	b1 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b1, spice.Currency)
	n += copy(message[n:], b1)
	b2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b2, spice.SupplementaryCurrency)
	copy(message[n:], b2)

	hash, signature := issuer.Sign(message)

	return Transaction{
		ID:                primitive.NilObjectID,
		CreatedAt:         createdAt,
		Hash:              hash,
		IssuerAddress:     issuer.Address(),
		ReceiverAddress:   receiverAddress,
		Subject:           subject,
		Data:              data,
		IssuerSignature:   signature,
		ReceiverSignature: []byte{},
		Spice:             spice,
	}, nil
}

// Sign verifies issuer signature and signs Transaction by the receiver.
func (t *Transaction) Sign(receiver Signer, v Verifier) ([32]byte, error) {
	now := time.Now()

	if t.CreatedAt.Unix() > now.Unix() {
		return [32]byte{}, ErrTransactionHasAFutureTime
	}

	if addTime(t.CreatedAt).Unix() < now.Unix() {
		return [32]byte{}, ErrExpiredTransaction
	}

	if receiver.Address() != t.ReceiverAddress {
		return [32]byte{}, ErrAddressIsInvalid
	}

	msgLen := len(t.Subject) + len(t.Data) + len(t.IssuerAddress) + len(receiver.Address()) + 24
	message := make([]byte, msgLen)
	n := copy(message, []byte(t.Subject))
	n += copy(message[n:], t.Data)
	n += copy(message[n:], []byte(t.IssuerAddress))
	n += copy(message[n:], []byte(receiver.Address()))
	b0 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b0, uint64(t.CreatedAt.UnixNano()))
	n += copy(message[n:], b0)
	b1 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b1, t.Spice.Currency)
	n += copy(message[n:], b1)
	b2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b2, t.Spice.SupplementaryCurrency)
	copy(message[n:], b2)

	if err := v.Verify(message, t.IssuerSignature, [32]byte(t.Hash), t.IssuerAddress); err != nil {
		return [32]byte{}, errors.Join(ErrSignatureNotValidOrDataCorrupted, err)
	}

	hash, signature := receiver.Sign(message)

	if !bytes.Equal(hash[:], t.Hash[:]) {
		return [32]byte{}, ErrTransactionHashIsInvalid
	}

	t.ReceiverSignature = signature
	return hash, nil
}

// IsContract returns true if the transaction contains not empty data buffer that is recognised as transaction with contract.
func (t Transaction) IsContract() bool {
	return len(t.Data) != 0
}

// IsSpiceTransfer returns true if the transaction transfers spice.
func (t Transaction) IsSpiceTransfer() bool {
	return !t.Spice.Empty()
}

// IsEmpty returns true if transaction doesn't transfers tokens or has no data.
func (t Transaction) IsEmpty() bool {
	return !t.IsContract() && !t.IsSpiceTransfer()
}

func (t *Transaction) VerifyIssuer(v Verifier) error {
	message := t.GetMessage()
	return v.Verify(message, t.IssuerSignature, t.Hash, t.IssuerAddress)
}

// Verify verifies transaction signatures.
func (t *Transaction) VerifyIssuerReceiver(v Verifier) error {
	message := t.GetMessage()
	if err := v.Verify(message, t.IssuerSignature, t.Hash, t.IssuerAddress); err != nil {
		return err
	}
	return v.Verify(message, t.ReceiverSignature, t.Hash, t.ReceiverAddress)
}

// GeMessage returns message used for signature validation.
func (t *Transaction) GetMessage() []byte {
	msgLen := len(t.Subject) + len(t.Data) + len(t.IssuerAddress) + len(t.ReceiverAddress) + 24
	message := make([]byte, msgLen)
	n := copy(message, []byte(t.Subject))
	n += copy(message[n:], t.Data)
	n += copy(message[n:], []byte(t.IssuerAddress))
	n += copy(message[n:], []byte(t.ReceiverAddress))
	b0 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b0, uint64(t.CreatedAt.UnixNano()))
	n += copy(message[n:], b0)
	b1 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b1, t.Spice.Currency)
	n += copy(message[n:], b1)
	b2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b2, t.Spice.SupplementaryCurrency)
	copy(message[n:], b2)

	return message
}

// CompareIssuerData compare transactions from Issuer perspective.
func (t *Transaction) CompareIssuerData(tx *Transaction) (bool, error) {
	if t == nil || tx == nil {
		return false, ErrNilTransaction
	}
	if t.Hash != tx.Hash {
		return false, nil
	}
	if t.IssuerAddress != tx.IssuerAddress {
		return false, nil
	}
	if t.ReceiverAddress != tx.ReceiverAddress {
		return false, nil
	}
	if !t.CreatedAt.Equal(tx.CreatedAt) {
		return false, nil
	}
	if !bytes.Equal(t.IssuerSignature, tx.IssuerSignature) {
		return false, nil
	}
	if !bytes.Equal(t.Data, tx.Data) {
		return false, nil
	}
	if t.Spice.Currency != tx.Spice.Currency || t.Spice.SupplementaryCurrency != tx.Spice.SupplementaryCurrency {
		return false, nil
	}
	return true, nil
}

// Encode encodes transaction to bytes slice.
func (t *Transaction) Encode() ([]byte, error) {
	buf, err := msgpack.Marshal(*t)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// String stringifies the transaction
func (t *Transaction) String() string {
	return fmt.Sprintf(
		"%s: [ SPICE %s ], created %s | %s => %s [ DATA SIZE %v ]",
		t.Subject, t.Spice, t.CreatedAt.UTC(), t.IssuerAddress, t.ReceiverAddress, len(t.Data),
	)
}

// Decode decodes slice buffer to transaction.
func Decode(buf []byte) (Transaction, error) {
	var t Transaction
	err := msgpackv2.Unmarshal(buf, &t)
	return t, err
}

func addTime(t time.Time) time.Time {
	return t.AddDate(0, 0, ExpirationTimeInDays)
}
