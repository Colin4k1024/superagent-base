interface FileMessageProps {
  files: Array<{
    name: string
    size?: number
    type?: string
    url?: string
  }>
}

function formatFileSize(bytes?: number): string {
  if (!bytes) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function getFileIcon(type?: string): string {
  if (!type) return '📄'
  if (type.startsWith('image/')) return '🖼️'
  if (type.includes('pdf')) return '📕'
  if (type.includes('word') || type.includes('doc')) return '📘'
  if (type.includes('excel') || type.includes('sheet') || type.includes('csv')) return '📊'
  if (type.includes('zip') || type.includes('rar')) return '📦'
  if (type.includes('text')) return '📝'
  return '📄'
}

export default function FileMessage({ files }: FileMessageProps) {
  if (!files || files.length === 0) return null

  return (
    <div className="space-y-2 my-1">
      {files.map((file, i) => (
        <div
          key={i}
          className="flex items-center gap-3 px-3 py-2.5 rounded-lg border border-gray-200 bg-gray-50 hover:bg-gray-100 transition-colors max-w-xs"
        >
          <span className="text-xl flex-shrink-0">{getFileIcon(file.type)}</span>
          <div className="flex-1 min-w-0">
            <div className="text-sm text-gray-800 truncate font-medium">{file.name}</div>
            {file.size && (
              <div className="text-xs text-gray-400">{formatFileSize(file.size)}</div>
            )}
          </div>
          {file.url && (
            <a
              href={file.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-500 hover:text-blue-700 text-xs flex-shrink-0"
            >
              下载
            </a>
          )}
        </div>
      ))}
    </div>
  )
}
