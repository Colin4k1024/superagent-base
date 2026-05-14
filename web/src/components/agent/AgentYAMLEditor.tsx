import Editor from '@monaco-editor/react'

interface Props {
  value: string
  onChange: (value: string) => void
  readOnly?: boolean
}

export function AgentYAMLEditor({ value, onChange, readOnly }: Props) {
  return (
    <Editor
      height="100%"
      language="yaml"
      theme="vs-light"
      value={value}
      onChange={(v) => onChange(v || '')}
      options={{
        minimap: { enabled: false },
        lineNumbers: 'on',
        fontSize: 13,
        readOnly,
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        tabSize: 2,
      }}
    />
  )
}
