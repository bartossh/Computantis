package accountant

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/bartossh/Computantis/src/transaction"

	msgpackv2 "github.com/shamaton/msgpack/v2"
	"github.com/vmihailenco/msgpack"
)

func calcNewWeight(leftWeight, rightWeight uint64) uint64 {
	return max(leftWeight, rightWeight) + 1
}

// Vertex is a Direct Acyclic Graph vertex that creates a AccountingBook inner graph.
type Vertex struct {
	SignerPublicAddress string                  `msgpack:"signer_public_address"`
	CreatedAt           time.Time               `msgpack:"created_at"`
	Signature           []byte                  `msgpack:"signature"`
	Transaction         transaction.Transaction `msgpack:"transaction"`
	Hash                [32]byte                `msgpack:"hash"`
	LeftParentHash      [32]byte                `msgpack:"left_parent_hash"`
	RightParentHash     [32]byte                `msgpack:"right_parent_hash"`
	Weight              uint64                  `msgpack:"weight"`
}

// NewVertex creates new Vertex but first validates transaction legitimacy.
// It is assumed that the transaction is verified.
func NewVertex(
	trx transaction.Transaction,
	leftParentHash, rightParentHash [32]byte,
	weight uint64, signer Signer,
) (Vertex, error) {
	candidate := Vertex{
		SignerPublicAddress: signer.Address(),
		CreatedAt:           time.Now(),
		Signature:           nil,
		Transaction:         trx,
		Hash:                [32]byte{},
		LeftParentHash:      leftParentHash,
		RightParentHash:     rightParentHash,
		Weight:              weight,
	}

	candidate.sign(signer)

	return candidate, nil
}

func (v *Vertex) initData() []byte {
	blockData := make([]byte, 0, 16)
	blockData = binary.LittleEndian.AppendUint64(blockData, uint64(v.CreatedAt.UnixNano()))
	blockData = binary.LittleEndian.AppendUint64(blockData, uint64(v.Weight))
	return bytes.Join([][]byte{
		v.Transaction.Hash[:], v.LeftParentHash[:], v.RightParentHash[:], blockData,
	},
		[]byte{},
	)
}

func (v *Vertex) sign(signer Signer) {
	data := v.initData()
	v.Hash, v.Signature = signer.Sign(data)
}

func (v *Vertex) verify(verifier signatureVerifier) error {
	switch len(v.Transaction.ReceiverSignature) != 0 {
	case true:
		if err := v.Transaction.VerifyIssuerReceiver(verifier); err != nil {
			return err
		}
	default:
		if err := v.Transaction.VerifyIssuer(verifier); err != nil {
			return err
		}
	}

	data := v.initData()
	return verifier.Verify(data, v.Signature[:], v.Hash, v.SignerPublicAddress)
}

func (v *Vertex) encode() ([]byte, error) {
	buf, err := msgpack.Marshal(*v)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func decodeVertex(buf []byte) (Vertex, error) {
	var v Vertex
	err := msgpackv2.Unmarshal(buf, &v)
	return v, err
}
