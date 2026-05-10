import { readFile, readdir } from 'node:fs/promises'
import { join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { dirname } from 'node:path'

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const srcRoot = resolve(webRoot, 'src')

const disallowedApplyPatterns = [
  /\@apply[^\n;]*(?:bg|border|text)-(?:gray|slate|zinc|neutral|stone|black|white|blue|sky)-/g,
  /\@apply[^\n;]*text-white/g,
]

const allowedFiles = new Set([
  'src/App.vue',
])

const findings = []
for (const file of await listVueFiles(srcRoot)) {
  const rel = relative(webRoot, file)
  if (allowedFiles.has(rel)) continue

  const content = await readFile(file, 'utf8')
  const scopedBlocks = [...content.matchAll(/<style\b[^>]*\bscoped\b[^>]*>([\s\S]*?)<\/style>/g)]
  for (const block of scopedBlocks) {
    const css = block[1]
    for (const pattern of disallowedApplyPatterns) {
      for (const match of css.matchAll(pattern)) {
        const line = content.slice(0, block.index + match.index).split('\n').length
        findings.push(`${rel}:${line}: scoped style should use theme variables instead of "${match[0].trim()}"`)
      }
    }
  }
}

if (findings.length > 0) {
  console.error('Theme audit failed:')
  for (const finding of findings) {
    console.error(`- ${finding}`)
  }
  process.exit(1)
}

console.log('theme audit passed')

async function listVueFiles(dir) {
  const entries = await readdir(dir, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    const path = join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...await listVueFiles(path))
    } else if (entry.isFile() && entry.name.endsWith('.vue')) {
      files.push(path)
    }
  }
  return files
}
