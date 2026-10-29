/**
 * Universal Multi-Format Export Engine for Mailvex Console
 * Supports:
 *  1. Microsoft Excel SpreadsheetML (.xlsx / .xls)
 *  2. RFC 4180 CSV with UTF-8 BOM (.csv)
 *  3. Formatted JSON (.json)
 */

export interface ExportColumn {
  header: string
  key: string
  type?: 'string' | 'number' | 'date'
}

/**
 * Escapes values for RFC 4180 CSV
 */
function escapeCSVValue(val: any): string {
  if (val === null || val === undefined) return ''
  const str = String(val)
  if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
    return `"${str.replace(/"/g, '""')}"`
  }
  return str
}

/**
 * Trigger browser file download from Blob
 */
function downloadBlob(blob: Blob, filename: string) {
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.style.display = 'none'
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  setTimeout(() => {
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)
  }, 200)
}

/**
 * Export data as RFC 4180 CSV with UTF-8 BOM
 */
export function exportToCSV(filename: string, columns: ExportColumn[], rows: Record<string, any>[]) {
  const headerLine = columns.map(c => escapeCSVValue(c.header)).join(',')
  const dataLines = rows.map(row => {
    return columns.map(col => escapeCSVValue(row[col.key])).join(',')
  })

  // \uFEFF is UTF-8 Byte Order Mark ensuring Microsoft Excel parses unicode correctly
  const csvContent = '\uFEFF' + [headerLine, ...dataLines].join('\r\n')
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const finalName = filename.endsWith('.csv') ? filename : `${filename}.csv`
  downloadBlob(blob, finalName)
}

/**
 * Export data as Microsoft Excel SpreadsheetML XML (.xls / .xlsx compatible)
 * Native Excel format supporting multiple data types and styled headers.
 */
export function exportToExcel(filename: string, sheetTitle: string, columns: ExportColumn[], rows: Record<string, any>[]) {
  const escapeXML = (str: any) => {
    if (str === null || str === undefined) return ''
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&apos;')
  }

  let xml = `<?xml version="1.0"?>
<?mso-application progid="Excel.Sheet"?>
<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:o="urn:schemas-microsoft-com:office:office"
 xmlns:x="urn:schemas-microsoft-com:office:excel"
 xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:html="http://www.w3.org/TR/REC-html40">
 <Styles>
  <Style ss:ID="Default" ss:Name="Normal">
   <Alignment ss:Vertical="Center"/>
   <Font ss:FontName="Segoe UI" ss:Size="10" ss:Color="#1E293B"/>
  </Style>
  <Style ss:ID="HeaderStyle">
   <Alignment ss:Horizontal="Center" ss:Vertical="Center"/>
   <Borders>
    <Border ss:Position="Bottom" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#CBD5E1"/>
   </Borders>
   <Font ss:FontName="Segoe UI" ss:Size="11" ss:Bold="1" ss:Color="#0F172A"/>
   <Interior ss:Color="#F1F5F9" ss:Pattern="Solid"/>
  </Style>
  <Style ss:ID="NumericStyle">
   <Alignment ss:Horizontal="Right" ss:Vertical="Center"/>
   <Font ss:FontName="Segoe UI" ss:Size="10" ss:Color="#1E293B"/>
  </Style>
 </Styles>
 <Worksheet ss:Name="${escapeXML(sheetTitle || 'Sheet1')}">
  <Table>
`
  // Column definitions
  for (let i = 0; i < columns.length; i++) {
    xml += `   <Column ss:AutoFitWidth="1" ss:Width="130"/>\n`
  }

  // Header Row
  xml += `   <Row ss:Height="26">\n`
  for (const col of columns) {
    xml += `    <Cell ss:StyleID="HeaderStyle"><Data ss:Type="String">${escapeXML(col.header)}</Data></Cell>\n`
  }
  xml += `   </Row>\n`

  // Data Rows
  for (const row of rows) {
    xml += `   <Row ss:Height="20">\n`
    for (const col of columns) {
      const val = row[col.key]
      const isNum = col.type === 'number' || (typeof val === 'number' && !isNaN(val))
      if (isNum && val !== null && val !== undefined && val !== '') {
        xml += `    <Cell ss:StyleID="NumericStyle"><Data ss:Type="Number">${val}</Data></Cell>\n`
      } else {
        xml += `    <Cell><Data ss:Type="String">${escapeXML(val)}</Data></Cell>\n`
      }
    }
    xml += `   </Row>\n`
  }

  xml += `  </Table>
 </Worksheet>
</Workbook>`

  const blob = new Blob([xml], { type: 'application/vnd.ms-excel;charset=utf-8;' })
  const finalName = filename.endsWith('.xls') || filename.endsWith('.xlsx') ? filename : `${filename}.xlsx`
  downloadBlob(blob, finalName)
}

/**
 * Export data as formatted JSON
 */
export function exportToJSON(filename: string, data: any) {
  const jsonContent = JSON.stringify(data, null, 2)
  const blob = new Blob([jsonContent], { type: 'application/json;charset=utf-8;' })
  const finalName = filename.endsWith('.json') ? filename : `${filename}.json`
  downloadBlob(blob, finalName)
}
