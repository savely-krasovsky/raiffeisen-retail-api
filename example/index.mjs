import api from '@actual-app/api';

// Import your transactions here
import transactions from './transactions.json' assert {type: 'json'}

(async () => {
  await api.init({
    // Budget data will be cached locally here.
    dataDir: 'data',
    serverURL: 'http://localhost:5656',
    password: 'YOUR_PASSWORD',
  });

  await api.downloadBudget('SYNC-ID-FROM-ADVANCED-SETTINGS');

  const accounts = await api.getAccounts();

  transactions.forEach((transaction, i) => {
    transactions[i]['date'] = new Date(Date.parse(transaction['date']))
  })

  await api.importTransactions(accounts[0].id, transactions)

  await api.shutdown();
})();
