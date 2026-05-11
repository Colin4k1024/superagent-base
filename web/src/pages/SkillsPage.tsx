import Header from '../components/Header'

const placeholderSkills = [
  { name: 'web_search', version: '1.0.0', description: 'Search the web for information.', installed: true },
  { name: 'http_request', version: '1.0.0', description: 'Make HTTP requests to external APIs.', installed: true },
  { name: 'code_execute', version: '1.0.0', description: 'Execute code snippets in a sandbox.', installed: true },
  { name: 'pdf_reader', version: '0.9.0', description: 'Extract text and metadata from PDF files.', installed: false },
  { name: 'image_gen', version: '0.8.0', description: 'Generate images from text prompts.', installed: false },
]

export default function SkillsPage() {
  return (
    <div className="flex flex-col h-full">
      <Header
        title="Skills"
        actions={
          <input
            type="search"
            placeholder="Search skills…"
            className="px-3 py-1.5 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        }
      />

      <div className="flex-1 overflow-auto p-6">
        <p className="text-sm text-gray-500 mb-4">
          Browse and manage skills from the SkillsHub marketplace.
        </p>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {placeholderSkills.map((skill) => (
            <div key={skill.name} className="bg-white rounded-lg border border-gray-200 p-4 flex flex-col gap-2">
              <div className="flex items-start justify-between">
                <div>
                  <p className="font-medium text-gray-900 text-sm">{skill.name}</p>
                  <p className="text-xs text-gray-400 font-mono">v{skill.version}</p>
                </div>
                <span
                  className={`text-xs px-2 py-0.5 rounded-full font-medium ${
                    skill.installed
                      ? 'bg-green-100 text-green-700'
                      : 'bg-gray-100 text-gray-500'
                  }`}
                >
                  {skill.installed ? 'Installed' : 'Available'}
                </span>
              </div>
              <p className="text-xs text-gray-600 leading-relaxed">{skill.description}</p>
              <div className="pt-1">
                {skill.installed ? (
                  <button className="text-xs text-red-600 hover:underline">Uninstall</button>
                ) : (
                  <button className="text-xs text-blue-600 hover:underline">Install</button>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
