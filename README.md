# Golang Raiffeisen Retail API client

Allows to export transactions from all accounts for Serbian branch of Raiffeisen Bank.
It could probably work for another branches, but I tested it only with Serbian retail account.
Feel free to contibute.

## How to use

See an [example](/example/main.go). It will create `.json` files with transactions which are ready to be imported into
[Actual Budget](https://actualbudget.org) using [this script](/example/index.mjs).

```bash
go build -o ./exporter ./example/main.go
./exporter -username YOUR_USERNAME -password YOUR_PASSWORD -from 01.01.2024 # optionally use -to 
```

```bash
mv transactions_RSD_XXXXXXXXXXXXXXXXX.json transactions.json
node ./example/index.mjs
```
