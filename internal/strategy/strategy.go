package strategy

import "github.com/AinurKadirov/crypto-parser/internal/domain"

// Strategy emits trading signals on each bar (Strategy pattern).
type Strategy interface {
	Name() string
	OnBar(bar domain.Bar, index int, ctx *Context) domain.Signal
}

// Context holds read-only backtest state for strategy decisions.
type Context struct {
	Balance  float64
	Position *domain.Position
	BarCount int
}

// HasPosition returns true if a position is open.
func (c *Context) HasPosition() bool {
	return c.Position != nil
}
