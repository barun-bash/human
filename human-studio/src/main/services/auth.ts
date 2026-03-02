import { BrowserWindow, net } from 'electron'
import Store from 'electron-store'

const store = new Store({
  encryptionKey: 'human-studio-auth-v1',
  name: 'auth',
})

const API_BASE = process.env.HUMAN_API_URL || 'http://localhost:8080'

interface AuthData {
  accessToken: string
  refreshToken: string
  user: {
    id: string
    email: string
    name: string
    auth_provider: string
    created_at: string
    updated_at: string
  }
}

interface Subscription {
  id: string
  user_id: string
  plan: string
  status: string
  current_period_end?: string
  trial_end?: string
  created_at: string
}

export class AuthService {
  private mainWindow: BrowserWindow | null = null

  setMainWindow(win: BrowserWindow) {
    this.mainWindow = win
  }

  // Get stored auth data (tokens + user)
  getStoredAuth(): { auth: AuthData; subscription?: Subscription } | null {
    const auth = store.get('auth') as AuthData | undefined
    if (!auth?.accessToken) return null
    const subscription = store.get('subscription') as Subscription | undefined
    return { auth, subscription }
  }

  // Start OAuth flow in a popup BrowserWindow
  async startOAuth(provider: string): Promise<AuthData & { is_new_user: boolean }> {
    const authURL = `${API_BASE}/api/auth/oauth/${provider}/start`

    return new Promise((resolve, reject) => {
      const oauthWindow = new BrowserWindow({
        width: 600,
        height: 700,
        show: true,
        webPreferences: {
          nodeIntegration: false,
          contextIsolation: true,
        },
      })

      oauthWindow.loadURL(authURL)

      // Watch for the callback page to load
      oauthWindow.webContents.on('did-finish-load', async () => {
        try {
          const title = oauthWindow.webContents.getTitle()
          if (title !== 'human-oauth-callback') return

          const resultJSON = await oauthWindow.webContents.executeJavaScript(
            `document.getElementById('auth-result')?.textContent || '{}'`
          )

          const result = JSON.parse(resultJSON)

          if (result.error) {
            reject(new Error(result.error))
          } else {
            // Store tokens
            const authData: AuthData = {
              accessToken: result.access_token,
              refreshToken: result.refresh_token,
              user: result.user,
            }
            store.set('auth', authData)
            resolve({ ...authData, is_new_user: result.is_new_user })
          }
        } catch (err) {
          reject(err)
        } finally {
          oauthWindow.close()
        }
      })

      oauthWindow.on('closed', () => {
        // If window was closed manually before callback
        reject(new Error('OAuth window was closed'))
      })
    })
  }

  // Refresh access token using refresh token
  async refreshTokens(): Promise<AuthData | null> {
    const stored = store.get('auth') as AuthData | undefined
    if (!stored?.refreshToken) return null

    try {
      const resp = await this.apiRequest('POST', '/api/auth/refresh', {
        refresh_token: stored.refreshToken,
      }, false)

      const authData: AuthData = {
        accessToken: resp.access_token,
        refreshToken: resp.refresh_token,
        user: resp.user,
      }
      store.set('auth', authData)
      return authData
    } catch {
      // Refresh failed — session expired
      this.logout()
      if (this.mainWindow && !this.mainWindow.isDestroyed()) {
        this.mainWindow.webContents.send('auth:session-expired')
      }
      return null
    }
  }

  // Get user profile from server
  async getProfile(): Promise<AuthData['user']> {
    return this.authenticatedRequest('GET', '/api/user/profile')
  }

  // Get subscription from server
  async getSubscription(): Promise<Subscription> {
    const sub = await this.authenticatedRequest<Subscription>('GET', '/api/billing/subscription')
    store.set('subscription', sub)
    return sub
  }

  // Select plan for new users
  async selectPlan(plan: string): Promise<Subscription> {
    const sub = await this.authenticatedRequest<Subscription>('POST', '/api/billing/select-plan', { plan })
    store.set('subscription', sub)
    return sub
  }

  // Validate stored tokens (try to get profile)
  async validateSession(): Promise<boolean> {
    try {
      await this.getProfile()
      return true
    } catch {
      // Try refresh
      const refreshed = await this.refreshTokens()
      return refreshed !== null
    }
  }

  // Logout — clear all stored data
  logout(): void {
    store.delete('auth')
    store.delete('subscription')
  }

  // Make an authenticated API request with auto-retry on 401
  private async authenticatedRequest<T>(method: string, path: string, body?: any): Promise<T> {
    try {
      return await this.apiRequest(method, path, body, true)
    } catch (err: any) {
      if (err.statusCode === 401) {
        // Try refresh
        const refreshed = await this.refreshTokens()
        if (refreshed) {
          return this.apiRequest(method, path, body, true)
        }
      }
      throw err
    }
  }

  // Low-level API request
  private apiRequest(method: string, path: string, body?: any, withAuth?: boolean): Promise<any> {
    return new Promise((resolve, reject) => {
      const url = `${API_BASE}${path}`
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      }

      if (withAuth) {
        const stored = store.get('auth') as AuthData | undefined
        if (stored?.accessToken) {
          headers['Authorization'] = `Bearer ${stored.accessToken}`
        }
      }

      const request = net.request({
        method,
        url,
        headers,
      } as any)

      // Set headers
      for (const [key, value] of Object.entries(headers)) {
        request.setHeader(key, value)
      }

      request.on('response', (response) => {
        let data = ''
        response.on('data', (chunk) => {
          data += chunk.toString()
        })
        response.on('end', () => {
          if (response.statusCode && response.statusCode >= 400) {
            const err: any = new Error(`API error: ${response.statusCode}`)
            err.statusCode = response.statusCode
            try {
              err.body = JSON.parse(data)
            } catch {
              err.body = data
            }
            reject(err)
            return
          }
          try {
            resolve(JSON.parse(data))
          } catch {
            resolve(data)
          }
        })
      })

      request.on('error', reject)

      if (body) {
        request.write(JSON.stringify(body))
      }
      request.end()
    })
  }
}
