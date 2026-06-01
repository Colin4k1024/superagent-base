import { useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'

interface ChatInputProps {
  value: string
  onChange: (value: string) => void
  onSend: () => void
  onStop?: () => void
  isLoading: boolean
  disabled?: boolean
}

export default function ChatInput({ value, onChange, onSend, onStop, isLoading, disabled }: ChatInputProps) {
  const { t } = useTranslation()
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        if (!disabled && value.trim()) {
          onSend()
        }
      }
    },
    [disabled, value, onSend],
  )

  const handleInput = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      onChange(e.target.value)
      // Auto-resize
      const el = e.target
      el.style.height = 'auto'
      el.style.height = Math.min(el.scrollHeight, 160) + 'px'
    },
    [onChange],
  )

  return (
    <div className="border-t border-gray-200 bg-white px-4 py-3">
      <div className="max-w-[840px] mx-auto">
        <div className="relative flex items-end gap-2 rounded-xl border border-gray-300 bg-white px-4 py-2 focus-within:ring-2 focus-within:ring-blue-500 focus-within:border-blue-500 transition-shadow">
          <textarea
            ref={textareaRef}
            className="flex-1 resize-none text-sm leading-relaxed outline-none placeholder-gray-400 min-h-[36px] max-h-[160px] py-1"
            rows={1}
            placeholder={t('chat.placeholder')}
            value={value}
            onChange={handleInput}
            onKeyDown={handleKeyDown}
            disabled={disabled}
          />
          {isLoading && value.trim() && (
            <button
              onClick={onSend}
              disabled={disabled}
              className="flex-shrink-0 w-8 h-8 flex items-center justify-center rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-gray-300 text-white transition-colors disabled:cursor-not-allowed"
              title="发送"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </button>
          )}
          {isLoading ? (
            <button
              onClick={onStop}
              className="flex-shrink-0 w-8 h-8 flex items-center justify-center rounded-lg bg-red-500 hover:bg-red-600 text-white transition-colors"
              title="停止生成"
            >
              <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                <rect x="6" y="6" width="12" height="12" rx="2" />
              </svg>
            </button>
          ) : (
            <button
              onClick={onSend}
              disabled={!value.trim() || disabled}
              className="flex-shrink-0 w-8 h-8 flex items-center justify-center rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-gray-300 text-white transition-colors disabled:cursor-not-allowed"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </button>
          )}
        </div>
        <div className="mt-1.5 text-[11px] text-gray-400 text-center">
          Enter 发送 · Shift+Enter 换行
        </div>
      </div>
    </div>
  )
}
