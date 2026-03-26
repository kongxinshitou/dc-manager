import axios from 'axios'

const api = axios.create({
  baseURL: 'http://localhost:8080/api',
  timeout: 30000,
})

export interface Device {
  id: number
  source: string
  asset_number: string
  status: string
  datacenter: string
  cabinet: string
  u_position: string
  start_u: number | null
  end_u: number | null
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
  start_u: number | null
  end_u: number | null
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
  order_by?: string
  sort?: 'asc' | 'desc'
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
  order_by?: string
  sort?: 'asc' | 'desc'
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

export const batchDeleteDevices = (ids: number[]) =>
  api.delete<{ deleted: number }>('/devices/batch', { data: { ids } }).then(r => r.data)

export const getDeviceOptions = () =>
  api.get<{ sources: string[]; datacenters: string[]; device_types: string[]; brands: string[] }>('/devices/options').then(r => r.data)

export const getDeviceCabinets = (datacenter: string) =>
  api.get<{ cabinets: string[] }>('/devices/cabinets', { params: { datacenter } }).then(r => r.data)

export const getDeviceByLocation = (datacenter: string, cabinet: string, startU: number | null, endU: number | null) =>
  api.get<{ device: Device | null }>('/devices/by-location', {
    params: { datacenter, cabinet, start_u: startU ?? undefined, end_u: endU ?? undefined }
  }).then(r => r.data)

export const exportDevices = (params: DeviceQuery) =>
  api.get('/devices/export', { params, responseType: 'blob' }).then(r => r.data as Blob)

export const importDevicesPreview = (file: File) => {
  const form = new FormData()
  form.append('file', file)
  return api.post<{ preview: Device[]; count: number }>('/devices/import', form).then(r => r.data)
}

export const importDevicesConfirm = (file: File) => {
  const form = new FormData()
  form.append('file', file)
  return api.post<{ inserted: number; skipped: number; message: string }>('/devices/import?confirm=true', form).then(r => r.data)
}

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

export const batchDeleteInspections = (ids: number[]) =>
  api.delete<{ deleted: number }>('/inspections/batch', { data: { ids } }).then(r => r.data)

export const importInspectionsPreview = (file: File) => {
  const form = new FormData()
  form.append('file', file)
  return api.post<{ preview: Inspection[]; count: number }>('/inspections/import', form).then(r => r.data)
}

export const importInspectionsConfirm = (file: File) => {
  const form = new FormData()
  form.append('file', file)
  return api.post<{ inserted: number; skipped: number; message: string }>('/inspections/import?confirm=true', form).then(r => r.data)
}

// Dashboard
export const getDashboard = (params?: { issues_page?: number; issues_page_size?: number }) =>
  api.get('/dashboard', { params }).then(r => r.data)

export default api
