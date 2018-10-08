package main

import (
	"github.com/vitelabs/go-vite/ledger"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"github.com/vitelabs/go-vite/vite"
	"math/big"
	"time"
)

func resender(vnode *vite.Vite) {
	var hasSended = big.NewInt(0)
	go func() {
		blocks, _ := vnode.Ledger().Ac().GetBlocksByAccAddr(&ledger.GenesisAccount, 0, 1, 10)
		for i := len(blocks) - 1; i >= 0; i-- {
			block := blocks[i]
			if hasSended.Cmp(block.Meta.Height) < 0 {
				err := vnode.Pm().SendMsg(nil, &protoTypes.Msg{
					Code:    protoTypes.AccountBlocksMsgCode,
					Payload: &protoTypes.AccountBlocksMsg{block},
				})

				if err == nil {
					hasSended = block.Meta.Height
				} else {
					break
				}
			} else {
				break
			}

		}

		time.Sleep(time.Second * 3)
	}()
}
