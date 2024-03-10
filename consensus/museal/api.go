package MuSeal

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/consensus"
)

type API struct {
	chain  consensus.ChainHeaderReader
	MuSeal *MuSeal
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *API) EchoNumber(ctx context.Context, number uint64) (uint64, error) {
	fmt.Println("called echo number")
	return number, nil
}
