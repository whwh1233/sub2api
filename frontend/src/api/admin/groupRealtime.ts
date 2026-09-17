import { apiClient } from '../client'

export interface GroupRealtimeRow {
  group_id: number
  group_name: string
  platform: string
  status: string
  rpm: number
  success: number
  failed: number
  cancelled: number
  success_rate: number | null
  failures: Record<string, number>
}

export interface GroupRealtimeSnapshot {
  start_time: string
  end_time: string
  collecting_since: string
  partial: boolean
  groups: GroupRealtimeRow[]
}

export async function getGroupRealtime(signal?: AbortSignal): Promise<GroupRealtimeSnapshot> {
  const { data } = await apiClient.get<GroupRealtimeSnapshot>('/admin/ops/group-realtime', { signal })
  return data
}
