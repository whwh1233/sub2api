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

export interface GroupHistoryRow {
  points: GroupHistoryPoint[]
  group_id: number
  group_name: string
  platform: string
  status: string
  started: number
  success: number
  failed: number
  cancelled: number
  rpm: number | null
  success_rate: number | null
}
export interface GroupHistoryPoint {
  bucket_start: string
  rpm: number | null
  success_rate: number | null
  started: number
  success: number
  failed: number
  cancelled: number
  partial: boolean
}
export interface GroupHistory {
  start_time: string
  end_time: string
  collecting_since: string
  bucket_seconds: number
  covered_seconds: number
  partial: boolean
  group_id: number | null
  groups: GroupHistoryRow[]
  points: GroupHistoryPoint[]
}
export type GroupHistoryWindow = 15 | 60 | 360 | 1440 | 10080
export async function getGroupHistory(windowMinutes: GroupHistoryWindow, groupID: number | null, signal?: AbortSignal): Promise<GroupHistory> {
  const { data } = await apiClient.get<GroupHistory>('/admin/ops/group-realtime/history', {
    params: { window_minutes: windowMinutes, ...(groupID !== null ? { group_id: groupID } : {}) }, signal,
  })
  return data
}
