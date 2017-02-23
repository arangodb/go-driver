package test

type UserDoc struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Account struct {
	ID   string   `json:"id"`
	User *UserDoc `json:"user"`
}
