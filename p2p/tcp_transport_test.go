package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTCPTransport(t *testing.T) {
	listenAddr := ":4000"
	tr  := NewTCPTransport(listenAddr)

	assert.Equal(t, tr.listenAddr, listenAddr)

	// Start server
	assert.Nil(t, tr.ListenAndAccept())
}


