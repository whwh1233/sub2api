export default {
  balances: {
    title: 'Balance Overview',
    description: 'Review aggregate user balances and quickly open balance adjustments or history',
    searchPlaceholder: 'Search email, username, or notes...',
    summary: {
      totalBalance: 'Total Balance',
      positiveUsers: 'Users With Balance',
      lowBalanceUsers: 'Low Balance Users',
      abnormalUsers: 'Abnormal Balances'
    },
    filters: {
      allBalanceStates: 'All Balances',
      positive: 'Positive Balance',
      low: 'Low Balance',
      abnormal: 'Abnormal Balance',
      zero: 'Zero Balance'
    },
    columns: {
      user: 'User',
      balance: 'Balance',
      status: 'Status',
      lastActive: 'Last Active',
      lastUsed: 'Last Used',
      created: 'Created At',
      actions: 'Actions'
    },
    actions: {
      deposit: 'Top Up',
      history: 'History'
    },
    empty: {
      title: 'No balance data'
    },
    errors: {
      summaryFailed: 'Failed to load balance summary',
      usersFailed: 'Failed to load user balance list'
    }
  }
}
