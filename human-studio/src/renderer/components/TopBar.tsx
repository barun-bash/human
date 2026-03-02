import React from 'react'
import {
  GitBranch,
  Play,
  Square,
  Hammer,
  CheckCircle,
  Rocket,
  Sun,
  Moon,
  ChevronDown,
} from 'lucide-react'
import { Button } from './ui/Button'
import { Badge } from './ui/Badge'
import { Avatar } from './ui/Avatar'
import { useProjectStore } from '../stores/project'
import { useSettingsStore } from '../stores/settings'
import { useBuildStore } from '../stores/build'
import { useAuthStore } from '../stores/auth'

interface TopBarProps {
  onCheck: () => void
  onBuild: () => void
  onRun: () => void
  onDeploy: () => void
  onStop: () => void
  onOpenProfile: () => void
  onConfigureKeys: () => void
}

export function TopBar({
  onCheck,
  onBuild,
  onRun,
  onDeploy,
  onStop,
  onOpenProfile,
  onConfigureKeys,
}: TopBarProps) {
  const projectName = useProjectStore((s) => s.projectName)
  const { llmProvider, setLLMProvider, theme, toggleTheme } = useSettingsStore()
  const buildStatus = useBuildStore((s) => s.status)
  const userName = useAuthStore((s) => s.user?.name)

  const isRunning = buildStatus === 'checking' || buildStatus === 'building' || buildStatus === 'running' || buildStatus === 'deploying'

  const isMac = navigator.platform.includes('Mac')

  return (
    <div
      className="titlebar-drag"
      style={{
        height: 48,
        display: 'flex',
        alignItems: 'center',
        paddingLeft: isMac ? 80 : 16, // Leave space for macOS traffic lights
        paddingRight: 16,
        gap: 12,
        borderBottom: '1px solid var(--border)',
        background: 'var(--bg-raised)',
        flexShrink: 0,
      }}
    >
      {/* Left: Logo + Project */}
      <div className="titlebar-no-drag" style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        {/* Logo */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <svg width="24" height="24" viewBox="0 0 120 120">
            <rect width="120" height="120" rx="24" fill="#0D0D0D" />
            <text x="24" y="84" fontFamily="Nunito, sans-serif" fontWeight="700" fontSize="72" letterSpacing="-1">
              <tspan fill="#F5F5F3">h</tspan>
              <tspan fill="#E85D3A" className="cursor-blink">_</tspan>
            </text>
          </svg>
          <span style={{ fontSize: 15, fontWeight: 700, color: 'var(--text-bright)', fontFamily: 'var(--font-logo)' }}>
            Human
          </span>
          <Badge variant="default">v0.1</Badge>
        </div>

        {/* Divider */}
        <div style={{ width: 1, height: 20, background: 'var(--border)' }} />

        {/* Project name */}
        <span style={{ fontSize: 12, color: 'var(--text-muted)', maxWidth: 150, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {projectName || 'No project'}
        </span>
      </div>

      {/* Center spacer */}
      <div style={{ flex: 1 }} />

      {/* Right: Actions */}
      <div className="titlebar-no-drag" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        {/* Git */}
        <Button variant="ghost" size="sm">
          <GitBranch size={14} />
          main
        </Button>

        {/* LLM Provider */}
        <Button variant="ghost" size="sm">
          {llmProvider === 'anthropic' ? 'Anthropic Claude' : llmProvider}
          <ChevronDown size={12} />
        </Button>

        {/* Theme toggle */}
        <button
          onClick={toggleTheme}
          style={{
            padding: 6,
            color: 'var(--text-muted)',
            background: 'transparent',
            border: 'none',
            borderRadius: 'var(--radius-sm)',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
          }}
          title="Toggle theme"
        >
          {theme === 'dark' ? <Sun size={15} /> : <Moon size={15} />}
        </button>

        {/* Divider */}
        <div style={{ width: 1, height: 20, background: 'var(--border)' }} />

        {/* Check / Build / Run / Deploy / Stop */}
        {isRunning ? (
          <Button variant="danger" size="sm" onClick={onStop}>
            <Square size={13} />
            Stop
          </Button>
        ) : (
          <>
            <Button variant="info" size="sm" onClick={onCheck}>
              <CheckCircle size={13} />
              Check
            </Button>
            <Button variant="primary" size="sm" onClick={onBuild}>
              <Hammer size={13} />
              Build
            </Button>
            <Button variant="success" size="sm" onClick={onRun}>
              <Play size={13} />
              Run
            </Button>
            <Button variant="ghost" size="sm" onClick={onDeploy} title="Deploy with Docker">
              <Rocket size={13} />
            </Button>
          </>
        )}

        {/* Divider */}
        <div style={{ width: 1, height: 20, background: 'var(--border)' }} />

        {/* Avatar */}
        <button onClick={onOpenProfile} style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 0 }}>
          <Avatar name={userName || 'User'} size={28} />
        </button>
      </div>
    </div>
  )
}
