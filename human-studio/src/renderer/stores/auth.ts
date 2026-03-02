import { create } from 'zustand'

export type AuthScreen = 'loading' | 'auth' | 'plan-select' | 'app'

export interface AuthUser {
  id: string
  email: string
  name: string
  auth_provider: string
  created_at: string
  updated_at: string
}

export interface AuthSubscription {
  id: string
  user_id: string
  plan: string
  status: string
  current_period_end?: string
  trial_end?: string
  created_at: string
}

interface AuthState {
  screen: AuthScreen
  user: AuthUser | null
  subscription: AuthSubscription | null
  isLoading: boolean
  error: string | null

  setScreen: (screen: AuthScreen) => void
  setUser: (user: AuthUser | null) => void
  setSubscription: (sub: AuthSubscription | null) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  screen: 'loading',
  user: null,
  subscription: null,
  isLoading: false,
  error: null,

  setScreen: (screen) => set({ screen }),
  setUser: (user) => set({ user }),
  setSubscription: (subscription) => set({ subscription }),
  setLoading: (isLoading) => set({ isLoading }),
  setError: (error) => set({ error }),
  logout: () => set({ screen: 'auth', user: null, subscription: null, error: null }),
}))
