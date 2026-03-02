import React, { useState } from 'react'
import { useAuthStore } from '../stores/auth'
import { api } from '../lib/ipc'

export function PlanSelector() {
  const [loading, setLoading] = useState(false)
  const { setSubscription, setScreen } = useAuthStore()

  const handleSelect = async (plan: string) => {
    setLoading(true)
    try {
      const sub = await api.auth.selectPlan(plan)
      setSubscription(sub)
      setScreen('app')
    } catch {
      // Fall through to app even if plan selection fails
      setScreen('app')
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
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 32,
          maxWidth: 700,
        }}
      >
        {/* Header */}
        <div style={{ textAlign: 'center' }}>
          <h1
            style={{
              fontSize: 24,
              fontWeight: 700,
              color: 'var(--text-bright)',
              fontFamily: 'var(--font-heading)',
              margin: '0 0 8px',
            }}
          >
            Choose your plan
          </h1>
          <p style={{ fontSize: 13, color: 'var(--text-muted)', margin: 0 }}>
            Start building with Human Studio. Upgrade anytime.
          </p>
        </div>

        {/* Plan Cards */}
        <div style={{ display: 'flex', gap: 20 }}>
          {/* Free Plan */}
          <PlanCard
            name="Free"
            price="$0"
            period="/month"
            features={[
              'Unlimited .human files',
              'All code generators',
              'Local builds',
              'Community support',
            ]}
            buttonLabel="Get Started"
            buttonVariant="secondary"
            onSelect={() => handleSelect('free')}
            disabled={loading}
          />

          {/* Pro Plan */}
          <PlanCard
            name="Pro"
            price="$19"
            period="/month"
            badge="RECOMMENDED"
            trialBadge="14-day free trial"
            features={[
              'Everything in Free',
              'Cloud deployments',
              'Team collaboration',
              'AI-powered assistance',
              'Priority support',
              'Custom themes',
            ]}
            buttonLabel="Start Free Trial"
            buttonVariant="primary"
            highlighted
            onSelect={() => handleSelect('pro')}
            disabled={loading}
          />
        </div>
      </div>
    </div>
  )
}

function PlanCard({
  name,
  price,
  period,
  badge,
  trialBadge,
  features,
  buttonLabel,
  buttonVariant,
  highlighted,
  onSelect,
  disabled,
}: {
  name: string
  price: string
  period: string
  badge?: string
  trialBadge?: string
  features: string[]
  buttonLabel: string
  buttonVariant: 'primary' | 'secondary'
  highlighted?: boolean
  onSelect: () => void
  disabled: boolean
}) {
  return (
    <div
      style={{
        width: 300,
        background: 'var(--bg-raised)',
        border: highlighted ? '2px solid var(--accent)' : '1px solid var(--border)',
        borderRadius: 'var(--radius-lg, 12px)',
        padding: 28,
        display: 'flex',
        flexDirection: 'column',
        gap: 20,
        position: 'relative',
      }}
    >
      {/* Badge */}
      {badge && (
        <div
          style={{
            position: 'absolute',
            top: -10,
            right: 20,
            padding: '4px 10px',
            fontSize: 9,
            fontWeight: 700,
            letterSpacing: '0.05em',
            background: 'var(--accent)',
            color: '#fff',
            borderRadius: 4,
          }}
        >
          {badge}
        </div>
      )}

      {/* Name + Price */}
      <div>
        <div
          style={{
            fontSize: 18,
            fontWeight: 700,
            color: 'var(--text-bright)',
            fontFamily: 'var(--font-heading)',
          }}
        >
          {name}
        </div>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 2, marginTop: 4 }}>
          <span style={{ fontSize: 32, fontWeight: 700, color: 'var(--text-bright)' }}>
            {price}
          </span>
          <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>{period}</span>
        </div>
        {trialBadge && (
          <div
            style={{
              display: 'inline-block',
              marginTop: 6,
              padding: '3px 8px',
              fontSize: 10,
              fontWeight: 600,
              background: 'var(--success)',
              color: '#fff',
              borderRadius: 4,
            }}
          >
            {trialBadge}
          </div>
        )}
      </div>

      {/* Features */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10, flex: 1 }}>
        {features.map((f) => (
          <div key={f} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="var(--success)"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
            <span style={{ fontSize: 12, color: 'var(--text)' }}>{f}</span>
          </div>
        ))}
      </div>

      {/* CTA Button */}
      <button
        onClick={onSelect}
        disabled={disabled}
        style={{
          width: '100%',
          padding: '12px 0',
          fontSize: 13,
          fontWeight: 700,
          fontFamily: 'var(--font-body)',
          border: buttonVariant === 'secondary' ? '1px solid var(--border)' : 'none',
          borderRadius: 'var(--radius-sm)',
          cursor: disabled ? 'default' : 'pointer',
          opacity: disabled ? 0.6 : 1,
          background: buttonVariant === 'primary' ? 'var(--accent)' : 'var(--bg-surface)',
          color: buttonVariant === 'primary' ? '#fff' : 'var(--text-bright)',
          transition: 'filter 150ms',
        }}
        onMouseEnter={(e) => {
          if (!disabled) e.currentTarget.style.filter = 'brightness(1.1)'
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.filter = ''
        }}
      >
        {buttonLabel}
      </button>
    </div>
  )
}
