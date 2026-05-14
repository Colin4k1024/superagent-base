import { ReactNode, createContext, useContext } from 'react'
import { cn } from '../../lib/cn'

interface TabsContextValue {
  value: string
  onChange: (value: string) => void
}

const TabsContext = createContext<TabsContextValue | null>(null)

function useTabs() {
  const ctx = useContext(TabsContext)
  if (!ctx) throw new Error('Tabs components must be used within <Tabs>')
  return ctx
}

interface TabsProps {
  value: string
  onChange: (value: string) => void
  children: ReactNode
  className?: string
}

function Tabs({ value, onChange, children, className }: TabsProps) {
  return (
    <TabsContext.Provider value={{ value, onChange }}>
      <div className={cn('flex flex-col', className)}>{children}</div>
    </TabsContext.Provider>
  )
}

function TabsList({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn('flex border-b border-gray-200', className)}>
      {children}
    </div>
  )
}

interface TabsTriggerProps {
  value: string
  children: ReactNode
  className?: string
}

function TabsTrigger({ value, children, className }: TabsTriggerProps) {
  const { value: activeValue, onChange } = useTabs()
  const isActive = activeValue === value
  return (
    <button
      type="button"
      onClick={() => onChange(value)}
      className={cn(
        'px-4 py-2 text-sm font-medium transition-colors',
        'border-b-2 -mb-px',
        isActive
          ? 'border-blue-600 text-blue-600'
          : 'border-transparent text-gray-500 hover:text-gray-700',
        className,
      )}
    >
      {children}
    </button>
  )
}

interface TabsContentProps {
  value: string
  children: ReactNode
  className?: string
}

function TabsContent({ value, children, className }: TabsContentProps) {
  const { value: activeValue } = useTabs()
  if (activeValue !== value) return null
  return <div className={cn('py-4', className)}>{children}</div>
}

export { Tabs, TabsList, TabsTrigger, TabsContent }
