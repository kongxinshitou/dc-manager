import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

// JWT 请求拦截器
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 401 响应拦截器
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.reload()
    }
    return Promise.reject(error)
  }
)

// ========== 类型定义 ==========

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
  images?: InspectionImage[]
  created_at: string
  updated_at: string
}

export interface InspectionImage {
  id: number
  inspection_id: number
  file_path: string
  file_name: string
  file_size: number
  content_type: string
  uploaded_at: string
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

export interface UserInfo {
  id: number
  username: string
  display_name: string
  role_id: number
  role_name: string
  permissions: string[]
  status?: string
  role?: RoleInfo
  created_at?: string
}

export interface RoleInfo {
  id: number
  name: string
  display_name: string
  permissions: string
  is_system: boolean
}

export interface PermGroup {
  label: string
  permissions: { code: string; label: string }[]
}

// ========== Auth API ==========

export const login = (username: string, password: string) =>
  api.post<{ token: string; user: UserInfo }>('/auth/login', { username, password }).then(r => r.data)

export const changePassword = (oldPassword: string, newPassword: string) =>
  api.post('/auth/change-password', { old_password: oldPassword, new_password: newPassword }).then(r => r.data)

// ========== User API ==========

export const getUsers = () =>
  api.get<UserInfo[]>('/users').then(r => r.data)

export const createUser = (data: { username: string; password: string; display_name: string; role_id: number }) =>
  api.post<UserInfo>('/users', data).then(r => r.data)

export const updateUser = (id: number, data: Partial<UserInfo>) =>
  api.put<UserInfo>(`/users/${id}`, data).then(r => r.data)

export const resetPassword = (id: number, newPassword: string) =>
  api.put(`/users/${id}/reset-password`, { new_password: newPassword }).then(r => r.data)

export const deleteUser = (id: number) =>
  api.delete(`/users/${id}`).then(r => r.data)

// ========== Role API ==========

export const getRoles = () =>
  api.get<RoleInfo[]>('/roles').then(r => r.data)

export const createRole = (data: { name: string; display_name: string; permissions: string[] }) =>
  api.post<RoleInfo>('/roles', data).then(r => r.data)

export const updateRole = (id: number, data: { name?: string; display_name?: string; permissions?: string[] }) =>
  api.put<RoleInfo>(`/roles/${id}`, data).then(r => r.data)

export const deleteRole = (id: number) =>
  api.delete(`/roles/${id}`).then(r => r.data)

export const getPermissionInfo = () =>
  api.get<{ groups: PermGroup[]; all: string[] }>('/permissions').then(r => r.data)

// ========== Device API ==========

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

// ========== Inspection API ==========

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

// ========== Image API ==========

export const uploadInspectionImages = (inspectionId: number, files: File[]) => {
  const form = new FormData()
  files.forEach(f => form.append('images', f))
  return api.post<InspectionImage[]>(`/inspections/${inspectionId}/images`, form, {
    headers: { 'Content-Type': 'multipart/form-data' }
  }).then(r => r.data)
}

export const deleteInspectionImage = (inspectionId: number, imageId: number) =>
  api.delete(`/inspections/${inspectionId}/images/${imageId}`).then(r => r.data)

export const getImageUrl = (filePath: string) => `/uploads/${filePath}`

// ========== Dashboard API ==========

export const getDashboard = (params?: { issues_page?: number; issues_page_size?: number }) =>
  api.get('/dashboard', { params }).then(r => r.data)

export default api
