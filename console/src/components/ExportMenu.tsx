import React, { useState, useRef, useEffect } from 'react'
import { Download, ChevronDown, FileSpreadsheet, FileText, FileCode } from 'lucide-react'
import { exportToExcel, exportToCSV, exportToJSON, ExportColumn } from '../utils/export'

interface ExportMenuProps {
  filename: string
  sheetTitle?: string
  columns: ExportColumn[]
  data: Record<string, any>[]
  label?: string
  size?: 'sm' | 'md'
  disabled?: boolean
}

export const ExportMenu: React.FC<ExportMenuProps> = ({
  filename,
  sheetTitle = 'Report',
  columns,
  data,
  label = 'Export',
  size = 'md',
  disabled = false,
}) => {
  const [isOpen, setIsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
    }
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isOpen])

  const handleExcel = () => {
    exportToExcel(filename, sheetTitle, columns, data)
    setIsOpen(false)
  }

  const handleCSV = () => {
    exportToCSV(filename, columns, data)
    setIsOpen(false)
  }

  const handleJSON = () => {
    exportToJSON(filename, data)
    setIsOpen(false)
  }

  const sizeClasses = size === 'sm'
    ? 'px-2.5 py-1.5 text-xs'
    : 'px-3.5 py-2 text-xs'

  return (
    <div className="relative inline-block text-left" ref={menuRef}>
      <button
        type="button"
        disabled={disabled}
        onClick={() => setIsOpen(!isOpen)}
        className={`btn-secondary ${sizeClasses} inline-flex items-center gap-1.5 font-semibold transition-all duration-150 disabled:opacity-50`}
        title="Export options (Excel, CSV, JSON)"
      >
        <Download className="w-3.5 h-3.5 text-sky-500" />
        <span>{label}</span>
        <ChevronDown className={`w-3 h-3 text-slate-400 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-1.5 w-48 rounded-xl bg-white/95 dark:bg-slate-900/95 backdrop-blur-xl border border-slate-200/80 dark:border-slate-800/90 shadow-xl z-50 py-1.5 focus:outline-none animate-in fade-in zoom-in-95 duration-100">
          <div className="px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-slate-400 dark:text-slate-500 border-b border-slate-100 dark:border-slate-800/80">
            Export Format
          </div>

          <button
            type="button"
            onClick={handleExcel}
            className="w-full text-left flex items-center gap-2.5 px-3 py-2 text-xs text-slate-700 dark:text-slate-200 hover:bg-slate-100/80 dark:hover:bg-slate-800/70 transition font-medium"
          >
            <FileSpreadsheet className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
            <div>
              <div className="font-semibold">Microsoft Excel</div>
              <div className="text-[10px] text-slate-400">.xlsx / .xls formatted</div>
            </div>
          </button>

          <button
            type="button"
            onClick={handleCSV}
            className="w-full text-left flex items-center gap-2.5 px-3 py-2 text-xs text-slate-700 dark:text-slate-200 hover:bg-slate-100/80 dark:hover:bg-slate-800/70 transition font-medium"
          >
            <FileText className="w-4 h-4 text-sky-600 dark:text-sky-400" />
            <div>
              <div className="font-semibold">CSV Spreadsheet</div>
              <div className="text-[10px] text-slate-400">.csv with UTF-8 BOM</div>
            </div>
          </button>

          <button
            type="button"
            onClick={handleJSON}
            className="w-full text-left flex items-center gap-2.5 px-3 py-2 text-xs text-slate-700 dark:text-slate-200 hover:bg-slate-100/80 dark:hover:bg-slate-800/70 transition font-medium"
          >
            <FileCode className="w-4 h-4 text-purple-600 dark:text-purple-400" />
            <div>
              <div className="font-semibold">Standard JSON</div>
              <div className="text-[10px] text-slate-400">.json data objects</div>
            </div>
          </button>
        </div>
      )}
    </div>
  )
}
