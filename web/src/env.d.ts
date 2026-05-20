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
  interface ConfigOptions {
    clientId: string
    ssoUrl: string
    tokenUrl: string
    appId?: string
  }

  interface LoginOptions {
    invalidateToken?: boolean
  }

  export function configUserCenter(options: ConfigOptions): void
  export function login(options?: LoginOptions): Promise<void>
  export function logout(): Promise<void>
}
