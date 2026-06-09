import { mkdir, readFile, stat, unlink, writeFile } from 'node:fs/promises'
import { execFile as execFileCallback } from 'node:child_process'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const execFile = promisify(execFileCallback)

const requiredDirectories = [
  'src/api',
  'src/assets/brand',
  'src/composables',
  'src/config',
  'src/lib',
  'src/router',
  'src/views',
  'public',
]

const syncedFiles = [
  {
    from: 'src/assets/brand/easyllm-app-icon.png',
    to: 'public/app-icon.png',
    reason: 'public/app-icon.png is the browser-facing copy of the canonical brand asset.',
  },
]

const generatedIconFiles = [
  { size: 16, to: 'public/favicon-16x16.png' },
  { size: 32, to: 'public/favicon-32x32.png' },
  { size: 180, to: 'public/apple-touch-icon.png' },
]

const deprecatedFiles = [
  'public/logo.svg',
  'src/assets/brand/easyllm-logo.svg',
]

const requiredFiles = [
  'public/app-icon.png',
  'public/favicon-16x16.png',
  'public/favicon-32x32.png',
  'public/apple-touch-icon.png',
  'src/assets/brand/easyllm-app-icon.png',
  'src/config/theme.js',
  'src/composables/useAppearance.js',
]

for (const dir of requiredDirectories) {
  await mkdir(resolve(webRoot, dir), { recursive: true })
}

for (const item of syncedFiles) {
  const source = resolve(webRoot, item.from)
  const target = resolve(webRoot, item.to)
  const sourceData = await readFile(source)
  const targetData = await readOptionalFile(target)
  if (!targetData || !sourceData.equals(targetData)) {
    await mkdir(dirname(target), { recursive: true })
    await writeFile(target, sourceData)
    console.log(`normalized ${item.to}`)
  }
}

const canGenerateIcons = await hasCommand('sips')
for (const icon of generatedIconFiles) {
  const source = resolve(webRoot, 'src/assets/brand/easyllm-app-icon.png')
  const target = resolve(webRoot, icon.to)
  if (canGenerateIcons) {
    await execFile('sips', ['-z', String(icon.size), String(icon.size), source, '--out', target])
  } else if (!(await exists(target))) {
    console.error(`Missing generated icon ${relativeToWebRoot(target)} and 'sips' is not available to create it.`)
    process.exitCode = 1
  }
}

for (const file of deprecatedFiles) {
  await removeIfExists(resolve(webRoot, file))
}

const missingFiles = []
for (const file of requiredFiles) {
  if (!(await exists(resolve(webRoot, file)))) {
    missingFiles.push(file)
  }
}

if (missingFiles.length > 0) {
  console.error('Missing required project files:')
  for (const file of missingFiles) {
    console.error(`- ${file}`)
  }
  process.exitCode = 1
} else {
  console.log('project structure normalized')
}

async function readOptionalFile(path) {
  try {
    return await readFile(path)
  } catch (error) {
    if (error?.code === 'ENOENT') return null
    throw error
  }
}

async function exists(path) {
  try {
    await stat(path)
    return true
  } catch (error) {
    if (error?.code === 'ENOENT') return false
    throw error
  }
}

async function hasCommand(command) {
  const lookup = process.platform === 'win32' ? 'where' : 'which'
  try {
    await execFile(lookup, [command])
    return true
  } catch {
    return false
  }
}

async function removeIfExists(path) {
  try {
    await unlink(path)
    console.log(`removed deprecated ${relativeToWebRoot(path)}`)
  } catch (error) {
    if (error?.code === 'ENOENT') return
    throw error
  }
}

function relativeToWebRoot(path) {
  return path.startsWith(webRoot) ? path.slice(webRoot.length + 1) : path
}
