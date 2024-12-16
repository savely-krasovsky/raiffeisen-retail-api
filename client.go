package raiffeisen

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/crypto/argon2"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const (
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0"
)

type Client interface {
	Login() error
	LoginFont(username, password string) error
	DashboardPreview() ([]*DashboardPreviewAccount, error)
	AllAccountBalance() ([]*AccountBalance, error)
	TransactionalAccountTurnover(productCoreID string, accountNumber string, filter *TransactionalAccountTurnoverFilter) (*TransactionalAccountTurnover, error)
	TransactionalAccountReservedFunds(accountNumber string) (ReservedTransactions, error)
}

type client struct {
	logger     *slog.Logger
	httpClient *http.Client
}

func NewClient() (Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	return &client{
		logger:     slog.Default(),
		httpClient: &http.Client{Jar: jar},
	}, nil
}

func (c *client) Login() error {
	req, _ := http.NewRequest(http.MethodGet, "https://rol.raiffeisenbank.rs/Retail/Home/Login", nil)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Error while request login page to fill cookie!", "err", err)
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		c.logger.Error("Error while reading response from login page!", "err", err)
		return err
	}

	return nil
}

func (c *client) LoginFont(username string, password string) error {
	usernameBytes := []byte(username)
	if len(usernameBytes) < 8 {
		usernameBytes = bytes.Join([][]byte{usernameBytes, bytes.Repeat([]byte{0}, 8-len(usernameBytes))}, nil)
	}

	saltedPassword := argon2.Key(
		[]byte(password),
		usernameBytes,
		3,
		4096,
		1,
		32,
	)

	request := struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		SessionID int    `json:"sessionID"`
	}{
		Username:  username,
		Password:  hex.EncodeToString(saltedPassword),
		SessionID: 1,
	}

	buf := new(bytes.Buffer)
	_ = json.NewEncoder(buf).Encode(request)

	req, _ := http.NewRequest(
		http.MethodPost,
		"https://rol.raiffeisenbank.rs/Retail/Protected/Services/RetailLoginService.svc/LoginFont",
		buf,
	)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Error while trying to login!", "err", err)
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return nil
}

func bomRemover(reader io.Reader) io.Reader {
	transformer := unicode.BOMOverride(encoding.Nop.NewDecoder())
	return transform.NewReader(reader, transformer)
}

func (c *client) DashboardPreview() ([]*DashboardPreviewAccount, error) {
	request := struct {
		GridName string `json:"gridName"`
	}{
		GridName: "RetailUserDashboardPreview",
	}

	buf := new(bytes.Buffer)
	_ = json.NewEncoder(buf).Encode(request)

	req, _ := http.NewRequest(
		http.MethodPost,
		"https://rol.raiffeisenbank.rs/Retail/Protected/Services/DataService.svc/GetDashboardsPreview",
		buf,
	)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Error while trying to get dashboard preview!", "err", err)
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response [][]string
	if err := json.NewDecoder(bomRemover(resp.Body)).Decode(&response); err != nil {
		c.logger.Error("Error while trying to decode dashboard preview response!", "err", err)
		return nil, err
	}

	accounts := make([]*DashboardPreviewAccount, len(response))
	for i, account := range response {
		accounts[i] = &DashboardPreviewAccount{
			Number:              account[5],
			CurrencyCode:        account[11],
			CurrencyCodeNumeric: account[10],
		}

		availableAmount, err := decimal.NewFromString(account[6])
		if err != nil {
			c.logger.Error("Cannot parse available amount!", "err", err)
			continue
		}
		accounts[i].AvailableAmount = availableAmount

		reservedAmount, err := decimal.NewFromString(account[4])
		if err != nil {
			c.logger.Error("Cannot parse reserved amount!", "err", err)
			continue
		}
		accounts[i].ReservedAmount = reservedAmount

		totalAmount, err := decimal.NewFromString(account[17])
		if err != nil {
			c.logger.Error("Cannot parse total amount!", "err", err)
			continue
		}
		accounts[i].TotalAmount = totalAmount
	}

	return accounts, nil
}

func (c *client) AllAccountBalance() ([]*AccountBalance, error) {
	request := struct {
		GridName string `json:"gridName"`
	}{
		GridName: "RetailAccountBalancePreviewFlat-L",
	}

	buf := new(bytes.Buffer)
	_ = json.NewEncoder(buf).Encode(request)

	req, _ := http.NewRequest(
		http.MethodPost,
		"https://rol.raiffeisenbank.rs/Retail/Protected/Services/DataService.svc/GetAllAccountBalance",
		buf,
	)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response [][]string
	if err := json.NewDecoder(bomRemover(resp.Body)).Decode(&response); err != nil {
		c.logger.Error("Error while trying to decode all account balance response!", "err", err)
		return nil, err
	}

	accounts := make([]*AccountBalance, len(response))
	for i, account := range response {
		accounts[i] = &AccountBalance{
			Number:              account[1],
			Description:         account[2],
			CurrencyCode:        account[3],
			CurrencyCodeNumeric: account[14],
			ProductCoreID:       account[13],
		}

		availableAmount, err := decimal.NewFromString(account[5])
		if err != nil {
			c.logger.Error("Cannot parse available amount!", "err", err)
			continue
		}
		accounts[i].AvailableAmount = availableAmount

		totalAmount, err := decimal.NewFromString(account[4])
		if err != nil {
			c.logger.Error("Cannot parse total amount!", "err", err)
			continue
		}
		accounts[i].TotalAmount = totalAmount

		lastTransactionAmount, err := decimal.NewFromString(account[6])
		if err != nil {
			c.logger.Error("Cannot parse last transaction amount!", "err", err)
			continue
		}
		accounts[i].LastTransactionAmount = lastTransactionAmount

		d, err := time.Parse("02.01.2006 15:04:05", account[7])
		if err != nil {
			c.logger.Error("Cannot parse last transaction date!", "err", err)
			continue
		}
		accounts[i].LastTransactionDate = d

	}

	return accounts, nil
}

func (c *client) TransactionalAccountTurnover(productCoreID string, accountNumber string, filter *TransactionalAccountTurnoverFilter) (*TransactionalAccountTurnover, error) {
	gridName := "RetailAccountTurnoverTransactionPreviewMasterDetail-S"
	// You can try this, but it lacks card number and cannot be used for foreign currency accounts
	// gridName := "RetailAccountTurnoverTransactionDomesticPreviewMasterDetail-S"

	request := &TransactionalAccountTurnoverRequest{
		AccountNumber: accountNumber,
		FilterParam:   filter,
		GridName:      gridName,
		ProductCoreID: productCoreID,
	}

	buf := new(bytes.Buffer)
	_ = json.NewEncoder(buf).Encode(request)

	req, _ := http.NewRequest(
		http.MethodPost,
		"https://rol.raiffeisenbank.rs/Retail/Protected/Services/DataService.svc/GetTransactionalAccountTurnover",
		buf,
	)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Error while getting transactional account turnover!", "err", err)
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response [][][]any
	if err := json.NewDecoder(bomRemover(resp.Body)).Decode(&response); err != nil {
		c.logger.Error("Error while trying to decode transactional account turnover response!", "err", err)
		return nil, err
	}

	if len(response) == 0 {
		return &TransactionalAccountTurnover{
			Transactions: make(Transactions, 0),
		}, nil
	}

	transactions := make([]*Transaction, len(response[0][1]))
	for i, transaction := range response[0][1] {
		transactions[i] = &Transaction{
			CurrencyCodeNumeric: transaction.([]any)[1].(string),
			CurrencyCode:        transaction.([]any)[2].(string),
			Place:               transaction.([]any)[6].(string),
			Reference:           transaction.([]any)[7].(string),
			Description:         transaction.([]any)[11].(string),
			ID:                  transaction.([]any)[12].(string),
			Type:                TransactionType(transaction.([]any)[13].(string)),
		}

		creditAmount, err := decimal.NewFromString(transaction.([]any)[8].(string))
		if err != nil {
			c.logger.Error("Cannot parse credit amount for transaction", "err", err)
			continue
		}
		debigAmount, err := decimal.NewFromString(transaction.([]any)[9].(string))
		if err != nil {
			c.logger.Error("Cannot parse debit amount for transaction", "err", err)
			continue
		}

		if !creditAmount.IsZero() {
			transactions[i].Amount = creditAmount.Neg()
		}
		if !debigAmount.IsZero() {
			transactions[i].Amount = debigAmount
		}

		d, err := time.Parse("02.01.2006 15:04:05", transaction.([]any)[3].(string))
		if err != nil {
			c.logger.Error("Cannot parse transaction date!", "err", err)
			continue
		}
		transactions[i].Date = d
	}

	return &TransactionalAccountTurnover{Transactions: transactions}, nil
}

func (c *client) TransactionalAccountReservedFunds(accountNumber string) (ReservedTransactions, error) {
	gridName := "RetailAccountReservedFundsPreviewFlat"

	request := &TransactionalAccountReservedFundsRequest{
		AccountNumber: accountNumber,
		GridName:      gridName,
	}

	buf := new(bytes.Buffer)
	_ = json.NewEncoder(buf).Encode(request)

	req, _ := http.NewRequest(
		http.MethodPost,
		"https://rol.raiffeisenbank.rs/Retail/Protected/Services/DataService.svc/GetTransactionalAccountReservedFunds",
		buf,
	)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Error while getting transactional account reserved funds!", "err", err)
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response [][]string
	if err := json.NewDecoder(bomRemover(resp.Body)).Decode(&response); err != nil {
		c.logger.Error("Error while trying to decode transactional account reserved funds response!", "err", err)
		return nil, err
	}

	transactions := make([]*ReservedTransaction, len(response))
	for i, transaction := range response {
		transactions[i] = &ReservedTransaction{
			CurrencyCodeNumeric: transaction[5],
			CurrencyCode:        transaction[4],
			Place:               transaction[2],
		}

		amount, err := decimal.NewFromString(transaction[3])
		if err != nil {
			c.logger.Error("Cannot parse amount for transaction", "err", err)
			continue
		}
		transactions[i].Amount = amount.Neg()

		d, err := time.Parse("02.01.2006 15:04:05", transaction[1])
		if err != nil {
			c.logger.Error("Cannot parse transaction date!", "err", err)
			continue
		}
		transactions[i].Date = d
	}

	return transactions, nil
}
