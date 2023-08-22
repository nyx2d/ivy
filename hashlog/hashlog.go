package hashlog

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
)

type HashLog struct {
}

type Entry struct {
	ID string // sha256(sha256(EntryData), signature)
	EntryData
	Signature string // base64(sign(sha256(EntryData), signer_private_key))
}

type EntryData struct {
	Previous string // ID of previous entry, empty is null
	Sequence uint64 // hashed as string representation
	Content  []byte // hashed directly
	Signer   string // base64(signer_public_key)
}

func NewEntry(previous string, sequence uint64, content []byte, signingKey ed25519.PrivateKey) (*Entry, error) {
	signer := base64.StdEncoding.EncodeToString(signingKey.Public().(ed25519.PublicKey))
	data := EntryData{
		Previous: previous,
		Sequence: sequence,
		Content:  content,
		Signer:   signer,
	}
	dataHash, err := data.Hash()
	if err != nil {
		return nil, err
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(signingKey, dataHash))

	// generate ID
	h := sha256.New()
	_, err = h.Write(dataHash)
	if err != nil {
		return nil, err
	}
	_, err = h.Write([]byte(signature))
	if err != nil {
		return nil, err
	}
	id := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return &Entry{
		ID:        id,
		EntryData: data,
		Signature: signature,
	}, nil
}

func (e *Entry) Verify() bool {
	rawSigner, err := base64.StdEncoding.DecodeString(e.Signer)
	if err != nil {
		return false
	}
	signer := ed25519.PublicKey(rawSigner)

	rawSignature, err := base64.StdEncoding.DecodeString(e.Signature)
	if err != nil {
		return false
	}

	dataHash, err := e.EntryData.Hash()
	if err != nil {
		return false
	}

	// confirm ID is correct
	h := sha256.New()
	_, err = h.Write(dataHash)
	if err != nil {
		return false
	}
	_, err = h.Write([]byte(e.Signature))
	if err != nil {
		return false
	}
	id := base64.StdEncoding.EncodeToString(h.Sum(nil))
	if id != e.ID {
		return false
	}

	return ed25519.Verify(signer, dataHash, rawSignature)
}

func (d EntryData) Hash() ([]byte, error) {
	h := sha256.New()
	seq := strconv.FormatUint(d.Sequence, 10)

	_, err := h.Write([]byte(d.Previous))
	if err != nil {
		return nil, err
	}
	_, err = h.Write([]byte(seq))
	if err != nil {
		return nil, err
	}
	_, err = h.Write(d.Content)
	if err != nil {
		return nil, err
	}
	_, err = h.Write([]byte(d.Signer))
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
