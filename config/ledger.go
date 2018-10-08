package config

type Ledger struct {
	IsDownload         bool `json:"IsDownload"`
	ResendGenesisBlock bool `json:"ResendGenesisBlock"`
}
