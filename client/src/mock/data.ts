import { ref } from 'vue'
import type {
  DaemonStatus,
  AgentSummary,
  AgentConfig,
  RunRecord,
  Skill,
  Session,
} from '../types/kontekst'

export const mockDaemonStatus: DaemonStatus = {
  bind: ':50051',
  endpoint: 'http://127.0.0.1:8080',
  modelDir: '~/models',
  llamaServerHealthy: true,
  llamaServerRunning: true,
  llamaServerPid: 12345,
  uptimeSeconds: 3600,
  startedAtRfc3339: '2026-02-04T10:00:00Z',
  dataDir: '~/.kontekst',
}

export const mockAgentSummaries = ref<AgentSummary[]>([
  {
    name: 'default',
    displayName: 'Default Agent',
    hasPrompt: true,
    hasConfig: true,
  },
  {
    name: 'coder',
    displayName: 'Code Assistant',
    hasPrompt: true,
    hasConfig: true,
  },
  {
    name: 'researcher',
    displayName: 'Research Agent',
    hasPrompt: true,
    hasConfig: false,
  },
])

export const mockAgentConfigs: Record<string, AgentConfig> = {
  default: {
    name: 'default',
    displayName: 'Default Agent',
    systemPrompt:
      'You are a helpful AI assistant. Answer questions accurately and concisely.',
    model: 'gpt-oss-20b-Q4_K_M.gguf',
    sampling: {
      temperature: 0.7,
      topP: 0.9,
      topK: 40,
      repeatPenalty: 1.1,
      maxTokens: 4096,
    },
    toolRole: true,
  },
  coder: {
    name: 'coder',
    displayName: 'Code Assistant',
    systemPrompt:
      'You are an expert software engineer. Help with coding tasks, debugging, and code review.',
    model: 'codestral-22b-Q4_K_M.gguf',
    sampling: {
      temperature: 0.3,
      topP: 0.95,
      maxTokens: 8192,
    },
    toolRole: true,
  },
  researcher: {
    name: 'researcher',
    displayName: 'Research Agent',
    systemPrompt:
      'You are a research assistant. Help gather, analyze, and synthesize information.',
    model: 'gpt-oss-20b-Q4_K_M.gguf',
    toolRole: false,
  },
}

export const mockRunRecords: RunRecord[] = [
  {
    runId: 'run-001',
    sessionId: 'session-abc',
    status: 'completed',
    timestamp: '2026-02-04T14:30:00Z',
  },
  {
    runId: 'run-002',
    sessionId: 'session-abc',
    status: 'completed',
    timestamp: '2026-02-04T14:35:00Z',
  },
  {
    runId: 'run-003',
    sessionId: 'session-def',
    status: 'started',
    timestamp: '2026-02-04T15:00:00Z',
  },
  {
    runId: 'run-004',
    sessionId: 'session-ghi',
    status: 'failed',
    timestamp: '2026-02-04T12:00:00Z',
  },
  {
    runId: 'run-005',
    sessionId: 'session-ghi',
    status: 'cancelled',
    timestamp: '2026-02-04T12:05:00Z',
  },
]

export const mockSessions = ref<Session[]>([
  {
    id: 'session-abc',
    agentName: 'default',
    createdAt: '2026-02-04T14:00:00Z',
    lastActiveAt: '2026-02-04T14:35:00Z',
    messages: [
      {
        role: 'system',
        content:
          'You are a helpful AI assistant. Answer questions accurately and concisely.',
        agentName: 'default',
        tokens: 20,
      },
      {
        role: 'user',
        content: 'What is the capital of France?',
        agentName: 'default',
        tokens: 8,
      },
      {
        role: 'assistant',
        content:
          'The capital of France is Paris. It is the largest city in France and serves as the country\'s political, economic, and cultural center.',
        agentName: 'default',
        tokens: 35,
      },
    ],
  },
  {
    id: 'session-def',
    agentName: 'coder',
    createdAt: '2026-02-04T15:00:00Z',
    lastActiveAt: '2026-02-04T15:00:00Z',
    messages: [
      {
        role: 'system',
        content:
          'You are an expert software engineer. Help with coding tasks, debugging, and code review.',
        agentName: 'coder',
        tokens: 22,
      },
      {
        role: 'user',
        content: 'Help me write a function to reverse a string in Go.',
        agentName: 'coder',
        tokens: 12,
      },
    ],
  },
  {
    id: 'session-ghi',
    agentName: 'researcher',
    createdAt: '2026-02-04T12:00:00Z',
    lastActiveAt: '2026-02-04T12:05:00Z',
    messages: [
      {
        role: 'system',
        content:
          'You are a research assistant. Help gather, analyze, and synthesize information.',
        agentName: 'researcher',
        tokens: 18,
      },
    ],
  },
])

export const mockSkills = ref<Skill[]>([
  {
    name: 'commit',
    description: 'Generate a git commit message based on staged changes',
    content:
      'Analyze the staged git changes and generate an appropriate commit message following conventional commit format.',
    path: '~/.kontekst/skills/commit.md',
    disableModelInvocation: false,
    userInvocable: true,
  },
  {
    name: 'review',
    description: 'Review code changes and provide feedback',
    content:
      'Review the provided code changes for bugs, security issues, and best practices. Provide constructive feedback.',
    path: '~/.kontekst/skills/review.md',
    disableModelInvocation: false,
    userInvocable: true,
  },
  {
    name: 'explain',
    description: 'Explain code or concepts in detail',
    content:
      'Provide a detailed explanation of the given code or concept. Break down complex ideas into understandable parts.',
    path: '~/.kontekst/skills/explain.md',
    disableModelInvocation: false,
    userInvocable: true,
  },
  {
    name: 'summarize',
    description: 'Summarize long text or documents',
    content: 'Create a concise summary of the provided text or document.',
    path: '~/.kontekst/skills/summarize.md',
    disableModelInvocation: true,
    userInvocable: false,
  },
])

export function getAgentConfig(name: string): AgentConfig | undefined {
  return mockAgentConfigs[name]
}

export function deleteAgent(name: string) {
  mockAgentSummaries.value = mockAgentSummaries.value.filter((a) => a.name !== name)
  delete mockAgentConfigs[name]
}

export function getSession(id: string): Session | undefined {
  return mockSessions.value.find((s) => s.id === id)
}

export function deleteSession(id: string) {
  mockSessions.value = mockSessions.value.filter((s) => s.id !== id)
}

export function getSkill(name: string): Skill | undefined {
  return mockSkills.value.find((s) => s.name === name)
}

export function deleteSkill(name: string) {
  mockSkills.value = mockSkills.value.filter((s) => s.name !== name)
}

export function getRunsForSession(sessionId: string): RunRecord[] {
  return mockRunRecords.filter((r) => r.sessionId === sessionId)
}
