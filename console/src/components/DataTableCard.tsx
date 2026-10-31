import React, { useState, useMemo, useEffect } from 'react'
import {
  Search,
  X,
  ChevronLeft,
  ChevronRight,
  FileSpreadsheet,
  Download,
  ChevronDown
} from 'lucide-react'
import { exportToExcel, exportToCSV, exportToJSON, ExportColumn } from '../utils/export'

export interface DataTableCardProps<T extends Record<string, any>> {
  title?: string
  subtitle?: string
  icon?: React.ComponentType<{ className?: string }>
  data: T[]
  exportFilename?: string
  exportSheetTitle?: string
  exportColumns?: ExportColumn[]
  exportFormatter?: (item: T) => Record<string, any>
  searchPlaceholder?: string
  searchFields?: (keyof T | string)[]
  defaultPageSize?: number
  pageSizeOptions?: number[]
  extraHeaderActions?: React.ReactNode
  emptyMessage?: string
  className?: string
  hideHeader?: boolean
  children: (paginatedItems: T[], allFilteredItems: T[]) => React.ReactNode
}

/**
 * Universal Hook for Searching, Paginating, and Exporting Data Tables
 */
export function useDataTable<T extends Record<string, any>>({
  data,
  searchFields,
  defaultPageSize = 10,
}: {
  data: T[]
  searchFields?: (keyof T | string)[]
  defaultPageSize?: number
}) {
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(defaultPageSize)

  // Reset page when search or pageSize changes
  useEffect(() => {
    setPage(1)
  }, [search, pageSize])

  const filteredData = useMemo(() => {
    if (!search.trim()) return data
    const query = search.toLowerCase().trim()
    return data.filter((item) => {
      if (searchFields && searchFields.length > 0) {
        return searchFields.some((field) => {
          const val = item[field as string]
          if (val === null || val === undefined) return false
          return String(val).toLowerCase().includes(query)
        })
      }
      return Object.values(item).some((val) => {
        if (val === null || val === undefined) return false
        if (typeof val === 'object') {
          return JSON.stringify(val).toLowerCase().includes(query)
        }
        return String(val).toLowerCase().includes(query)
      })
    })
  }, [data, search, searchFields])

  const totalPages = Math.max(1, Math.ceil(filteredData.length / pageSize))
  const safePage = Math.min(page, totalPages)
  const startIndex = (safePage - 1) * pageSize
  const endIndex = Math.min(startIndex + pageSize, filteredData.length)

  const paginatedData = useMemo(() => {
    return filteredData.slice(startIndex, endIndex)
  }, [filteredData, startIndex, endIndex])

  return {
    search,
    setSearch,
    page: safePage,
    setPage,
    pageSize,
    setPageSize,
    filteredData,
    paginatedData,
    totalItems: data.length,
    filteredCount: filteredData.length,
    totalPages,
    startIndex,
    endIndex,
  }
}

/**
 * Reusable Pagination Footer Component
 */
export function TablePaginationFooter({
  page,
  totalPages,
  onPageChange,
  startIndex,
  endIndex,
  totalCount,
  filteredCount,
}: {
  page: number
  totalPages: number
  onPageChange: (p: number) => void
  startIndex: number
  endIndex: number
  totalCount: number
  filteredCount: number
}) {
  const pageNumbers = useMemo(() => {
    const pages: (number | string)[] = []
    if (totalPages <= 7) {
      for (let i = 1; i <= totalPages; i++) pages.push(i)
    } else {
      if (page <= 4) {
        pages.push(1, 2, 3, 4, 5, '...', totalPages)
      } else if (page >= totalPages - 3) {
        pages.push(1, '...', totalPages - 4, totalPages - 3, totalPages - 2, totalPages - 1, totalPages)
      } else {
        pages.push(1, '...', page - 1, page, page + 1, '...', totalPages)
      }
    }
    return pages
  }, [page, totalPages])

  if (filteredCount === 0) return null

  return (
    <div className="p-3 sm:px-6 border-t border-slate-100 dark:border-slate-800/80 bg-slate-50/40 dark:bg-slate-950/30 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs text-slate-500 dark:text-slate-400">
      <div>
        Showing <span className="font-semibold text-slate-800 dark:text-slate-200 font-mono">{startIndex + 1}</span> to{' '}
        <span className="font-semibold text-slate-800 dark:text-slate-200 font-mono">{endIndex}</span> of{' '}
        <span className="font-semibold text-slate-800 dark:text-slate-200 font-mono">{filteredCount}</span> entries
        {filteredCount < totalCount && (
          <span className="ml-1 text-[11px] text-slate-400 italic">
            (filtered from {totalCount} total)
          </span>
        )}
      </div>

      {totalPages > 1 && (
        <div className="flex items-center space-x-1 self-center sm:self-auto">
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => onPageChange(Math.max(1, page - 1))}
            className="p-1.5 rounded-lg border border-slate-200 dark:border-slate-700/80 bg-white/80 dark:bg-slate-900/80 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-200 disabled:opacity-30 disabled:pointer-events-none transition"
            title="Previous page"
          >
            <ChevronLeft className="w-3.5 h-3.5" />
          </button>

          <div className="flex items-center space-x-1">
            {pageNumbers.map((p, idx) => {
              if (typeof p === 'string') {
                return (
                  <span key={`ellipsis-${idx}`} className="px-2 text-slate-400">
                    ...
                  </span>
                )
              }
              const isCurrent = p === page
              return (
                <button
                  key={p}
                  type="button"
                  onClick={() => onPageChange(p)}
                  className={`min-w-[28px] h-7 px-2 rounded-lg text-xs font-semibold font-mono transition-all duration-150 ${
                    isCurrent
                      ? 'bg-sky-500 text-white shadow-xs'
                      : 'border border-slate-200 dark:border-slate-700/80 bg-white/80 dark:bg-slate-900/80 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-300'
                  }`}
                >
                  {p}
                </button>
              )
            })}
          </div>

          <button
            type="button"
            disabled={page >= totalPages}
            onClick={() => onPageChange(Math.min(totalPages, page + 1))}
            className="p-1.5 rounded-lg border border-slate-200 dark:border-slate-700/80 bg-white/80 dark:bg-slate-900/80 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-200 disabled:opacity-30 disabled:pointer-events-none transition"
            title="Next page"
          >
            <ChevronRight className="w-3.5 h-3.5" />
          </button>
        </div>
      )}
    </div>
  )
}

/**
 * Universal Frosted-Glass Card with Search, Pagination & Export as Excel
 */
export function DataTableCard<T extends Record<string, any>>({
  title,
  subtitle,
  icon: Icon,
  data,
  exportFilename = 'export',
  exportSheetTitle = 'Sheet1',
  exportColumns,
  exportFormatter,
  searchPlaceholder = 'Search...',
  searchFields,
  defaultPageSize = 10,
  pageSizeOptions = [10, 25, 50, 100],
  extraHeaderActions,
  emptyMessage = 'No records found.',
  className = '',
  hideHeader = false,
  children,
}: DataTableCardProps<T>) {
  const {
    search,
    setSearch,
    page,
    setPage,
    pageSize,
    setPageSize,
    filteredData,
    paginatedData,
    totalItems,
    filteredCount,
    totalPages,
    startIndex,
    endIndex,
  } = useDataTable({ data, searchFields, defaultPageSize })

  const [exportMenuOpen, setExportMenuOpen] = useState(false)

  // Export data preparation
  const getExportRows = () => {
    const sourceData = filteredData.length > 0 ? filteredData : data
    if (exportFormatter) {
      return sourceData.map(exportFormatter)
    }
    return sourceData
  }

  // Infer export columns if not explicitly provided
  const resolvedExportColumns: ExportColumn[] = useMemo(() => {
    if (exportColumns && exportColumns.length > 0) return exportColumns
    if (data.length === 0) return []
    const sample = exportFormatter ? exportFormatter(data[0]) : data[0]
    return Object.keys(sample).map((k) => ({
      header: k.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
      key: k,
    }))
  }, [exportColumns, data, exportFormatter])

  const handleExportExcel = () => {
    exportToExcel(exportFilename, exportSheetTitle, resolvedExportColumns, getExportRows())
    setExportMenuOpen(false)
  }

  const handleExportCSV = () => {
    exportToCSV(exportFilename, resolvedExportColumns, getExportRows())
    setExportMenuOpen(false)
  }

  const handleExportJSON = () => {
    exportToJSON(exportFilename, getExportRows())
    setExportMenuOpen(false)
  }

  return (
    <div
      className={`bg-white/75 dark:bg-slate-900/75 backdrop-blur-xl border border-white/60 dark:border-slate-800/80 rounded-2xl shadow-glass dark:shadow-glass-dark overflow-hidden flex flex-col ${className}`}
    >
      {/* Optional Card Title Header */}
      {!hideHeader && title && (
        <div className="p-4 sm:px-6 border-b border-slate-100 dark:border-slate-800/80 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div className="flex items-center space-x-2.5">
            {Icon && (
              <div className="p-2 rounded-xl bg-sky-500/10 text-sky-600 dark:text-sky-400">
                <Icon className="w-4 h-4" />
              </div>
            )}
            <div>
              <div className="flex items-center space-x-2">
                <h3 className="font-bold text-sm sm:text-base text-slate-900 dark:text-white">
                  {title}
                </h3>
                <span className="text-[11px] font-semibold px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 border border-slate-200/80 dark:border-slate-700/60 font-mono">
                  {search ? `${filteredCount} of ${totalItems}` : `${totalItems} Total`}
                </span>
              </div>
              {subtitle && (
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">{subtitle}</p>
              )}
            </div>
          </div>
          {extraHeaderActions && (
            <div className="flex items-center space-x-2 self-start sm:self-auto">
              {extraHeaderActions}
            </div>
          )}
        </div>
      )}

      {/* Search, Filter & Export Action Bar */}
      <div className="p-3 sm:px-6 bg-slate-50/50 dark:bg-slate-950/40 border-b border-slate-100 dark:border-slate-800/80 flex flex-col md:flex-row md:items-center justify-between gap-3">
        {/* Search Input Bar */}
        <div className="relative flex-1 max-w-md">
          <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={searchPlaceholder}
            className="w-full pl-9 pr-8 py-2 bg-white/80 dark:bg-slate-900/80 border border-slate-200 dark:border-slate-700/80 rounded-xl text-xs text-slate-800 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-sky-500/30 focus:border-sky-500 backdrop-blur-md transition-all shadow-xs"
          />
          {search && (
            <button
              type="button"
              onClick={() => setSearch('')}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 p-0.5 rounded-full transition"
              title="Clear search"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
        </div>

        {/* Controls: Page Size & Export as Excel Dropdown */}
        <div className="flex items-center space-x-2.5 self-end md:self-auto">
          {/* Page Size Selector */}
          <div className="flex items-center space-x-1.5 text-xs text-slate-500 dark:text-slate-400">
            <span className="hidden sm:inline">Per page:</span>
            <select
              value={pageSize}
              onChange={(e) => setPageSize(Number(e.target.value))}
              className="bg-white/80 dark:bg-slate-900/80 border border-slate-200 dark:border-slate-700/80 rounded-xl px-2 py-1.5 text-xs text-slate-700 dark:text-slate-300 font-semibold focus:outline-none focus:ring-2 focus:ring-sky-500/30 cursor-pointer shadow-xs"
            >
              {pageSizeOptions.map((opt) => (
                <option key={opt} value={opt} className="bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200">
                  {opt}
                </option>
              ))}
            </select>
          </div>

          {/* Export to Excel Dropdown Menu */}
          <div className="relative inline-block text-left">
            <button
              type="button"
              disabled={data.length === 0}
              onClick={() => setExportMenuOpen(!exportMenuOpen)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white/90 dark:bg-slate-800/80 hover:bg-white dark:hover:bg-slate-700/80 text-slate-800 dark:text-slate-100 text-xs font-semibold border border-slate-200/80 dark:border-slate-700/70 shadow-xs transition-all duration-150 hover:scale-[1.01] active:scale-[0.99] disabled:opacity-40"
              title="Export as Excel or CSV"
            >
              <FileSpreadsheet className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
              <span>Export</span>
              <ChevronDown className={`w-3 h-3 text-slate-400 transition-transform ${exportMenuOpen ? 'rotate-180' : ''}`} />
            </button>

            {exportMenuOpen && (
              <>
                <div
                  className="fixed inset-0 z-40"
                  onClick={() => setExportMenuOpen(false)}
                />
                <div className="absolute right-0 mt-1.5 w-52 rounded-xl bg-white/95 dark:bg-slate-900/95 backdrop-blur-xl border border-slate-200/80 dark:border-slate-800/90 shadow-xl z-50 py-1.5 focus:outline-none animate-in fade-in zoom-in-95 duration-100">
                  <div className="px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-slate-400 dark:text-slate-500 border-b border-slate-100 dark:border-slate-800/80 flex items-center justify-between">
                    <span>Export Data</span>
                    <span className="font-mono text-[9px] text-sky-500 font-bold">{getExportRows().length} rows</span>
                  </div>

                  <button
                    type="button"
                    onClick={handleExportExcel}
                    className="w-full text-left flex items-center gap-2.5 px-3 py-2 text-xs text-slate-700 dark:text-slate-200 hover:bg-emerald-500/10 hover:text-emerald-700 dark:hover:text-emerald-300 transition font-medium"
                  >
                    <FileSpreadsheet className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0" />
                    <div>
                      <div className="font-semibold flex items-center gap-1.5">
                        <span>Microsoft Excel</span>
                        <span className="text-[9px] bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 px-1 rounded font-mono font-bold">.xlsx</span>
                      </div>
                      <div className="text-[10px] text-slate-400">Styled spreadsheet with headers</div>
                    </div>
                  </button>

                  <button
                    type="button"
                    onClick={handleExportCSV}
                    className="w-full text-left flex items-center gap-2.5 px-3 py-2 text-xs text-slate-700 dark:text-slate-200 hover:bg-sky-500/10 hover:text-sky-700 dark:hover:text-sky-300 transition font-medium"
                  >
                    <Download className="w-4 h-4 text-sky-600 dark:text-sky-400 flex-shrink-0" />
                    <div>
                      <div className="font-semibold flex items-center gap-1.5">
                        <span>CSV Spreadsheet</span>
                        <span className="text-[9px] bg-sky-500/10 text-sky-600 dark:text-sky-400 border border-sky-500/20 px-1 rounded font-mono font-bold">.csv</span>
                      </div>
                      <div className="text-[10px] text-slate-400">UTF-8 BOM comma-separated</div>
                    </div>
                  </button>

                  <button
                    type="button"
                    onClick={handleExportJSON}
                    className="w-full text-left flex items-center gap-2.5 px-3 py-2 text-xs text-slate-700 dark:text-slate-200 hover:bg-purple-500/10 hover:text-purple-700 dark:hover:text-purple-300 transition font-medium"
                  >
                    <div className="w-4 h-4 rounded bg-purple-500/10 text-purple-600 dark:text-purple-400 flex items-center justify-center font-mono text-[10px] font-bold flex-shrink-0">
                      {'{}'}
                    </div>
                    <div>
                      <div className="font-semibold flex items-center gap-1.5">
                        <span>Raw JSON</span>
                        <span className="text-[9px] bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 px-1 rounded font-mono font-bold">.json</span>
                      </div>
                      <div className="text-[10px] text-slate-400">Full structured object tree</div>
                    </div>
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Table Content or Empty State */}
      {data.length === 0 ? (
        <div className="p-8 sm:p-12 text-center text-slate-400 dark:text-slate-500 text-xs">
          {emptyMessage}
        </div>
      ) : filteredData.length === 0 ? (
        <div className="p-8 sm:p-12 text-center space-y-3">
          <p className="text-xs text-slate-500 dark:text-slate-400">
            No records found matching <span className="font-semibold text-slate-800 dark:text-slate-200 font-mono">"{search}"</span>
          </p>
          <button
            type="button"
            onClick={() => setSearch('')}
            className="btn-secondary text-xs px-3 py-1.5"
          >
            Clear Filter
          </button>
        </div>
      ) : (
        <div className="overflow-x-auto w-full">
          {children(paginatedData, filteredData)}
        </div>
      )}

      {/* Pagination Footer */}
      <TablePaginationFooter
        page={page}
        totalPages={totalPages}
        onPageChange={setPage}
        startIndex={startIndex}
        endIndex={endIndex}
        totalCount={totalItems}
        filteredCount={filteredCount}
      />
    </div>
  )
}
