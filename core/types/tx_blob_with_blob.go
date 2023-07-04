package types

import (
	"bytes"
	"io"

	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/rlp"
)

type BlobTxWithBlobs struct {
	*Transaction
	Blobs       []kzg4844.Blob
	Commitments []kzg4844.Commitment
	Proofs      []kzg4844.Proof
}

func NewBlobTxWithBlobs(tx *Transaction, blobs []kzg4844.Blob, commitments []kzg4844.Commitment, proofs []kzg4844.Proof) *BlobTxWithBlobs {
	if tx == nil {
		return nil
	}
	return &BlobTxWithBlobs{
		Transaction: tx,
		Blobs:       blobs,
		Commitments: commitments,
		Proofs:      proofs,
	}
}

type innerType struct {
	BlobTx      *BlobTx
	Blobs       []kzg4844.Blob
	Commitments []kzg4844.Commitment
	Proofs      []kzg4844.Proof
}

// DecodeRLP implements rlp.Decoder
func (tx *BlobTxWithBlobs) DecodeRLP2(s *rlp.Stream) error {
	kind, size, err := s.Kind()
	switch {
	case err != nil:
		return err
	case kind == rlp.List:
		// It's a legacy transaction.
		var inner LegacyTx
		err := s.Decode(&inner)
		if err == nil {
			tx.Transaction.setDecoded(&inner, rlp.ListSize(size))
		}
		return err
	default:
		// It's an EIP-2718 typed TX envelope.
		var b []byte
		if b, err = s.Bytes(); err != nil {
			return err
		}
		if b[0] == BlobTxType {
			var blobTypedTx innerType
			//err := s.Decode(&blobTypedTx)
			err := rlp.DecodeBytes(b[1:], &blobTypedTx)
			if err == nil {
				tx.Transaction = NewTx(blobTypedTx.BlobTx)
				tx.Blobs = blobTypedTx.Blobs
				tx.Commitments = blobTypedTx.Commitments
				tx.Proofs = blobTypedTx.Proofs
			}
			return err
		}
		inner, err := tx.Transaction.decodeTyped(b)
		if err == nil {
			tx.Transaction.setDecoded(inner, uint64(len(b)))
		}
		return err
	}
}

func (tx *BlobTxWithBlobs) DecodeRLP(s *rlp.Stream) error {
	return tx.DecodeRLP2(s)
	var typedTx Transaction
	err := s.Decode(&typedTx)
	if err == nil {
		tx.Transaction = &typedTx
		return nil
	}
	var blobTypedTx innerType
	if err := s.Decode(&blobTypedTx); err == nil {
		tx.Transaction = NewTx(blobTypedTx.BlobTx)
		tx.Blobs = blobTypedTx.Blobs
		tx.Commitments = blobTypedTx.Commitments
		tx.Proofs = blobTypedTx.Proofs
		return nil
	}
	return err
}

func (tx *BlobTxWithBlobs) EncodeRLP(w io.Writer) error {
	blobTx, ok := tx.Transaction.inner.(*BlobTx)
	if !ok {
		// For non-blob transactions, the encoding is just the transaction.
		return tx.Transaction.EncodeRLP(w)
	}

	// For blob transactions, the encoding is the transaction together with the blobs.
	// Use temporary buffer from pool.
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()

	buf.WriteByte(BlobTxType)
	innerValue := &innerType{
		BlobTx:      blobTx,
		Blobs:       tx.Blobs,
		Commitments: tx.Commitments,
		Proofs:      tx.Proofs,
	}
	err := rlp.Encode(buf, innerValue)
	if err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}
