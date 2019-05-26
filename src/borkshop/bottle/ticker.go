package bottle

// Ticker writes a new generation from a previous.
type Ticker interface {
	Tick(next, prev *Generation)
}

// Tickers is a list of tickers to tick for each tick.
type Tickers []Ticker

var _ Ticker = Tickers(nil)

// Tick ticks all the tickers.
func (tickers Tickers) Tick(next, prev *Generation) {
	for i := 0; i < len(tickers); i++ {
		tickers[i].Tick(next, prev)
	}
}
