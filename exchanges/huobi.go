//lint:file-ignore ST1006 receiver name should be a reflection of its identity; don't use generic names such as "this" or "self"
package exchanges

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/svanas/nefertiti/aggregation"
	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/flag"
	exchange "github.com/svanas/nefertiti/huobi"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/multiplier"
	"github.com/svanas/nefertiti/notify"
	"github.com/svanas/nefertiti/precision"
)

type Huobi struct {
	*model.ExchangeInfo
	symbols []exchange.Symbol
}

func (self *Huobi) getBaseURL(sandbox bool) string {
	return self.ExchangeInfo.REST.URI
}

func (self *Huobi) getSymbols(client *exchange.Client, cached bool) ([]exchange.Symbol, error) {
	if self.symbols == nil || !cached {
		var err error
		if self.symbols, err = client.Symbols(); err != nil {
			return nil, errors.Wrap(err, 1)
		}
	}
	return self.symbols, nil
}

func (self *Huobi) getSymbol(client *exchange.Client, market string, cached bool) (*exchange.Symbol, error) {
	symbols, err := self.getSymbols(client, cached)
	if err != nil {
		return nil, err
	}

	for _, symbol := range symbols {
		if symbol.Symbol == market {
			return &symbol, nil
		}
	}

	return nil, errors.Errorf("symbol %v does not exist", market)
}

func (self *Huobi) error(err error, level int64, service model.Notify) {
	pc, file, line, _ := runtime.Caller(1)
	prefix := errors.FormatCaller(pc, file, line)

	msg := fmt.Sprintf("%s %v", prefix, err)
	_, ok := err.(*errors.Error)
	if ok && flag.Debug() {
		log.Printf("[ERROR] %s", err.(*errors.Error).ErrorStack(prefix, ""))
	} else {
		log.Printf("[ERROR] %s", msg)
	}

	if service != nil {
		if notify.CanSend(level, notify.ERROR) {
			err := service.SendMessage(msg, "Huobi - ERROR", model.ONCE_PER_MINUTE)
			if err != nil {
				log.Printf("[ERROR] %v", err)
			}
		}
	}
}

func (self *Huobi) GetInfo() *model.ExchangeInfo {
	return self.ExchangeInfo
}

func (self *Huobi) GetClient(permission model.Permission, sandbox bool) (interface{}, error) {
	if permission != model.PRIVATE {
		return exchange.New(self.getBaseURL(sandbox), "", ""), nil
	}

	var (
		err       error
		apiKey    string
		apiSecret string
	)
	if apiKey, apiSecret, err = promptForApiKeys("Huobi"); err != nil {
		return nil, err
	}

	return exchange.New(self.getBaseURL(sandbox), apiKey, apiSecret), nil
}

func (self *Huobi) GetMarkets(cached, sandbox bool, blacklist []string) ([]model.Market, error) {
	var out []model.Market

	symbols, err := self.getSymbols(exchange.New(self.getBaseURL(sandbox), "", ""), cached)
	if err != nil {
		return nil, err
	}

	for _, symbol := range symbols {
		if symbol.Online() && symbol.Enabled() && func() bool {
			for _, ignore := range blacklist {
				if strings.EqualFold(symbol.Symbol, ignore) {
					return false
				}
			}
			return true
		}() {
			out = append(out, model.Market{
				Name:  symbol.Symbol,
				Base:  symbol.BaseCurrency,
				Quote: symbol.QuoteCurrency,
			})
		}
	}

	return out, nil
}

func (self *Huobi) FormatMarket(base, quote string) string {
	return strings.ToLower(base + quote)
}

func (self *Huobi) Sell(
	strategy model.Strategy,
	hold, earn model.Markets,
	sandbox, tweet, debug bool,
	success model.OnSuccess,
) error {
	if strategy != model.STRATEGY_STANDARD {
		return errors.New("strategy not implemented")
	}

	// apiKey, apiSecret, err := promptForApiKeys("Huobi")
	// if err != nil {
	// 	return err
	// }

	// service, err := notify.New().Init(flag.Interactive(), true)
	// if err != nil {
	// 	return err
	// }

	return errors.New("Not implemented")
}

func (self *Huobi) Order(
	client interface{},
	side model.OrderSide,
	market string,
	size float64,
	price float64,
	kind model.OrderType,
	metadata string,
) (oid []byte, raw []byte, err error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return nil, nil, errors.New("invalid argument: client")
	}

	if oid, err = huobiClient.PlaceOrder(market, func() exchange.OrderType {
		if side == model.BUY {
			if kind == model.LIMIT {
				return exchange.OrderTypeBuyLimit
			} else {
				return exchange.OrderTypeBuyMarket
			}
		} else {
			if kind == model.LIMIT {
				return exchange.OrderTypeSellLimit
			} else {
				return exchange.OrderTypeSellMarket
			}
		}
	}(), size, price, metadata); err != nil {
		return nil, nil, errors.Wrap(err, 1)
	}

	return oid, nil, nil
}

func (self *Huobi) StopLoss(client interface{}, market string, size float64, price float64, kind model.OrderType, metadata string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (self *Huobi) OCO(client interface{}, market string, size float64, price, stop float64, metadata string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (self *Huobi) GetClosed(client interface{}, market string) (model.Orders, error) {
	return nil, errors.New("Not implemented")
}

func (self *Huobi) GetOpened(client interface{}, market string) (model.Orders, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var (
		err    error
		orders []exchange.Order
		output model.Orders
	)

	if orders, err = huobiClient.OpenOrders(market); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	for _, order := range orders {
		output = append(output, model.Order{
			Side: func() model.OrderSide {
				if order.Sell() {
					return model.SELL
				}
				return model.BUY
			}(),
			Market:    market,
			Size:      order.Amount,
			Price:     order.Price,
			CreatedAt: order.GetCreatedAt(),
		})
	}

	return output, nil
}

func (self *Huobi) GetBook(client interface{}, market string, side model.BookSide) (interface{}, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	book, err := huobiClient.OrderBook(market)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return func() []exchange.BookEntry {
		if side == model.BOOK_SIDE_ASKS {
			return book.Asks
		} else {
			return book.Bids
		}
	}(), nil
}

func (self *Huobi) Aggregate(client, book interface{}, market string, agg float64) (model.Book, error) {
	bids, ok := book.([]exchange.BookEntry)
	if !ok {
		return nil, errors.New("invalid argument: book")
	}

	prec, err := self.GetPricePrec(client, market)
	if err != nil {
		return nil, err
	}

	var out model.Book
	for _, e := range bids {
		price := precision.Round(aggregation.Round(e.Price(), agg), prec)
		entry := out.EntryByPrice(price)
		if entry != nil {
			entry.Size = entry.Size + e.Size()
		} else {
			entry = &model.Buy{
				Market: market,
				Price:  price,
				Size:   e.Size(),
			}
			out = append(out, *entry)
		}
	}

	return out, nil
}

func (self *Huobi) GetTicker(client interface{}, market string) (float64, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}

	ticker, err := huobiClient.Ticker(market)
	if err != nil {
		return 0, errors.Wrap(err, 1)
	}

	return ticker.Price, nil
}

func (self *Huobi) Get24h(client interface{}, market string) (*model.Stats, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var (
		err error
		sum *exchange.Summary
	)
	if sum, err = huobiClient.Summary(market); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return &model.Stats{
		Market: market,
		High:   sum.High,
		Low:    sum.Low,
		BtcVolume: func(sum *exchange.Summary) float64 {
			symbols, err := self.getSymbols(huobiClient, true)
			if err == nil {
				for _, symbol := range symbols {
					if symbol.Symbol == market {
						if strings.EqualFold(symbol.BaseCurrency, model.BTC) {
							return sum.Volume
						}
						tick, err := huobiClient.Ticker(self.FormatMarket(symbol.BaseCurrency, model.BTC))
						if err == nil {
							return sum.Volume * tick.Price
						}
					}
				}
			}
			return 0
		}(sum),
	}, nil
}

func (self *Huobi) GetPricePrec(client interface{}, market string) (int, error) {
	return 8, errors.New("Not implemented")
}

func (self *Huobi) GetSizePrec(client interface{}, market string) (int, error) {
	return 0, errors.New("Not implemented")
}

func (self *Huobi) GetMaxSize(client interface{}, base, quote string, hold, earn bool, def float64, mult multiplier.Mult) float64 {
	return model.GetSizeMax(hold, earn, def, mult, func() int {
		prec, err := self.GetSizePrec(client, self.FormatMarket(base, quote))
		if err != nil {
			return 0
		}
		return prec
	})
}

func (self *Huobi) Cancel(client interface{}, market string, side model.OrderSide) error {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	orders, err := huobiClient.OpenOrders(market)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	for _, order := range orders {
		if ((side == model.BUY) && order.Buy()) || ((side == model.SELL) && order.Sell()) {
			if err := huobiClient.CancelOrder(order.Id); err != nil {
				return errors.Wrap(err, 1)
			}
		}
	}

	return nil
}

func (self *Huobi) Buy(client interface{}, cancel bool, market string, calls model.Calls, deviation float64, kind model.OrderType) error {
	return errors.New("Not implemented")
}

func (self *Huobi) IsLeveragedToken(name string) bool {
	return false
}

func (self *Huobi) HasAlgoOrder(client interface{}, market string) (bool, error) {
	return false, nil
}

func newHuobi() model.Exchange {
	return &Huobi{
		ExchangeInfo: &model.ExchangeInfo{
			Code: "HUBI",
			Name: "Huobi",
			URL:  "https://www.huobi.com",
			REST: model.Endpoint{
				URI: "https://api.huobi.pro",
			},
			WebSocket: model.Endpoint{
				URI: "wss://api.huobi.pro/ws",
			},
			Country: "Seychelles",
		},
	}
}
