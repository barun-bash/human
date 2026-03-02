import React from 'react'
import { X, User, Key, Plug, CreditCard, LogOut, Trash2 } from 'lucide-react'
import { Button } from './ui/Button'
import { Input } from './ui/Input'
import { useAuthStore } from '../stores/auth'
import { api } from '../lib/ipc'

interface ProfilePanelProps {
  open: boolean
  onClose: () => void
}

const MCP_SERVICES = [
  { name: 'Figma', connected: true },
  { name: 'GitHub', connected: true },
  { name: 'Slack', connected: false },
  { name: 'Vercel', connected: false },
  { name: 'AWS', connected: false },
]

export function ProfilePanel({ open, onClose }: ProfilePanelProps) {
  const { user, subscription, logout } = useAuthStore()

  const handleLogout = () => {
    api.auth.logout()
    logout()
    onClose()
  }

  if (!open) return null

  return (
    <>
      {/* Overlay */}
      <div
        onClick={onClose}
        style={{
          position: 'fixed',
          inset: 0,
          zIndex: 40,
          background: 'rgba(0,0,0,0.3)',
        }}
      />

      {/* Panel */}
      <div
        style={{
          position: 'fixed',
          top: 0,
          right: 0,
          bottom: 0,
          zIndex: 50,
          width: 360,
          background: 'var(--bg-raised)',
          borderLeft: '1px solid var(--border)',
          boxShadow: '0 0 48px rgba(0,0,0,0.3)',
          overflowY: 'auto',
        }}
      >
        {/* Header */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '16px 24px',
            borderBottom: '1px solid var(--border)',
          }}
        >
          <h2
            style={{
              fontSize: 16,
              fontWeight: 600,
              color: 'var(--text-bright)',
              fontFamily: 'var(--font-heading)',
              margin: 0,
            }}
          >
            Profile
          </h2>
          <button
            onClick={onClose}
            style={{
              padding: 4,
              color: 'var(--text-muted)',
              background: 'transparent',
              border: 'none',
              borderRadius: 'var(--radius-sm)',
              cursor: 'pointer',
              display: 'flex',
            }}
          >
            <X size={16} />
          </button>
        </div>

        <div style={{ padding: 24, display: 'flex', flexDirection: 'column', gap: 32 }}>
          {/* User Profile */}
          <section style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <SectionHeader icon={<User size={14} />} title="User Profile" />
            <Input label="Name" placeholder="Your name" defaultValue={user?.name || ''} />
            <Input label="Email" type="email" placeholder="you@example.com" defaultValue={user?.email || ''} />
            <Button variant="primary" size="sm">Save changes</Button>
          </section>

          {/* Password */}
          <section style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <SectionHeader icon={<Key size={14} />} title="Password" />
            <Input label="Current password" type="password" />
            <Input label="New password" type="password" />
            <Button variant="secondary" size="sm">Reset password</Button>
          </section>

          {/* MCP Connections */}
          <section style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <SectionHeader icon={<Plug size={14} />} title="MCP Connections" />
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {MCP_SERVICES.map((svc) => (
                <div
                  key={svc.name}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '8px 12px',
                    background: 'var(--bg-surface)',
                    borderRadius: 'var(--radius-sm)',
                    border: '1px solid var(--border)',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span
                      style={{
                        width: 8,
                        height: 8,
                        borderRadius: '50%',
                        background: svc.connected ? 'var(--success)' : 'var(--text-dim)',
                      }}
                    />
                    <span style={{ fontSize: 12, color: 'var(--text)' }}>{svc.name}</span>
                  </div>
                  <Button variant="ghost" size="sm">
                    {svc.connected ? 'Disconnect' : 'Connect'}
                  </Button>
                </div>
              ))}
            </div>
          </section>

          {/* Subscription */}
          <section style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <SectionHeader icon={<CreditCard size={14} />} title="Subscription" />
            <div
              style={{
                padding: 14,
                background: 'var(--bg-surface)',
                borderRadius: 'var(--radius-sm)',
                border: '1px solid var(--border)',
                display: 'flex',
                flexDirection: 'column',
                gap: 10,
              }}
            >
              {/* Plan + status */}
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-bright)' }}>
                    {subscription?.plan === 'pro' ? 'Pro Plan' : 'Free Plan'}
                  </span>
                  <span
                    style={{
                      padding: '2px 6px',
                      fontSize: 9,
                      fontWeight: 600,
                      background: subscription?.status === 'trialing' ? 'var(--info)' : 'var(--accent)',
                      color: '#fff',
                      borderRadius: 4,
                    }}
                  >
                    {subscription?.status === 'trialing' ? 'Trial' : 'Active'}
                  </span>
                </div>
              </div>

              {/* Price + billing */}
              {subscription?.plan === 'pro' ? (
                <p style={{ fontSize: 11, color: 'var(--text-muted)', margin: 0 }}>
                  $19/month
                  {subscription?.status === 'trialing' && subscription?.trial_end
                    ? ` \u00B7 Trial ends ${new Date(subscription.trial_end).toLocaleDateString()}`
                    : subscription?.current_period_end
                      ? ` \u00B7 Next billing: ${new Date(subscription.current_period_end).toLocaleDateString()}`
                      : ''}
                </p>
              ) : (
                <p style={{ fontSize: 11, color: 'var(--text-muted)', margin: 0 }}>
                  $0/month \u00B7 Upgrade for team features and cloud deployments
                </p>
              )}

              {/* Payment method */}
              {subscription?.plan === 'pro' && (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', paddingTop: 8, borderTop: '1px solid var(--border)' }}>
                  <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                    Visa \u2022\u2022\u2022\u2022 4242
                  </span>
                  <Button variant="ghost" size="sm">Change</Button>
                </div>
              )}
            </div>
          </section>

          {/* Danger zone */}
          <section
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: 12,
              paddingTop: 16,
              borderTop: '1px solid var(--border)',
            }}
          >
            <DangerButton icon={<LogOut size={12} />} label="Logout" onClick={handleLogout} />
            <DangerButton icon={<Trash2 size={12} />} label="Delete account" />
          </section>
        </div>
      </div>
    </>
  )
}

function SectionHeader({ icon, title }: { icon: React.ReactNode; title: string }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        fontSize: 13,
        fontWeight: 600,
        color: 'var(--text-bright)',
      }}
    >
      {icon}
      {title}
    </div>
  )
}

function DangerButton({ icon, label, onClick }: { icon: React.ReactNode; label: string; onClick?: () => void }) {
  return (
    <button
      onClick={onClick}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        fontSize: 12,
        color: 'var(--text-muted)',
        background: 'transparent',
        border: 'none',
        cursor: 'pointer',
        padding: 0,
        fontFamily: 'var(--font-body)',
      }}
      onMouseEnter={(e) => { e.currentTarget.style.color = 'var(--error)' }}
      onMouseLeave={(e) => { e.currentTarget.style.color = 'var(--text-muted)' }}
    >
      {icon}
      {label}
    </button>
  )
}
