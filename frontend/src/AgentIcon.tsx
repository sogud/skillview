import { Bot, Layers, Folder } from 'lucide-react'
import logo0 from '@lobehub/icons-static-svg/icons/claudecode-color.svg?url'
import logo1 from '@lobehub/icons-static-svg/icons/codex-color.svg?url'
import logo2 from '@lobehub/icons-static-svg/icons/cursor.svg?url'
import logo3 from '@lobehub/icons-static-svg/icons/windsurf.svg?url'
import logo4 from '@lobehub/icons-static-svg/icons/devin.svg?url'
import logo5 from '@lobehub/icons-static-svg/icons/geminicli-color.svg?url'
import logo6 from '@lobehub/icons-static-svg/icons/githubcopilot.svg?url'
import logo7 from '@lobehub/icons-static-svg/icons/opencode.svg?url'
import logo8 from '@lobehub/icons-static-svg/icons/pi.svg?url'
import logo9 from '@lobehub/icons-static-svg/icons/qoder.svg?url'
import logo10 from '@lobehub/icons-static-svg/icons/qwen-color.svg?url'
import logo11 from '@lobehub/icons-static-svg/icons/kilocode.svg?url'
import logo12 from '@lobehub/icons-static-svg/icons/roocode.svg?url'
import logo13 from '@lobehub/icons-static-svg/icons/trae.svg?url'
import logo14 from '@lobehub/icons-static-svg/icons/antigravity-color.svg?url'
import logo15 from '@lobehub/icons-static-svg/icons/kimi.svg?url'
import logo16 from '@lobehub/icons-static-svg/icons/kiro-color.svg?url'
import logo17 from '@lobehub/icons-static-svg/icons/cline.svg?url'
import logo18 from '@lobehub/icons-static-svg/icons/goose.svg?url'
import logo19 from '@lobehub/icons-static-svg/icons/grok.svg?url'
import logo20 from '@lobehub/icons-static-svg/icons/mistral-color.svg?url'
import logo21 from '@lobehub/icons-static-svg/icons/minimax-color.svg?url'
import logo22 from '@lobehub/icons-static-svg/icons/amp.svg?url'
import logo23 from '@lobehub/icons-static-svg/icons/openclaw.svg?url'
import logo24 from '@lobehub/icons-static-svg/icons/zencoder.svg?url'
import hermes from '@lobehub/icons-static-svg/icons/hermesagent.svg?url'
import zed from './assets/zed.svg?url'
import vscode from './assets/vscode.svg?url'

const logos: Record<string, string> = {
  'claudecode-color': logo0,
  'codex-color': logo1,
  cursor: logo2,
  windsurf: logo3,
  devin: logo4,
  'geminicli-color': logo5,
  githubcopilot: logo6,
  opencode: logo7,
  pi: logo8,
  qoder: logo9,
  'qwen-color': logo10,
  kilocode: logo11,
  roocode: logo12,
  trae: logo13,
  'antigravity-color': logo14,
  kimi: logo15,
  'kiro-color': logo16,
  cline: logo17,
  goose: logo18,
  grok: logo19,
  'mistral-color': logo20,
  'minimax-color': logo21,
  amp: logo22,
  openclaw: logo23,
  zencoder: logo24,
  hermes: hermes,
  zed: zed,
  'vscode-color': vscode,
}

export function AgentIcon({
  icon,
  name,
  size = 14,
}: {
  icon?: string
  name: string
  size?: number
}) {
  const src = icon ? logos[icon] : undefined
  if (src)
    return (
      <img
        className={`agent-icon${icon?.endsWith('-color') ? '' : ' mono'}`}
        src={src}
        alt=""
        aria-hidden
        draggable={false}
        style={{ width: size, height: size }}
      />
    )
  const Icon =
    name === 'Shared' || name === 'Shared Skills'
      ? Layers
      : name === 'Local'
        ? Folder
        : Bot
  return <Icon className="agent-icon" size={size} aria-hidden />
}
