export type RunID = string
export type SessionID = string

export interface DaemonStatus {
  bind: string
  endpoint: string
  modelDir: string
  llamaServerHealthy: boolean
  llamaServerRunning: boolean
  llamaServerPid: number
  uptimeSeconds: number
  startedAtRfc3339: string
  dataDir: string
}

export interface AgentSummary {
  name: string
  displayName: string
  hasPrompt: boolean
  hasConfig: boolean
}

export interface SamplingConfig {
  temperature?: number
  topP?: number
  topK?: number
  repeatPenalty?: number
  maxTokens?: number
}

export interface AgentConfig {
  name: string
  displayName: string
  systemPrompt: string
  model: string
  sampling?: SamplingConfig
  toolRole: boolean
}

export type RunStatus = 'started' | 'completed' | 'cancelled' | 'failed'

export interface RunRecord {
  runId: RunID
  sessionId: SessionID
  status: RunStatus
  timestamp: string
}

export interface Skill {
  name: string
  description: string
  content: string
  path: string
  disableModelInvocation: boolean
  userInvocable: boolean
}

export type Role = 'system' | 'user' | 'assistant' | 'tool'

export interface ToolCall {
  id: string
  name: string
  arguments: string
}

export interface ToolResult {
  toolCallId: string
  content: string
}

export interface Message {
  role: Role
  content: string
  toolCalls?: ToolCall[]
  toolResult?: ToolResult
  agentName: string
  tokens: number
}

export interface Session {
  id: SessionID
  agentName: string
  createdAt: string
  lastActiveAt: string
  messages: Message[]
}
