interface Reference {
  title: string
  url?: string
  source?: string
}

interface CardMessageProps {
  title?: string
  content: string
  references?: Reference[]
}

export default function CardMessage({ title, content, references }: CardMessageProps) {
  return (
    <div className="my-2 rounded-lg border border-gray-200 overflow-hidden bg-white">
      {title && (
        <div className="px-4 py-2.5 border-b border-gray-100 bg-gray-50">
          <h4 className="text-sm font-medium text-gray-800">{title}</h4>
        </div>
      )}
      <div className="px-4 py-3 text-sm text-gray-700 whitespace-pre-wrap">
        {content}
      </div>
      {references && references.length > 0 && (
        <div className="px-4 py-2.5 border-t border-gray-100 bg-gray-50">
          <div className="text-[10px] text-gray-400 uppercase mb-1.5">引用来源</div>
          <div className="space-y-1">
            {references.map((ref, i) => (
              <div key={i} className="flex items-center gap-1.5 text-xs">
                <span className="text-gray-400">[{i + 1}]</span>
                {ref.url ? (
                  <a
                    href={ref.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 hover:text-blue-800 truncate"
                  >
                    {ref.title}
                  </a>
                ) : (
                  <span className="text-gray-600 truncate">{ref.title}</span>
                )}
                {ref.source && (
                  <span className="text-gray-400 flex-shrink-0">· {ref.source}</span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
