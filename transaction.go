package driver

// TransactionOptions contains options that customize the transaction.
type TransactionOptions struct {
	// Transaction size limit in bytes. Honored by the RocksDB storage engine only.
	MaxTransactionSize int

	// An optional boolean flag that, if set, will force the transaction to write
	// all data to disk before returning.
	WaitForSync bool

	// Optional arguments passed to action.
	Params []string
}

type transactionRequest struct {
	MaxTransactionSize int                           `json:"maxTransactionSize"`
	WaitForSync        bool                          `json:"waitForSync"`
	Params             []string                      `json:"params"`
	Action             string                        `json:"action"`
	Collections        transactionCollectionsRequest `json:"collections"`
}

type transactionCollectionsRequest struct {
	Read  []string `json:"read,omitempty"`
	Write []string `json:"write,omitempty"`
}

type transactionResponse struct {
	Error        bool        `json:"error"`
	Code         int         `json:"code"`
	Result       interface{} `json:"result"`
	ErrorNum     int         `json:"errorNum"`
	ErrorMessage string      `json:"errorMessage"`
}
