package driver

type transactionRequest struct {
	Collections transactionCollectionsRequest `json:"collections"`
	Action      string                        `json:"action"`
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
