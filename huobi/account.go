package huobi

import (
	"encoding/json"
	"fmt"
)

type (
	AccountType  string
	AccountState string
)

const (
	AccountTypeSpot        AccountType = "spot"
	AccountTypeMargin      AccountType = "margin"
	AccountTypeOTC         AccountType = "otc"
	AccountTypePoint       AccountType = "point"
	AccountTypeSuperMargin AccountType = "super-margin"
	AccountTypeInvestment  AccountType = "investment"
	AccountTypeBorrow      AccountType = "borrow"
)

const (
	AccountStateWorking AccountState = "working"
	AccountStateLock    AccountState = "lock"
)

type Account struct {
	Id          int64        `json:"id"`    // unique account id
	AccountType AccountType  `json:"type"`  // the type of this account (see above)
	State       AccountState `json:"state"` // account state (see above)
}

func (client *Client) Accounts() ([]Account, error) {
	type Response struct {
		Data []Account `json:"data"`
	}

	var (
		err  error
		body []byte
		resp Response
	)

	if body, err = client.get("/v1/account/accounts", nil, true); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (client *Client) Account(accountType AccountType, accountState AccountState) (*Account, error) {
	accounts, err := client.Accounts()
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if account.AccountType == accountType && account.State == accountState {
			return &account, nil
		}
	}
	return nil, fmt.Errorf("account not found")
}
