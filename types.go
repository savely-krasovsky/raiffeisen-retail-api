package raiffeisen

import (
	"time"

	"github.com/shopspring/decimal"
)

type TransactionalAccountTurnoverRequest struct {
	AccountNumber string                              `json:"accountNumber"`
	FilterParam   *TransactionalAccountTurnoverFilter `json:"filterParam"`
	GridName      string                              `json:"gridName"`
	ProductCoreID string                              `json:"productCoreID"`
}

type TransactionalAccountReservedFundsRequest struct {
	AccountNumber string `json:"accountNumber"`
	GridName      string `json:"gridName"`
}

type TransactionalAccountTurnoverFilter struct {
	CurrencyCodeNumeric string `json:"CurrencyCodeNumeric"`
	FromDate            string `json:"FromDate"`
	ToDate              string `json:"ToDate"`
	ItemType            string `json:"ItemType"`
	ItemCount           string `json:"ItemCount"`
	FromAmount          string `json:"FromAmount"`
	ToAmount            string `json:"ToAmount"`
	PaymentPurpose      string `json:"PaymentPurpose"`
}

type DashboardPreviewAccount struct {
	Number              string          `json:"number"`
	CurrencyCode        string          `json:"currency_code"`
	CurrencyCodeNumeric string          `json:"currency_code_numeric"`
	TotalAmount         decimal.Decimal `json:"total_amount"`
	AvailableAmount     decimal.Decimal `json:"available_amount"`
	ReservedAmount      decimal.Decimal `json:"reserved_amount"`
}

type AccountBalance struct {
	Number                string          `json:"number"`
	Description           string          `json:"description"`
	CurrencyCode          string          `json:"currency_code"`
	CurrencyCodeNumeric   string          `json:"currency_code_numeric"`
	TotalAmount           decimal.Decimal `json:"total_amount"`
	AvailableAmount       decimal.Decimal `json:"available_amount"`
	LastTransactionAmount decimal.Decimal `json:"last_transaction_amount"`
	LastTransactionDate   time.Time       `json:"last_transaction_date"`
	ProductCoreID         string          `json:"product_core_id"`
}

type TransactionalAccountTurnover struct {
	Transactions Transactions `json:"transactions"`
}

type Transactions []*Transaction

func (tt Transactions) ToActualBudgetTransactions() []*ActualBudgetTransaction {
	transactions := make([]*ActualBudgetTransaction, len(tt))
	for i, t := range tt {
		transactions[i] = t.ToActualBudgetTransaction()
	}

	return transactions
}

type Transaction struct {
	CurrencyCodeNumeric string          `json:"currency_code_numeric"`
	CurrencyCode        string          `json:"currency_code"`
	Date                time.Time       `json:"date"`
	Place               string          `json:"place"`
	Reference           string          `json:"reference"`
	Amount              decimal.Decimal `json:"amount"`
	Description         string          `json:"description"`
	ID                  string          `json:"id"`
	Type                TransactionType `json:"type"`
}

type TransactionType string

const (

	// POSTransactionType represents a point-of-sale credit or debit transaction type.
	POSTransactionType TransactionType = "POS"
	// OtherTransactionType represents a transaction type that does not fall into predefined categories,
	// could be both credit or debit.
	OtherTransactionType TransactionType = "Other"
	// ExchBuyTransactionType represents a credit transaction type for exchange buy operations.
	ExchBuyTransactionType TransactionType = "ExchBuy"
	// ExchSellTransactionType represents a debit transaction type for exchange sell operations.
	ExchSellTransactionType TransactionType = "ExchSell"
	// IncomeTransactionType represents a debit type of transaction associated with receiving income.
	IncomeTransactionType TransactionType = "Income"
	// IncomeCashTransactionType represents a debit transaction type indicating income received in cash.
	IncomeCashTransactionType TransactionType = "IncomeCash"
)

func (t *Transaction) ToActualBudgetTransaction() *ActualBudgetTransaction {
	return &ActualBudgetTransaction{
		Date:          t.Date,
		Amount:        t.Amount.Mul(decimal.NewFromInt(100)).IntPart(),
		PayeeName:     t.Place,
		ImportedPayee: t.Place,
		Notes:         t.Description,
		ImportedID:    t.ID,
		Cleared:       true,
	}
}

type ActualBudgetTransaction struct {
	Date          time.Time `json:"date"`
	Amount        int64     `json:"amount"`
	PayeeName     string    `json:"payee_name"`
	ImportedPayee string    `json:"imported_payee"`
	Notes         string    `json:"notes,omitempty"`
	ImportedID    string    `json:"imported_id,omitempty"`
	Cleared       bool      `json:"cleared"`
}

type ReservedTransaction struct {
	Date                time.Time       `json:"date"`
	Place               string          `json:"place"`
	Amount              decimal.Decimal `json:"amount"`
	CurrencyCodeNumeric string          `json:"currency_code_numeric"`
	CurrencyCode        string          `json:"currency_code"`
}

func (t *ReservedTransaction) ToActualBudgetTransaction() *ActualBudgetTransaction {
	return &ActualBudgetTransaction{
		Date:          t.Date,
		Amount:        t.Amount.Mul(decimal.NewFromInt(100)).IntPart(),
		PayeeName:     t.Place,
		ImportedPayee: t.Place,
		Cleared:       false,
	}
}

type ReservedTransactions []*ReservedTransaction

func (tt ReservedTransactions) ToActualBudgetTransactions() []*ActualBudgetTransaction {
	transactions := make([]*ActualBudgetTransaction, len(tt))
	for i, t := range tt {
		transactions[i] = t.ToActualBudgetTransaction()
	}

	return transactions
}
