# Golang Raiffeisen Retail API client

Allows to export transactions from all accounts for Serbian branch of Raiffeisen Bank.
It could probably work for another branches, but not tests. Feel free to contibute.

## How to use

See an [example](/example/main.go). It will create `.json` files with transactions which are ready to be imported into
[Actual Budget](https://actualbudget.org) using [this script](/example/index.mjs).
