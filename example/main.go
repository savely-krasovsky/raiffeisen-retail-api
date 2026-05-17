package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/savely-krasovsky/raiffeisen-retail-api"
)

var (
	username string
	password string
	from     string
	to       string
)

func main() {
	now := time.Now()
	flag.StringVar(&username, "username", "", "Username")
	flag.StringVar(&password, "password", "", "Password")
	flag.StringVar(&from, "from", "", "From date")
	flag.StringVar(&to, "to", now.Format("02.01.2006"), "To date")
	flag.Parse()

	c, err := raiffeisen.NewClient()
	if err != nil {
		panic(err)
	}

	if err := c.Login(); err != nil {
		panic(err)
	}

	loginResp, err := c.LoginFont(username, password)
	if err != nil {
		panic(err)
	}

	if loginResp.ForceSecondLogin {
		fmt.Println("2FA required — approve the push notification on your phone...")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		pushResult, err := c.RequestLoginPush(ctx, loginResp.Ticket, username)
		cancel()
		if err != nil {
			fmt.Println("2FA push failed")
			panic(err)
		}
		fmt.Printf("Push %s.\n", pushResult.Status)
		if err := c.LoginUPPush(loginResp.Ticket, pushResult.PushRequestContent, loginResp.GeneratedSessionID); err != nil {
			fmt.Println("can't finish 2FA login")
			panic(err)
		}
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
			FromDate:            from,
			ToDate:              to,
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
