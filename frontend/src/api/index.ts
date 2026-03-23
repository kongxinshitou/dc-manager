import axios from 'axios'

const api = axios.create({
  baseURL: 'http://localhost:8080/api',
  timeout: 10000,
})

export interface Device {
  id: number
  source: string
  asset_number: string
  status: string
  datacenter: string
  cabinet: string
  u_position: string
  brand: string
  model: string
  device_type: string
  serial_number: string
  os: string
  ip_address: string
  system_account: string
  mgmt_ip: string
  mgmt_account: string
  manufacture_date: string | null
  warranty_start: string | null
  warranty_end: string | null
  purpose: string
  owner: string
  remark: string
  created_at: string
  updated_at: string
}

export interface Inspection {
  id: number
  device_id: number | null
  device?: Device
  datacenter: string
  cabinet: string
  u_position: string
  found_at: string
  inspector: string
  issue: string
  severity: string
  status: string
  resolved_at: string | null
  remark: string
  created_at: string
  updated_at: string
}

export interface PageResult<T> {
  total: number
  page: number
  data: T[]
}

export interface DeviceQuery {
  source?: string
  status?: string
  datacenter?: string
  cabinet?: string
  brand?: string
  model?: string
  device_type?: string
  ip_address?: string
  owner?: string
  keyword?: string
  page?: number
  page_size?: number
}

export interface InspectionQuery {
  datacenter?: string
  cabinet?: string
  inspector?: string
  severity?: string
  status?: string
  start_time?: string
  end_time?: string
  keyword?: string
  page?: number
  page_size?: number
}

// Devices
export const getDevices = (params: DeviceQuery) =>
  api.get<PageResult<Device>>('/devices', { params }).then(r => r.data)

export const getDevice = (id: number) =>
  api.get<Device>(`/devices/${id}`).then(r => r.data)

export const createDevice = (data: Partial<Device>) =>
  api.post<Device>('/devices', data).then(r => r.data)

export const updateDevice = (id: number, data: Partial<Device>) =>
  api.put<Device>(`/devices/${id}`, data).then(r => r.data)

export const deleteDevice = (id: number) =>
  api.delete(`/devices/${id}`).then(r => r.data)

export const getDeviceOptions = () =>
  api.get<{ sources: string[]; datacenters: string[]; device_types: string[]; brands: string[] }>('/devices/options').then(r => r.data)

// Inspections
export const getInspections = (params: InspectionQuery) =>
  api.get<PageResult<Inspection>>('/inspections', { params }).then(r => r.data)

export const getInspection = (id: number) =>
  api.get<Inspection>(`/inspections/${id}`).then(r => r.data)

export const createInspection = (data: Partial<Inspection>) =>
  api.post<Inspection>('/inspections', data).then(r => r.data)

export const updateInspection = (id: number, data: Partial<Inspection>) =>
  api.put<Inspection>(`/inspections/${id}`, data).then(r => r.data)

export const deleteInspection = (id: number) =>
  api.delete(`/inspections/${id}`).then(r => r.data)

// Dashboard
export const getDashboard = () =>
  api.get('/dashboard').then(r => r.data)

export default api
