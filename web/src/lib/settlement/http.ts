import axios from 'axios'
import { login } from '../iam'
import type {
  UserInfo,
  FlowNodeItem,
  SettlementCondition,
  LuohuDetail,
  BaoMingDetail,
  HukouChangeStatus,
  HuKouInfo,
} from './types'

const apiInstance = axios.create({
  baseURL: '/hrssc',
  timeout: 15000,
  withCredentials: true,
})

apiInstance.interceptors.request.use((config) => {
  const token = localStorage.getItem('haier-user-center-access-token') || ''
  config.headers['access-token'] = token
  return config
})

apiInstance.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401 || error.response?.status === 403) {
      login({ invalidateToken: true })
    }
    return Promise.reject(error)
  },
)

let sessionReady: Promise<void> | null = null

export function ensureSession(): Promise<void> {
  if (!sessionReady) {
    const token = localStorage.getItem('haier-user-center-access-token') || ''
    sessionReady = axios.get('/hrssc/feishuNew/loginCallback', {
      params: { accessToken: token },
      withCredentials: true,
    }).then(() => {}).catch(() => {})
  }
  return sessionReady
}

interface ApiResponse<T> {
  success: boolean
  result?: T
  data?: T
  obj?: T
  message?: string
  msg?: string
  code?: string
  code1?: string
  aorderCode?: string
}

export async function getAllUserInfo() {
  const res = await apiInstance.post<ApiResponse<UserInfo>>('/user/getAllUserInfo')
  return res.data
}

export async function getJxByEmpCode() {
  const res = await apiInstance.post<ApiResponse<{ isJx: boolean; reason?: string }>>(
    '/eRecord/getJxByEmpCode',
  )
  return res.data
}

export async function getAIToDeal(params: { userCode: string }) {
  const formData = new FormData()
  formData.append('userCode', params.userCode)
  const res = await apiInstance.post<ApiResponse<FlowNodeItem[]> & { code1?: string; aorderCode?: string }>(
    '/businesshall/businessinfo/getAIToDeal',
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return res.data
}

export async function getListDictionaryByIdAndYqId(params: { id: string; yqId?: string }) {
  const searchParams = new URLSearchParams()
  searchParams.append('id', params.id)
  if (params.yqId) searchParams.append('yqId', params.yqId)
  const res = await apiInstance.post<ApiResponse<SettlementCondition[]>>(
    '/eRecord/getListDictionaryByIdAndYqId',
    searchParams.toString(),
    { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } },
  )
  return res.data
}

export async function baomingByLh(params: Record<string, unknown>) {
  const searchParams = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null) {
      searchParams.append(key, String(value))
    }
  })
  const res = await apiInstance.post<ApiResponse<{ orderCode: string }>>(
    '/eRecord/baomingByLh',
    searchParams.toString(),
    { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } },
  )
  return res.data
}

export async function saveFile(formData: FormData) {
  const res = await apiInstance.post<ApiResponse<{ fileUUID: string; fileName: string }>>(
    '/new/fileManage/saveFile',
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return res.data
}

export async function getLuohuDetailsByOrderCode(params: { orderCode: string }) {
  const formData = new FormData()
  formData.append('orderCode', params.orderCode)
  const res = await apiInstance.post<LuohuDetail>(
    '/eRecord/getLuohuDetailsByOrderCode',
    formData,
  )
  return res.data
}

export async function lhsq(params: {
  orderCode: string
  jsonListStr: string
  registeredOutsideProvince: string
  onlineMigration: string
}) {
  const formData = new FormData()
  formData.append('orderCode', params.orderCode)
  formData.append('jsonListStr', params.jsonListStr)
  formData.append('registeredOutsideProvince', params.registeredOutsideProvince)
  formData.append('onlineMigration', params.onlineMigration)
  const res = await apiInstance.post<{ success: boolean; msg?: string; code?: string }>(
    '/eRecord/lhsq',
    formData,
  )
  return res.data
}

export async function getPcsByOrderCode(params: { orderCode: string }) {
  const formData = new FormData()
  formData.append('orderCode', params.orderCode)
  const res = await apiInstance.post<{
    success: boolean
    obj?: { companyCode: string; companyName: string; registeredArea: string; registeredAddress: string; policeStation: string }
  }>('/eRecord/getPcsByOrderCode', formData)
  return res.data
}

export async function checkHuKouTypeApply(params: { orderCode: string }) {
  const formData = new FormData()
  formData.append('orderCode', params.orderCode)
  const res = await apiInstance.post<{ success: boolean; msg?: string; obj?: HukouChangeStatus }>(
    '/eRecord/checkHuKouTypeApply',
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return res.data
}

export async function getUserHuKouInfo() {
  const formData = new FormData()
  const res = await apiInstance.post<{ success: boolean; msg: string; obj: HuKouInfo; code: string }>(
    '/eRecord/getUserHuKouInfo',
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return res.data
}

export async function hukouTypeApply(params: Record<string, unknown>) {
  const formData = new FormData()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null) {
      formData.append(key, String(value))
    }
  })
  const res = await apiInstance.post<{ success: boolean; msg?: string; code?: string }>(
    '/eRecord/hukouTypeApply',
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return res.data
}

export async function getLhBaoMingByOrderCode(params: { orderCode: string }) {
  const formData = new FormData()
  formData.append('orderCode', params.orderCode)
  const res = await apiInstance.post<ApiResponse<BaoMingDetail>>(
    '/eRecord/getLhBaoMingByOrderCode',
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return res.data
}

export async function ocrHouseholdStamp(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  const res = await apiInstance.post<{
    success: boolean
    msg: string
    code: number
    obj?: {
      household?: { item_list: Array<{ key: string; value: string; confidence: number; description: string }>; type: string }
      stamp?: { details: { stamp: Array<{ type: string; value: string; color: string; stamp_shape: string }> } }
    }
  }>('/ocr/household-stamp', formData, { headers: { 'Content-Type': 'multipart/form-data' }, timeout: 30000 })
  return res.data
}
