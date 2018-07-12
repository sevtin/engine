package trading_engine

// import "github.com/ryszard/goskiplist/skiplist"
// import "trading_engine/skiplist"

// PricePoint holds the records for a particular price
type PricePoint struct {
	Entries []BookEntry
	// BuyBookEntries  []BookEntry
}

// NewPricePoints creates a new skiplist in which to hold all price points
func NewPricePoints() *SkipList {
	return NewCustomMap(cmp_func)
}

func cmp_func(l, r uint64) bool {
	return l < r
}
