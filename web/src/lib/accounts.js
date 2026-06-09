export function isOAuthAccount(account) {
  return !account?.account_type || account.account_type === 'oauth'
}

export function isAPIAccount(account) {
  return account?.account_type === 'api'
}

export function filterOAuthAccounts(accounts = []) {
  return accounts.filter(isOAuthAccount)
}

export function filterAPIAccounts(accounts = []) {
  return accounts.filter(isAPIAccount)
}
