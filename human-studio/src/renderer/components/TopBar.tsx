import React, { useState, useEffect } from 'react'
import {
  GitBranch,
  Play,
  Square,
  Hammer,
  CheckCircle,
  Sun,
  Moon,
} from 'lucide-react'
import { Button } from './ui/Button'
import { Badge } from './ui/Badge'
import { Avatar } from './ui/Avatar'
import { Dropdown } from './ui/Dropdown'
import { useProjectStore } from '../stores/project'
import { useSettingsStore } from '../stores/settings'
import { useBuildStore } from '../stores/build'
import { useAuthStore } from '../stores/auth'
import { api } from '../lib/ipc'

const LLM_ITEMS = [
  { label: 'Anthropic Claude', value: 'anthropic' },
  { label: 'OpenAI GPT-4', value: 'openai' },
  { label: 'Google Gemini', value: 'gemini' },
  { label: 'Ollama (local)', value: 'ollama' },
  { label: 'Groq', value: 'groq' },
  { label: 'OpenRouter', value: 'openrouter' },
  { label: '', value: '', divider: true },
  { label: 'Configure API keys...', value: '__configure' },
]

interface TopBarProps {
  onCheck: () => void
  onBuild: () => void
  onRun: () => void
  onStop: () => void
  onOpenProfile: () => void
  onConfigureKeys: () => void
}

export function TopBar({
  onCheck,
  onBuild,
  onRun,
  onStop,
  onOpenProfile,
  onConfigureKeys,
}: TopBarProps) {
  const projectName = useProjectStore((s) => s.projectName)
  const projectDir = useProjectStore((s) => s.projectDir)
  const { llmProvider, setLLMProvider, theme, toggleTheme } = useSettingsStore()
  const buildStatus = useBuildStore((s) => s.status)
  const userName = useAuthStore((s) => s.user?.name)
  const authLogout = useAuthStore((s) => s.logout)

  const [branch, setBranch] = useState('main')

  const isRunning = buildStatus === 'checking' || buildStatus === 'building' || buildStatus === 'running'

  const isMac = navigator.platform.includes('Mac')

  // Fetch real git branch when project changes
  useEffect(() => {
    if (!projectDir || !api) return
    api.git.branch(projectDir).then((b: string) => {
      if (b) setBranch(b.trim())
    }).catch(() => {})
  }, [projectDir])

  const handleGitAction = (value: string) => {
    if (!projectDir) return
    switch (value) {
      case 'push':
        api.git.push(projectDir)
        break
      case 'pull':
        api.git.pull(projectDir)
        break
      case 'create-branch': {
        const name = window.prompt('New branch name:')
        if (name?.trim()) {
          api.git.createBranch(projectDir, name.trim()).then(() => {
            setBranch(name.trim())
          })
        }
        break
      }
    }
  }

  const handleLLMChange = (value: string) => {
    if (value === '__configure') {
      onConfigureKeys()
      return
    }
    setLLMProvider(value)
  }

  const handleAvatarAction = (value: string) => {
    switch (value) {
      case 'profile':
        onOpenProfile()
        break
      case 'logout':
        api.auth.logout()
        authLogout()
        break
    }
  }

  const llmLabel = LLM_ITEMS.find((i) => i.value === llmProvider)?.label || llmProvider

  return (
    <div
      className="titlebar-drag"
      style={{
        height: 48,
        display: 'flex',
        alignItems: 'center',
        paddingLeft: isMac ? 80 : 16,
        paddingRight: 16,
        gap: 12,
        borderBottom: '1px solid var(--border)',
        background: 'var(--bg-raised)',
        flexShrink: 0,
      }}
    >
      {/* Left: Logo + Project */}
      <div className="titlebar-no-drag" style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <svg width="24" height="24" viewBox="0 0 120 120">
            <rect width="120" height="120" rx="24" fill="var(--bg-surface)" />
            <text x="24" y="84" fontFamily="Nunito, sans-serif" fontWeight="700" fontSize="72" letterSpacing="-1">
              <tspan fill="var(--text-bright)">h</tspan>
              <tspan fill="var(--accent)" className="cursor-blink">_</tspan>
            </text>
          </svg>
          <span style={{ fontSize: 15, fontWeight: 700, color: 'var(--text-bright)', fontFamily: 'var(--font-logo)' }}>
            Human
          </span>
          <Badge variant="default">v0.1</Badge>
        </div>

        <div style={{ width: 1, height: 20, background: 'var(--border)' }} />

        <span style={{ fontSize: 12, color: 'var(--text-muted)', maxWidth: 150, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {projectName || 'No project'}
        </span>
      </div>

      <div style={{ flex: 1 }} />

      {/* Right: Actions */}
      <div className="titlebar-no-drag" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        {/* Git dropdown */}
        <Dropdown
          items={[
            { label: branch, value: '__current', icon: <GitBranch size={12} />, disabled: true },
            { label: '', value: '', divider: true },
            { label: 'Push changes', value: 'push' },
            { label: 'Pull latest', value: 'pull' },
            { label: 'Create branch...', value: 'create-branch' },
          ]}
          value="__current"
          onChange={handleGitAction}
          trigger={
            <span style={{ display: 'flex', alignItems: 'center', gap: 5, fontSize: 12, color: 'var(--text-muted)' }}>
              <GitBranch size={13} />
              {branch}
            </span>
          }
        />

        {/* LLM Provider dropdown */}
        <Dropdown
          items={LLM_ITEMS}
          value={llmProvider}
          onChange={handleLLMChange}
          trigger={
            <span style={{ display: 'flex', alignItems: 'center', gap: 5, fontSize: 12, color: 'var(--text-muted)' }}>
              {llmLabel}
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><polyline points="6 9 12 15 18 9" /></svg>
            </span>
          }
        />

        {/* Theme toggle */}
        <button
          onClick={toggleTheme}
          style={{
            width: 28,
            height: 28,
            color: 'var(--text-muted)',
            background: 'transparent',
            border: '1px solid var(--border)',
            borderRadius: '50%',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
          title="Toggle theme"
        >
          {theme === 'dark' ? <Sun size={14} /> : <Moon size={14} />}
        </button>

        <div style={{ width: 1, height: 20, background: 'var(--border)' }} />

        {/* Check / Build / Run / Stop */}
        {isRunning ? (
          <Button variant="danger" size="sm" onClick={onStop}>
            <Square size={13} />
            Stop
          </Button>
        ) : (
          <>
            <Button variant="secondary" size="sm" onClick={onCheck}>
              <CheckCircle size={13} style={{ color: '#22D3EE' }} />
              Check
            </Button>
            <Button variant="primary" size="sm" onClick={onBuild}>
              <Hammer size={13} />
              Build
            </Button>
            <Button variant="secondary" size="sm" onClick={onRun}>
              <Play size={13} style={{ color: '#2D8C5A' }} />
              Run
            </Button>
          </>
        )}

        <div style={{ width: 1, height: 20, background: 'var(--border)' }} />

        {/* Avatar dropdown */}
        <Dropdown
          items={[
            { label: 'Profile & Settings', value: 'profile' },
            { label: '', value: '', divider: true },
            { label: 'Log out', value: 'logout', danger: true },
          ]}
          value=""
          onChange={handleAvatarAction}
          align="right"
          trigger={<Avatar name={userName || 'User'} size={28} />}
        />
      </div>
    </div>
  )
}
