/**
 * Admin Balances API endpoints
 * Read-only aggregate balance metrics for administrators.
 */

import { apiClient } from '../client'

export interface AdminBalanceSummary {
  total_balance: number
  positive_balance_users: number
  low_balance_users: number
  abnormal_balance_users: number
  low_balance_threshold: number
  generated_at: string
}

export async function getSummary(): Promise<AdminBalanceSummary> {
  const { data } = await apiClient.get<AdminBalanceSummary>('/admin/balances/summary')
  return data
}

export const balancesAPI = {
  getSummary,
}

export default balancesAPI
