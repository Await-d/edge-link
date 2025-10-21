import axios, { AxiosInstance, AxiosError } from 'axios'
import { ErrorResponse } from '@/types/api'

const apiClient: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    // TODO: 添加认证token
    // const token = localStorage.getItem('auth_token')
    // if (token) {
    //   config.headers.Authorization = `Bearer ${token}`
    // }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
apiClient.interceptors.response.use(
  (response) => {
    return response
  },
  (error: AxiosError<ErrorResponse>) => {
    if (error.response) {
      // 服务器返回错误状态码
      const { status, data } = error.response

      if (status === 401) {
        // 未授权,跳转到登录页
        // window.location.href = '/login'
        console.error('Unauthorized access')
      } else if (status === 403) {
        console.error('Forbidden:', data.message)
      } else if (status === 404) {
        console.error('Resource not found:', data.message)
      } else if (status >= 500) {
        console.error('Server error:', data.message)
      }
    } else if (error.request) {
      // 请求已发送但未收到响应
      console.error('Network error: No response from server')
    } else {
      // 请求配置错误
      console.error('Request error:', error.message)
    }

    return Promise.reject(error)
  }
)

export default apiClient
