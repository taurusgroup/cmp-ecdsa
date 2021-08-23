package cmp

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/taurusgroup/multi-party-sig/internal/test"
	"github.com/taurusgroup/multi-party-sig/pkg/ecdsa"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/pool"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
)

func do(t *testing.T, id party.ID, ids []party.ID, threshold int, message []byte, pl *pool.Pool, n *test.Network, wg *sync.WaitGroup) {
	defer wg.Done()
	h, err := protocol.NewHandler(StartKeygen(pl, curve.Secp256k1{}, ids, threshold, id))
	require.NoError(t, err)
	require.NoError(t, test.HandlerLoop(id, h, n))
	r, err := h.Result()
	require.NoError(t, err)
	require.IsType(t, &Config{}, r)
	c := r.(*Config)

	h, err = protocol.NewHandler(StartRefresh(pl, c))
	require.NoError(t, err)
	require.NoError(t, test.HandlerLoop(c.ID, h, n))

	r, err = h.Result()
	require.NoError(t, err)
	require.IsType(t, &Config{}, r)
	c = r.(*Config)

	h, err = protocol.NewHandler(StartSign(pl, c, ids, message))
	require.NoError(t, err)
	require.NoError(t, test.HandlerLoop(c.ID, h, n))

	signResult, err := h.Result()
	require.NoError(t, err)
	require.IsType(t, &ecdsa.Signature{}, signResult)
	signature := signResult.(*ecdsa.Signature)
	assert.True(t, signature.Verify(c.PublicPoint(), message))

	h, err = protocol.NewHandler(StartPresign(pl, c, ids))
	require.NoError(t, err)

	require.NoError(t, test.HandlerLoop(c.ID, h, n))

	signResult, err = h.Result()
	require.NoError(t, err)
	require.IsType(t, &ecdsa.PreSignature{}, signResult)
	preSignature := signResult.(*ecdsa.PreSignature)
	assert.NoError(t, preSignature.Validate())

	h, err = protocol.NewHandler(StartPresignOnline(c, preSignature, message))
	require.NoError(t, err)
	require.NoError(t, test.HandlerLoop(c.ID, h, n))

	signResult, err = h.Result()
	require.NoError(t, err)
	require.IsType(t, &ecdsa.Signature{}, signResult)
	signature = signResult.(*ecdsa.Signature)
	assert.True(t, signature.Verify(c.PublicPoint(), message))
}

func TestCMP(t *testing.T) {
	N := 5
	T := N - 1
	message := []byte("hello")

	pl := pool.NewPool(0)
	defer pl.TearDown()

	partyIDs := test.PartyIDs(N)

	n := test.NewNetwork(partyIDs)

	var wg sync.WaitGroup
	wg.Add(N)
	for _, id := range partyIDs {
		go do(t, id, partyIDs, T, message, pl, n, &wg)
	}
	wg.Wait()
}