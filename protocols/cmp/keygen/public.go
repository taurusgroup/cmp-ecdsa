package keygen

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/taurusgroup/cmp-ecdsa/pkg/math/curve"
	"github.com/taurusgroup/cmp-ecdsa/pkg/paillier"
	"github.com/taurusgroup/cmp-ecdsa/pkg/pedersen"
)

type Public struct {
	// ECDSA public key, may be nil if the keygen has not run yet
	ECDSA *curve.Point

	// Paillier public key, may be nil if the keygen has not run yet
	Paillier *paillier.PublicKey

	// Pedersen auxiliary parameters, may be nil if the keygen has not run yet
	Pedersen *pedersen.Parameters
}

func (p *Public) Clone() *Public {
	return &Public{
		ECDSA:    curve.NewIdentityPoint().Set(p.ECDSA),
		Paillier: paillier.NewPublicKey(p.Paillier.N()),
		Pedersen: p.Pedersen.Clone(),
	}
}

// Validate returns an error if Public is invalid. Otherwise return nil.
func (p *Public) Validate() error {
	if p == nil || p.ECDSA == nil || p.Paillier == nil || p.Pedersen == nil {
		return errors.New("public: one or more field is empty")
	}

	// ECDSA is not identity
	if p.ECDSA.IsIdentity() {
		return errors.New("public: ECDSA public key is identity")
	}

	// Paillier check
	if err := p.Paillier.Validate(); err != nil {
		return fmt.Errorf("public: %w", err)
	}

	// Pedersen check
	if err := p.Pedersen.Validate(); err != nil {
		return fmt.Errorf("public: %w", err)
	}

	// Both N's are the same
	if p.Paillier.N().Cmp(p.Pedersen.N) != 0 {
		return errors.New("public: Pedersen and Paillier should share the same N")
	}

	return nil
}

var _ json.Marshaler = (*Public)(nil)
var _ json.Unmarshaler = (*Public)(nil)

type jsonParty struct {
	ECDSA *curve.Point `json:"ecdsa"`
	N     []byte       `json:"n"`
	S     []byte       `json:"s"`
	T     []byte       `json:"t"`
}

func (p Public) MarshalJSON() ([]byte, error) {
	x := jsonParty{
		ECDSA: p.ECDSA,
		N:     p.Pedersen.N.Bytes(),
		S:     p.Pedersen.S.Bytes(),
		T:     p.Pedersen.T.Bytes(),
	}
	return json.Marshal(x)
}

func (p *Public) UnmarshalJSON(bytes []byte) error {
	var x jsonParty
	err := json.Unmarshal(bytes, &x)
	if err != nil {
		return err
	}
	var n, s, t big.Int
	n.SetBytes(x.N)
	s.SetBytes(x.S)
	t.SetBytes(x.T)
	p.ECDSA = x.ECDSA
	p.Paillier = paillier.NewPublicKey(&n)
	p.Pedersen = &pedersen.Parameters{
		N: &n,
		S: &s,
		T: &t,
	}
	return p.Validate()
}
