package types

import (
	"bytes"
	"io"

	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/rlp"
)

type BlobTxWithBlobs struct {
	Transaction
	Blobs       []kzg4844.Blob
	Commitments []kzg4844.Commitment
	Proofs      []kzg4844.Proof
}

func NewBlobTxWithBlobs(tx *Transaction, blobs []kzg4844.Blob, commitments []kzg4844.Commitment, proofs []kzg4844.Proof) *BlobTxWithBlobs {
	if tx == nil {
		return nil
	}
	return &BlobTxWithBlobs{
		Transaction: *tx,
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
				tx.Transaction = *NewTx(blobTypedTx.BlobTx)
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
		tx.Transaction = typedTx
		return nil
	}
	var blobTypedTx innerType
	if err := s.Decode(&blobTypedTx); err == nil {
		tx.Transaction = *NewTx(blobTypedTx.BlobTx)
		tx.Blobs = blobTypedTx.Blobs
		tx.Commitments = blobTypedTx.Commitments
		tx.Proofs = blobTypedTx.Proofs
		return nil
	}
	return err
}

type innerType2 struct {
	BlobTx      rlp.RawValue
	Blobs       []kzg4844.Blob
	Commitments []kzg4844.Commitment
	Proofs      []kzg4844.Proof
}

type wrapper struct {
	TxType byte
	Inner  innerType2
}

func (tx *BlobTxWithBlobs) EncodeRLP(w io.Writer) error {
	var b bytes.Buffer
	if err := tx.Transaction.EncodeRLP(&b); err != nil {
		return err
	}
	if tx.Transaction.Type() != BlobTxType {
		_, err := w.Write(b.Bytes())
		return err
	}
	byt, _ := rlp.EncodeToBytes(tx.Transaction.inner.(*BlobTx))
	return rlp.Encode(w, &wrapper{
		TxType: BlobTxType,
		Inner: innerType2{
			BlobTx:      byt,
			Blobs:       tx.Blobs,
			Commitments: tx.Commitments,
			Proofs:      tx.Proofs,
		}})
	//_, err := w.Write(b.Bytes())
	//return err
}
