import React, { useState } from 'react'
import { useAuthStore } from '../stores/auth'
import { api } from '../lib/ipc'

type Tab = 'login' | 'signup'

export function AuthScreen() {
  const [tab, setTab] = useState<Tab>('login')
  const { isLoading, error, setLoading, setError, setUser, setSubscription, setScreen } =
    useAuthStore()

  const isDev = !!(window as any).__VITE_DEV_SERVER_URL || (import.meta as any).env?.DEV

  const handleDevBypass = () => {
    setUser({
      id: 'dev-user-001',
      email: 'dev@humanstudio.local',
      name: 'Dev User',
      auth_provider: 'dev',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    })
    setSubscription({
      id: 'dev-sub-001',
      user_id: 'dev-user-001',
      plan: 'pro',
      status: 'trialing',
      trial_end: new Date(Date.now() + 14 * 24 * 60 * 60 * 1000).toISOString(),
      created_at: new Date().toISOString(),
    })
    setScreen('app')
  }

  const handleOAuth = async (provider: string) => {
    setLoading(true)
    setError(null)
    try {
      const result = await api.auth.oauth(provider)
      setUser(result.user)

      if (result.is_new_user) {
        setScreen('plan-select')
      } else {
        // Existing user — fetch subscription and go to app
        try {
          const sub = await api.auth.getSubscription()
          setSubscription(sub)
        } catch {
          // Non-fatal: default to free
        }
        setScreen('app')
      }
    } catch (err: any) {
      if (err.message !== 'OAuth window was closed') {
        setError(err.message || 'Authentication failed')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'var(--bg)',
      }}
    >
      <div
        style={{
          width: 400,
          background: 'var(--bg-raised)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius-lg, 12px)',
          padding: 40,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 28,
        }}
      >
        {/* Logo */}
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
          <svg width="48" height="48" viewBox="0 0 120 120">
            <rect width="120" height="120" rx="24" fill="var(--bg-surface)" />
            <text
              x="24"
              y="84"
              fontFamily="Nunito, sans-serif"
              fontWeight="700"
              fontSize="72"
              letterSpacing="-1"
            >
              <tspan fill="var(--text-bright)">h</tspan>
              <tspan fill="var(--accent)">_</tspan>
            </text>
          </svg>
          <span
            style={{
              fontSize: 22,
              fontWeight: 700,
              color: 'var(--text-bright)',
              fontFamily: 'var(--font-logo)',
            }}
          >
            Human Studio
          </span>
          <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
            Build full-stack apps with structured English
          </span>
        </div>

        {/* Tabs */}
        <div
          style={{
            display: 'flex',
            width: '100%',
            background: 'var(--bg-surface)',
            borderRadius: 'var(--radius-sm)',
            padding: 3,
          }}
        >
          {(['login', 'signup'] as Tab[]).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              style={{
                flex: 1,
                padding: '8px 0',
                fontSize: 13,
                fontWeight: 600,
                fontFamily: 'var(--font-body)',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                background: tab === t ? 'var(--bg-raised)' : 'transparent',
                color: tab === t ? 'var(--text-bright)' : 'var(--text-muted)',
                transition: 'all 150ms',
              }}
            >
              {t === 'login' ? 'Log In' : 'Sign Up'}
            </button>
          ))}
        </div>

        {/* OAuth Buttons */}
        <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 10 }}>
          <OAuthButton
            provider="google"
            label="Continue with Google"
            icon={<GoogleIcon />}
            onClick={() => handleOAuth('google')}
            disabled={isLoading}
          />
          <OAuthButton
            provider="slack"
            label="Continue with Slack"
            icon={<SlackIcon />}
            onClick={() => handleOAuth('slack')}
            disabled={isLoading}
          />
          <OAuthButton
            provider="outlook"
            label="Continue with Outlook"
            icon={<OutlookIcon />}
            onClick={() => handleOAuth('outlook')}
            disabled={isLoading}
          />
        </div>

        {/* Loading */}
        {isLoading && (
          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>Authenticating...</div>
        )}

        {/* Error */}
        {error && (
          <div
            style={{
              width: '100%',
              padding: '10px 14px',
              fontSize: 12,
              color: 'var(--error)',
              background: 'var(--bg-surface)',
              borderRadius: 'var(--radius-sm)',
              border: '1px solid var(--error)',
            }}
          >
            {error}
          </div>
        )}

        {/* Dev bypass */}
        {isDev && (
          <button
            onClick={handleDevBypass}
            style={{
              padding: '8px 16px',
              fontSize: 12,
              color: 'var(--text-dim)',
              background: 'transparent',
              border: '1px dashed var(--border)',
              borderRadius: 'var(--radius-sm)',
              cursor: 'pointer',
              fontFamily: 'var(--font-body)',
              transition: 'all 150ms',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--text-muted)'
              e.currentTarget.style.color = 'var(--text-muted)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--border)'
              e.currentTarget.style.color = 'var(--text-dim)'
            }}
          >
            Continue as Dev User
          </button>
        )}

        {/* Footer */}
        <div style={{ fontSize: 10, color: 'var(--text-dim)', textAlign: 'center' }}>
          By continuing, you agree to our Terms of Service and Privacy Policy
        </div>
      </div>
    </div>
  )
}

function OAuthButton({
  label,
  icon,
  onClick,
  disabled,
}: {
  provider: string
  label: string
  icon: React.ReactNode
  onClick: () => void
  disabled: boolean
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        width: '100%',
        padding: '12px 16px',
        fontSize: 13,
        fontWeight: 600,
        fontFamily: 'var(--font-body)',
        background: 'var(--bg-surface)',
        color: 'var(--text-bright)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius-sm)',
        cursor: disabled ? 'default' : 'pointer',
        opacity: disabled ? 0.6 : 1,
        transition: 'all 150ms',
      }}
      onMouseEnter={(e) => {
        if (!disabled) e.currentTarget.style.borderColor = 'var(--accent)'
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = 'var(--border)'
      }}
    >
      {icon}
      {label}
    </button>
  )
}

function GoogleIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24">
      <path
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
        fill="#4285F4"
      />
      <path
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
        fill="#34A853"
      />
      <path
        d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
        fill="#FBBC05"
      />
      <path
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
        fill="#EA4335"
      />
    </svg>
  )
}

function SlackIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24">
      <path
        d="M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zm1.271 0a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313z"
        fill="#E01E5A"
      />
      <path
        d="M8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zm0 1.271a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312z"
        fill="#36C5F0"
      />
      <path
        d="M18.956 8.834a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zm-1.27 0a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.163 0a2.528 2.528 0 0 1 2.523 2.522v6.312z"
        fill="#2EB67D"
      />
      <path
        d="M15.163 18.956a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.163 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zm0-1.27a2.527 2.527 0 0 1-2.52-2.523 2.527 2.527 0 0 1 2.52-2.52h6.315A2.528 2.528 0 0 1 24 15.163a2.528 2.528 0 0 1-2.522 2.523h-6.315z"
        fill="#ECB22E"
      />
    </svg>
  )
}

function OutlookIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24">
      <path d="M24 7.387v10.478c0 .23-.08.424-.238.576a.806.806 0 0 1-.588.236h-8.174v-12.5h8.174c.228 0 .423.08.588.236.159.152.238.346.238.576v.398z" fill="#0364B8" />
      <path d="M24 7.387v-.398c0-.23-.08-.424-.238-.576A.806.806 0 0 0 23.174 6.177H15v6.5h9V7.387z" fill="#0A2767" />
      <path d="M15 6.177v12.5h8.174c.228 0 .423-.08.588-.236.159-.152.238-.346.238-.576V7.387L15 6.177z" fill="#0364B8" />
      <path d="M13.313 5.29H0v14.58l13.313-2.37V5.29z" fill="#0364B8" />
      <path
        d="M6.656 9.804c-.92 0-1.66.312-2.22.938-.56.625-.84 1.427-.84 2.406 0 .96.273 1.745.82 2.354.546.61 1.27.914 2.17.914.93 0 1.68-.306 2.24-.918.56-.612.84-1.408.84-2.39 0-.98-.273-1.786-.82-2.406-.546-.6-1.27-.898-2.19-.898zm-.07 1.18c.53 0 .945.196 1.25.586.304.39.456.918.456 1.578 0 .68-.156 1.22-.468 1.61-.312.39-.73.587-1.25.587-.536 0-.955-.193-1.258-.578-.303-.385-.454-.917-.454-1.594 0-.672.153-1.204.46-1.594.304-.398.727-.594 1.264-.594z"
        fill="white"
      />
    </svg>
  )
}
