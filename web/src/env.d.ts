/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_IAM_CLIENT_ID: string
  readonly VITE_IAM_SSO_URL: string
  readonly VITE_IAM_TOKEN_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '@haier/iam' {
  interface UserCenterConfig {
    clientId: string
    ssoUrl: string
    tokenUrl: string
    redirectUri?: string
    exitUrl?: string
    appId?: string
  }

  interface LoginParams {
    force?: boolean
    invalidateToken?: boolean
  }

  interface LoginSuccessRes {
    success: true
    token?: string
    userInfo?: Record<string, unknown>
  }

  interface LoginFailRes {
    success: false
    errorMessage?: string
    code?: number
  }

  type LoginResponse = LoginSuccessRes | LoginFailRes

  export function configUserCenter(config: UserCenterConfig): void
  export function login(params?: LoginParams): Promise<LoginResponse>
  export function logout(): Promise<boolean>
}
