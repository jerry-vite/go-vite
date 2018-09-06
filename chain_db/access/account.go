package access

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/helper"
	"github.com/vitelabs/go-vite/ledger"
)

type Account struct {
	db *leveldb.DB
}

func NewAccount(db *leveldb.DB) *Account {
	return &Account{
		db: db,
	}
}

func (accountAccess *Account) GetAccountByAddress(address *types.Address) (*ledger.Account, error) {
	keyAccountMeta, _ := helper.EncodeKey(database.DBKP_ACCOUNT, address.Bytes())

	data, dgErr := accountAccess.db.Get(keyAccountMeta, nil)
	if dgErr != nil {
		return nil, dgErr
	}
	account := &ledger.Account{}
	dsErr := account.DbDeSerialize(data)

	if dsErr != nil {
		return nil, dsErr
	}

	return account, nil
}
