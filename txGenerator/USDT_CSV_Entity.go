package txGenerator

type USDTCSVEntity struct {
	From  string `csv:"from_address"`
	To    string `csv:"to_address"`
	Hash  string `csv:"transaction_hash"`
	Value string `csv:"value"`
}
