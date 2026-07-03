import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'

export interface RelayBalanceStation {
  id: number
  name: string
  base_url: string
  script: string
  package_json: string
  cron_expression: string
  enabled: boolean
  last_balance: number | null
  last_currency: string
  last_status: string
  last_error: string
  last_run_at: string | null
  next_run_at: string | null
  created_at: string
  updated_at: string
}

export interface RelayBalanceRun {
  id: number
  station_id: number
  station_name: string
  balance: number | null
  currency: string
  status: string
  stdout: string
  stderr: string
  error: string
  raw: string
  duration_ms: number
  started_at: string
  finished_at: string | null
}

export interface RelayBalanceStationRequest {
  name: string
  base_url: string
  script: string
  package_json: string
  cron_expression: string
  enabled: boolean
}

export async function listStations(
  page = 1,
  pageSize = 20,
  filters?: { search?: string; enabled?: string }
): Promise<BasePaginationResponse<RelayBalanceStation>> {
  const { data } = await apiClient.get<BasePaginationResponse<RelayBalanceStation>>('/admin/relay-balance/stations', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function createStation(request: RelayBalanceStationRequest): Promise<RelayBalanceStation> {
  const { data } = await apiClient.post<RelayBalanceStation>('/admin/relay-balance/stations', request)
  return data
}

export async function updateStation(id: number, request: RelayBalanceStationRequest): Promise<RelayBalanceStation> {
  const { data } = await apiClient.put<RelayBalanceStation>(`/admin/relay-balance/stations/${id}`, request)
  return data
}

export async function deleteStation(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/relay-balance/stations/${id}`)
  return data
}

export async function runStation(id: number): Promise<RelayBalanceRun> {
  const { data } = await apiClient.post<RelayBalanceRun>(`/admin/relay-balance/stations/${id}/run`)
  return data
}

export async function listRuns(
  page = 1,
  pageSize = 20,
  filters?: {
    station_id?: number | string
    status?: string
    started_from?: string
    started_to?: string
    sort_order?: 'asc' | 'desc'
    granularity?: '' | 'hour' | 'day'
  }
): Promise<BasePaginationResponse<RelayBalanceRun>> {
  const { data } = await apiClient.get<BasePaginationResponse<RelayBalanceRun>>('/admin/relay-balance/runs', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export interface RelayBalanceTrendResponse {
  buckets: string[]
  series: RelayBalanceTrendSeries[]
  total: number[]
}

export interface RelayBalanceTrendSeries {
  station_id: number
  station_name: string
  balances: number[]
}

export interface RelayBalanceTotalResponse {
  total_balance: number
  currency: string
  station_count: number
}

export async function getTotalBalance(): Promise<RelayBalanceTotalResponse> {
  const { data } = await apiClient.get<RelayBalanceTotalResponse>('/admin/relay-balance/total-balance')
  return data
}

export async function getTrend(params: {
  started_from?: string
  started_to?: string
  granularity?: 'hour' | 'day'
}): Promise<RelayBalanceTrendResponse> {
  const { data } = await apiClient.get<RelayBalanceTrendResponse>('/admin/relay-balance/trend', { params })
  return data
}

const relayBalanceAPI = {
  listStations,
  createStation,
  updateStation,
  deleteStation,
  runStation,
  listRuns,
  getTotalBalance,
  getTrend
}

export default relayBalanceAPI
