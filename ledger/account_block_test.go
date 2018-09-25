package ledger

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"testing"
	"time"
)

func TestAccountBlock(t *testing.T) {
	accountAddress, _ := types.HexToAddress("vite_18068b64b49852e1c4dfbc304c4e606011e068836260bc9975")
	toAddress, _ := types.HexToAddress("vite_4827fbc6827797ac4d9e814affb34b4c5fa85d39bf96d105e7")
	prevHash, _ := types.HexToHash("37389229df60ccbd5ca2c96d48cc08a2d3f4ba5f4f24618681b119f19309ad17")
	snapshotTimestamp, _ := types.HexToHash("bc3002d5f874350854806ab7db3e00c656bc3240533e47704b2d49e1386b8ca7")
	var accountBlock = &AccountBlock{
		Meta: &AccountBlockMeta{
			Height: big.NewInt(19),
		},
		AccountAddress:    &accountAddress,
		To:                &toAddress,
		PrevHash:          &prevHash,
		Amount:            big.NewInt(1000000000000000000),
		TokenId:           &MockViteTokenId,
		Timestamp:         uint64(time.Now().Unix()),
		Data:              "",
		SnapshotTimestamp: &snapshotTimestamp,
		Nounce:            []byte{0, 0, 0, 0, 0},
		Difficulty:        []byte{0, 0, 0, 0, 0},
		FAmount:           big.NewInt(0),
	}

	fmt.Println(accountBlock.ComputeHash())
}
