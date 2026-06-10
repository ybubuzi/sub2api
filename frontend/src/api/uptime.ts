import { apiClient } from './client'

export type UptimeWindow = '1h' | '6h'
export type UserUptimeDimension = 'all' | 'api_key' | 'model'
export type AdminUptimeDimension = 'all' | 'model' | 'group'

export interface UptimeBucket {
  start: string
  total: number
  success: number
  failure: number
  availability: number | null
}

export interface UptimeSeriesStats {
  total: number
  success: number
  failure: number
  availability: number | null
}

export interface UptimeSeries {
  key: string
  label: string
  summary: UptimeSeriesStats
  buckets: UptimeBucket[]
}

export interface UptimeChartResponse {
  scope: 'user' | 'admin'
  window: UptimeWindow
  dimension: UserUptimeDimension | AdminUptimeDimension
  start_time: string
  end_time: string
  bucket_seconds: number
  success_source: string
  failure_source: string
  sla_excludes_business_limited: boolean
  summary: UptimeSeriesStats
  series: UptimeSeries[]
}

export async function getUserUptime(
  params: { window?: UptimeWindow; dimension?: UserUptimeDimension },
  options?: { signal?: AbortSignal }
): Promise<UptimeChartResponse> {
  const { data } = await apiClient.get<UptimeChartResponse>('/usage/uptime', {
    params,
    signal: options?.signal
  })
  return data
}

export async function getAdminUptime(
  params: { window?: UptimeWindow; dimension?: AdminUptimeDimension },
  options?: { signal?: AbortSignal }
): Promise<UptimeChartResponse> {
  const { data } = await apiClient.get<UptimeChartResponse>('/admin/usage/uptime', {
    params,
    signal: options?.signal
  })
  return data
}

export const uptimeAPI = {
  getUserUptime,
  getAdminUptime
}

export default uptimeAPI
