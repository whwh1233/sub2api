export default {
  balances: {
    title: '余额总览',
    description: '查看全站用户余额汇总，并快速进入余额调整和明细',
    searchPlaceholder: '搜索邮箱、用户名或备注...',
    summary: {
      totalBalance: '总余额',
      positiveUsers: '有余额用户',
      lowBalanceUsers: '低余额用户',
      abnormalUsers: '异常余额用户'
    },
    filters: {
      allBalanceStates: '全部余额',
      positive: '有余额',
      low: '低余额',
      abnormal: '异常余额',
      zero: '零余额'
    },
    columns: {
      user: '用户',
      balance: '余额',
      status: '状态',
      lastActive: '最后活跃',
      lastUsed: '最后使用',
      created: '创建时间',
      actions: '操作'
    },
    actions: {
      deposit: '充值',
      history: '明细'
    },
    empty: {
      title: '暂无余额数据'
    },
    errors: {
      summaryFailed: '加载余额汇总失败',
      usersFailed: '加载用户余额列表失败'
    }
  }
}
