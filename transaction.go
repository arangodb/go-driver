package driver

// TransactionOptions contains options that customize the transaction.
type TransactionOptions struct {
	// Transaction size limit in bytes. Honored by the RocksDB storage engine only.
	MaxTransactionSize int

	// An optional numeric value that can be used to set a timeout for waiting on collection
	// locks. If not specified, a default value will be used.
	// Setting lockTimeout to 0 will make ArangoDB not time out waiting for a lock.
	LockTimeout *int

	// An optional boolean flag that, if set, will force the transaction to write
	// all data to disk before returning.
	WaitForSync bool

	// Maximum number of operations after which an intermediate commit is performed
	// automatically. Honored by the RocksDB storage engine only.
	IntermediateCommitCount *int

	// Optional arguments passed to action.
	Params []interface{}

	// Maximum total size of operations after which an intermediate commit is
	// performed automatically. Honored by the RocksDB storage engine only.
	IntermediateCommitSize *int

	// Collections that the transaction reads from.
	ReadCollections []string

	// Collections that the transaction writes to.
	WriteCollections []string
}

type transactionRequest struct {
	MaxTransactionSize      int                           `json:"maxTransactionSize"`
	LockTimeout             *int                          `json:"lockTimeout,omitempty"`
	WaitForSync             bool                          `json:"waitForSync"`
	IntermediateCommitCount *int                          `json:"intermediateCommitCount,omitempty"`
	Params                  []interface{}                 `json:"params"`
	IntermediateCommitSize  *int                          `json:"intermediateCommitSize,omitempty"`
	Action                  string                        `json:"action"`
	Collections             transactionCollectionsRequest `json:"collections"`
}

type transactionCollectionsRequest struct {
	Read  []string `json:"read,omitempty"`
	Write []string `json:"write,omitempty"`
}

type transactionResponse struct {
	Result interface{} `json:"result"`
}
