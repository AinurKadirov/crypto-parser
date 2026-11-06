package strategy

import "github.com/AinurKadirov/crypto-parser/internal/domain"

// BuyAndHold buys once on the first bar and holds until the end.
type BuyAndHold struct{}

func (BuyAndHold) Name() string { return "buy_and_hold" }

func (BuyAndHold) OnBar(_ domain.Bar, index int, ctx *Context) domain.Signal {
	if index == 0 && !ctx.HasPosition() {
		return domain.Signal{Action: domain.ActionBuy, SizePct: 100, Reason: "buy_and_hold entry"}
	}
	return domain.Signal{Action: domain.ActionHold}
}
