package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/savely-krasovsky/raiffeisen-retail-api"
)

var (
	username string
	password string
)

func main() {
	flag.StringVar(&username, "username", "", "Username")
	flag.StringVar(&password, "password", "", "Password")
	flag.Parse()

	c, err := raiffeisen.NewClient()
	if err != nil {
		panic(err)
	}

	if err := c.Login(); err != nil {
		panic(err)
	}

	if err := c.LoginFont(username, password); err != nil {
		panic(err)
	}

	accountBalances, err := c.AllAccountBalance()
	if err != nil {
		panic(err)
	}

	f, err := os.Create("accounts.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(accountBalances); err != nil {
		panic(err)
	}

	for _, account := range accountBalances {
		turnover, err := c.TransactionalAccountTurnover(account.ProductCoreID, account.Number, &raiffeisen.TransactionalAccountTurnoverFilter{
			CurrencyCodeNumeric: account.CurrencyCodeNumeric,
			FromDate:            "01.01.2024",
			ToDate:              "01.11.2024",
		})
		if err != nil {
			panic(err)
		}

		reserved, err := c.TransactionalAccountReservedFunds(account.Number)
		if err != nil {
			panic(err)
		}

		func() {
			f, err := os.Create(fmt.Sprintf("transactions_%s_%s.json", account.CurrencyCode, account.Number))
			if err != nil {
				panic(err)
			}
			defer f.Close()

			encoder := json.NewEncoder(f)
			encoder.SetIndent("", "  ")

			if err := encoder.Encode(slices.Concat(
				reserved.ToActualBudgetTransactions(),
				turnover.Transactions.ToActualBudgetTransactions(),
			)); err != nil {
				panic(err)
			}
		}()
	}
}
