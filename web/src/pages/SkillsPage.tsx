import { useState, useEffect, useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import Header from '../components/Header'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog } from '../components/ui/dialog'
import { skillsApi, type SkillInfo } from '../lib/api'

const TYPE_BADGE: Record<string, string> = {
  builtin: 'bg-blue-100 text-blue-700',
  http: 'bg-green-100 text-green-700',
  composite: 'bg-purple-100 text-purple-700',
}

function TypeBadge({ type }: { type?: string }) {
  if (!type) return null
  const cls = TYPE_BADGE[type] ?? 'bg-gray-100 text-gray-600'
  return (
    <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${cls}`}>
      {type}
    </span>
  )
}

interface SkillCardProps {
  skill: SkillInfo
  onInstall: (name: string) => void
  onUninstall: (name: string) => void
  installing: boolean
  uninstalling: boolean
}

function SkillCard({ skill, onInstall, onUninstall, installing, uninstalling }: SkillCardProps) {
  const { t } = useTranslation()
  const [confirmOpen, setConfirmOpen] = useState(false)

  return (
    <>
      <div className="bg-white rounded-lg border border-gray-200 p-4 flex flex-col gap-2">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <p className="font-semibold text-gray-900 text-sm truncate">{skill.name}</p>
            <span className="text-xs text-gray-400 font-mono">v{skill.version}</span>
          </div>
          <TypeBadge type={skill.type} />
        </div>
        <p className="text-xs text-gray-600 leading-relaxed line-clamp-2">{skill.description}</p>
        <div className="pt-1">
          {skill.installed ? (
            <Button
              variant="outline"
              size="sm"
              className="text-red-600 border-red-300 hover:bg-red-50"
              onClick={() => setConfirmOpen(true)}
              loading={uninstalling}
            >
              {t('skills.uninstall')}
            </Button>
          ) : (
            <Button
              variant="default"
              size="sm"
              onClick={() => onInstall(skill.name)}
              loading={installing}
            >
              {t('skills.install')}
            </Button>
          )}
        </div>
      </div>

      <Dialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        title={t('skills.uninstall')}
      >
        <p className="text-sm text-gray-700 mb-4">
          {t('skills.confirmUninstall', { name: skill.name })}
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={() => setConfirmOpen(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            loading={uninstalling}
            onClick={() => {
              setConfirmOpen(false)
              onUninstall(skill.name)
            }}
          >
            {t('skills.uninstall')}
          </Button>
        </div>
      </Dialog>
    </>
  )
}

export default function SkillsPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [inputValue, setInputValue] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Debounce: update searchQuery 300ms after input stops
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => setSearchQuery(inputValue.trim()), 300)
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [inputValue])

  const { data: listData, isLoading: listLoading, isError: listError } = useQuery({
    queryKey: ['skills'],
    queryFn: skillsApi.list,
  })

  const { data: searchData } = useQuery({
    queryKey: ['skills-search', searchQuery],
    queryFn: () => skillsApi.search(searchQuery),
    enabled: searchQuery.length > 0,
  })

  const installMutation = useMutation({
    mutationFn: (name: string) => skillsApi.install(name),
    onSuccess: (_data, name) => {
      qc.invalidateQueries({ queryKey: ['skills'] })
      qc.invalidateQueries({ queryKey: ['skills-search'] })
      toast.success(t('skills.installSuccess', { name }))
    },
    onError: (err: Error, name) => {
      toast.error(`Failed to install ${name}: ${err.message}`)
    },
  })

  const uninstallMutation = useMutation({
    mutationFn: (name: string) => skillsApi.uninstall(name),
    onSuccess: (_data, name) => {
      qc.invalidateQueries({ queryKey: ['skills'] })
      qc.invalidateQueries({ queryKey: ['skills-search'] })
      toast.success(t('skills.uninstallSuccess', { name }))
    },
    onError: (err: Error, name) => {
      toast.error(`Failed to uninstall ${name}: ${err.message}`)
    },
  })

  const installed = listData?.skills ?? []
  const searchResults = searchData?.results ?? []

  return (
    <div className="flex flex-col h-full">
      <Header
        title={t('skills.title')}
        actions={
          <Input
            type="search"
            placeholder={t('skills.search')}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            className="w-56"
          />
        }
      />

      <div className="flex-1 overflow-auto p-6 space-y-8">
        {/* Installed skills section */}
        <section>
          <h2 className="text-sm font-semibold text-gray-700 mb-3">{t('skills.installed')}</h2>

          {listLoading && (
            <div className="flex items-center gap-2 text-sm text-gray-500">
              <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
              </svg>
              Loading…
            </div>
          )}

          {listError && (
            <div className="rounded-md bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
              Failed to load installed skills. Please try again.
            </div>
          )}

          {!listLoading && !listError && installed.length === 0 && (
            <p className="text-sm text-gray-400 italic">{t('skills.empty')}</p>
          )}

          {installed.length > 0 && (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {installed.map((skill) => (
                <SkillCard
                  key={skill.name}
                  skill={{ ...skill, installed: true }}
                  onInstall={(name) => installMutation.mutate(name)}
                  onUninstall={(name) => uninstallMutation.mutate(name)}
                  installing={installMutation.isPending && installMutation.variables === skill.name}
                  uninstalling={uninstallMutation.isPending && uninstallMutation.variables === skill.name}
                />
              ))}
            </div>
          )}
        </section>

        {/* Search results section — only shown when there are results */}
        {searchQuery.length > 0 && (
          <section>
            <h2 className="text-sm font-semibold text-gray-700 mb-3">{t('skills.available')}</h2>

            {searchResults.length === 0 ? (
              <p className="text-sm text-gray-400 italic">No results for &ldquo;{searchQuery}&rdquo;.</p>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {searchResults.map((skill) => (
                  <SkillCard
                    key={skill.name}
                    skill={{ ...skill, installed: false }}
                    onInstall={(name) => installMutation.mutate(name)}
                    onUninstall={(name) => uninstallMutation.mutate(name)}
                    installing={installMutation.isPending && installMutation.variables === skill.name}
                    uninstalling={uninstallMutation.isPending && uninstallMutation.variables === skill.name}
                  />
                ))}
              </div>
            )}
          </section>
        )}
      </div>
    </div>
  )
}
