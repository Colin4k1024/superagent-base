import type { ChatMessage } from '../../lib/api'
import MarkdownRenderer from './MarkdownRenderer'
import ThinkingBlock from './ThinkingBlock'
import ToolCallBlock from './ToolCallBlock'
import FileMessage from './FileMessage'
import CardMessage from './CardMessage'

interface MessageBubbleProps {
  message: ChatMessage
  isStreaming?: boolean
  isLast?: boolean
}

export default function MessageBubble({ message, isStreaming, isLast }: MessageBubbleProps) {
  const isUser = message.role === 'user'

  return (
    <div className={`flex ${isUser ? 'justify-end' : 'justify-start'} group`}>
      <div
        className={`max-w-[75%] ${
          isUser
            ? 'bg-blue-600 text-white rounded-2xl rounded-br-sm px-4 py-2.5'
            : 'bg-white border border-gray-200 text-gray-800 rounded-2xl rounded-bl-sm px-4 py-3 shadow-sm'
        }`}
      >
        {/* 思考过程 */}
        {message.thinking && (
          <ThinkingBlock
            content={message.thinking}
            steps={message.thinkingSteps}
            isStreaming={isStreaming && isLast && !message.content}
          />
        )}

        {/* 工具调用 */}
        {message.toolCalls && message.toolCalls.map((tc, i) => (
          <ToolCallBlock
            key={i}
            toolName={tc.name}
            args={tc.args}
            result={tc.result}
            status={tc.status}
          />
        ))}

        {/* 文件消息 */}
        {message.files && message.files.length > 0 && (
          <FileMessage files={message.files} />
        )}

        {/* 卡片消息 */}
        {message.card && (
          <CardMessage
            title={message.card.title}
            content={message.card.content}
            references={message.references}
          />
        )}

        {/* 文本内容 */}
        {message.content && (
          isUser ? (
            <span className="text-sm leading-relaxed whitespace-pre-wrap">{message.content}</span>
          ) : (
            <MarkdownRenderer content={message.content} />
          )
        )}

        {/* 流式光标 */}
        {isStreaming && isLast && !isUser && (
          <span className="inline-block w-1.5 h-4 bg-gray-400 animate-pulse ml-0.5 align-middle rounded-sm" />
        )}
      </div>
    </div>
  )
}
