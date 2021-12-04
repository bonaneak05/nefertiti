package huobi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type (
	OrderType  string
	OrderState string
)

const (
	OrderTypeBuyMarket        OrderType = "buy-market"
	OrderTypeSellMarket       OrderType = "sell-market"
	OrderTypeBuyLimit         OrderType = "buy-limit"
	OrderTypeSellLimit        OrderType = "sell-limit"
	OrderTypeBuyIOC           OrderType = "buy-ioc"  // IOC stands for "immediately or cancel", it means the order will be canceled if it couldn't be matched. If the order is partially traded, the remaining part will be canceled.
	OrderTypeSellIOC          OrderType = "sell-ioc" // IOC stands for "immediately or cancel", it means the order will be canceled if it couldn't be matched. If the order is partially traded, the remaining part will be canceled.
	OrderTypeBuyLimitMaker    OrderType = "buy-limit-maker"
	OrderTypeSellLimitMaker   OrderType = "sell-limit-maker"
	OrderTypeBuyStopLimit     OrderType = "buy-stop-limit"
	OrderTypeSellStopLimit    OrderType = "sell-stop-limit"
	OrderTypeBuyLimitFOK      OrderType = "buy-limit-fok"  // FOK stands for "fill or kill", it means the order will be cancelled if it couldn't be fully matched. Even if the order can be partially filled, the entire order will be cancelled.
	OrderTypeSellLimitFOK     OrderType = "sell-limit-fok" // FOK stands for "fill or kill", it means the order will be cancelled if it couldn't be fully matched. Even if the order can be partially filled, the entire order will be cancelled.
	OrderTypeBuyStopLimitFOK  OrderType = "buy-stop-limit-fok"
	OrderTypeSellStopLimitFOK OrderType = "sell-stop-limit-fok"
)

const (
	OrderStateCreated         OrderState = "created"          // The order is created, and not in the matching queue yet.
	OrderStateSubmitted       OrderState = "submitted"        // The order is submitted, and already in the matching queue, waiting for deal.
	OrderStatePartialFilled   OrderState = "partial-filled"   // The order is already in the matching queue and partially traded, and is waiting for further matching and trade.
	OrderStateFilled          OrderState = "filled"           // The order is already traded and not in the matching queue any more.
	OrderStatePartialCanceled OrderState = "partial-canceled" // The order is not in the matching queue any more. The status is transferred from 'partial-filled', the order is partially trade, but remaining is canceled.
	OrderStateCanceling       OrderState = "canceling"        // The order is under canceling, but haven't been removed from matching queue yet.
	OrderStateCanceled        OrderState = "canceled"         // The order is not in the matching queue any more, and completely canceled. There is no trade associated with this order.
)

type Order struct {
	Symbol           string     `json:"symbol"` // the trading symbol to trade, e.g. btcusdt, bccbtc
	Amount           float64    `json:"amount,string"`
	Price            float64    `json:"price,string"` // the limit price of limit order
	CreatedAt        int64      `json:"created-at"`   // the timestamp in milliseconds when the order was created
	AccountId        int64      `json:"account-id"`
	ClientOrderId    string     `json:"client-order-id"`           // the identity defined by the client
	FilledAmount     float64    `json:"filled-amount,string"`      // the amount which has been filled
	FilledCashAmount float64    `json:"filled-cash-amount,string"` // the filled total in quote currency
	FilledFees       float64    `json:"filled-fees,string"`        // transaction fee
	Id               int64      `json:"id"`                        // the unique identity for the order
	State            OrderState `json:"state"`                     // order status (see above)
	OrderType        OrderType  `json:"type"`                      // order type (see above)
}

func (order *Order) Buy() bool {
	return order.OrderType == OrderTypeBuyMarket ||
		order.OrderType == OrderTypeBuyLimit ||
		order.OrderType == OrderTypeBuyIOC ||
		order.OrderType == OrderTypeBuyLimitMaker ||
		order.OrderType == OrderTypeBuyStopLimit ||
		order.OrderType == OrderTypeBuyLimitFOK ||
		order.OrderType == OrderTypeBuyStopLimitFOK
}

func (order *Order) Sell() bool {
	return !order.Buy()
}

func (order *Order) GetCreatedAt() time.Time {
	if order.CreatedAt == 0 {
		return time.Now().UTC()
	} else {
		return time.Unix((order.CreatedAt / 1000), 0)
	}
}

func (client *Client) PlaceOrder(symbol string, orderType OrderType, amount, price float64, metadata string) ([]byte, error) {
	var (
		err     error
		account *Account
	)

	if account, err = client.Account(AccountTypeSpot, AccountStateWorking); err != nil {
		return nil, err
	}

	type (
		Request struct {
			AccountId     string `json:"account-id"`
			Symbol        string `json:"symbol"`
			OrderType     string `json:"type"`
			Amount        string `json:"amount"`
			Price         string `json:"price,omitempty"`
			ClientOrderId string `json:"client-order-id,omitempty"`
		}
		Response struct {
			Data string `json:"data"`
		}
	)

	request := Request{
		AccountId: strconv.FormatInt(account.Id, 10),
		Symbol:    symbol,
		OrderType: string(orderType),
		Amount:    strconv.FormatFloat(amount, 'f', -1, 64),
		Price: func() string {
			if price > 0 {
				return strconv.FormatFloat(price, 'f', -1, 64)
			}
			return ""
		}(),
		ClientOrderId: metadata,
	}

	var (
		body []byte
		resp Response
	)

	if body, err = client.post("/v1/order/orders/place", request); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return []byte(resp.Data), nil
}

func (client *Client) OpenOrders(symbol string) ([]Order, error) {
	type Response struct {
		Data []Order `json:"data"`
	}

	var (
		err  error
		body []byte
		resp Response
	)

	params := url.Values{}
	params.Add("symbol", symbol)

	if body, err = client.get("/v1/order/openOrders", params, true); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (client *Client) CancelOrder(orderId int64) error {
	_, err := client.post(fmt.Sprintf("/v1/order/orders/%d/submitcancel", orderId), nil)
	return err
}
