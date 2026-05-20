import { configUserCenter, login, logout } from '@haier/iam'

export function initIAM() {
  configUserCenter({
    clientId: import.meta.env.VITE_IAM_CLIENT_ID,
    ssoUrl: import.meta.env.VITE_IAM_SSO_URL,
    tokenUrl: import.meta.env.VITE_IAM_TOKEN_URL,
  })
}

export { login, logout }
