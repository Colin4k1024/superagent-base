/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { AgentLoopMessage } from '@/lib/agentloop-types'
import ThinkingBlock from '@/components/chat/ThinkingBlock'
import ToolCallBlock from '@/components/chat/ToolCallBlock'
import MarkdownRenderer from '@/components/chat/MarkdownRenderer'
import TurnSeparator from './TurnSeparator'
import InterruptForm from './InterruptForm'

interface Props {
  message: AgentLoopMessage
  isLast: boolean
  onResume?: (values: Record<string, string>) => void
}

export default function AgentLoopMessageItem({ message, isLast, onResume }: Props) {
  // User message
  if (message.role === 'user') {
    return (
      <div className="flex justify-end">
        <div className="max-w-[70%] bg-blue-600 text-white rounded-2xl rounded-br-md px-4 py-2.5 text-sm">
          {message.content}
        </div>
      </div>
    )
  }

  // Assistant message
  return (
    <div className="flex flex-col gap-0">
      {/* Preempted banner */}
      {message.status === 'preempted' && (
        <div className="mb-2 px-3 py-1.5 bg-amber-50 border border-amber-200 rounded-md text-xs text-amber-700">
          Execution interrupted
        </div>
      )}

      {/* Error banner */}
      {message.status === 'error' && message.errorMessage && (
        <div className="mb-2 px-3 py-1.5 bg-red-50 border border-red-200 rounded-md text-xs text-red-700">
          Error: {message.errorMessage}
        </div>
      )}

      {/* Turns */}
      {message.turns.map((turn, i) => (
        <div key={turn.turn}>
          {/* Turn separator (skip for first turn if it's turn 1) */}
          {i > 0 && (
            <TurnSeparator
              turn={turn.turn}
              total={turn.total}
              startTime={turn.startTime}
              endTime={turn.endTime}
              isStreaming={turn.isStreaming}
            />
          )}

          {/* Thinking blocks */}
          {turn.thinking && (
            <ThinkingBlock content={turn.thinking} isStreaming={turn.isStreaming && isLast} />
          )}

          {/* Tool calls */}
          {turn.toolCalls.length > 0 && (
            <div>
              {turn.toolCalls.map((tc) => (
                <ToolCallBlock
                  key={tc.id}
                  toolName={tc.name}
                  args={tc.args}
                  result={tc.result}
                  status={tc.status}
                />
              ))}
            </div>
          )}

          {/* Text content */}
          {turn.text && (
            <div className="prose prose-sm max-w-none">
              <MarkdownRenderer content={turn.text.replace(/\[DONE\]/g, '')} />
              {turn.isStreaming && isLast && (
                <span className="inline-block w-0.5 h-4 bg-gray-900 animate-pulse ml-0.5" />
              )}
            </div>
          )}
        </div>
      ))}

      {/* Interrupt form */}
      {message.interrupt && onResume && (
        <InterruptForm
          reason={message.interrupt.reason}
          fields={message.interrupt.fields}
          onSubmit={onResume}
        />
      )}
    </div>
  )
}
