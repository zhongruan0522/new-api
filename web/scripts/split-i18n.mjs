// One-time script: physically splits src/i18n/locales/{en,zh}.json big files into
// per-feature section files under src/i18n/locales/{en,zh}/<section>.json.
//
// Run from the web/ package root:  node scripts/split-i18n.mjs
//
// Classification: each key is located by scanning web/src for t('key') / t("key")
// / i18n.t('key') usages and bucketed by the feature directory of the first hit.
// Unmatched / ambiguous keys go to common.json. Layout keys go to layout.json.
import fs from 'node:fs/promises'
import path from 'node:path'

const ROOT = path.resolve('src')
const LOCALES_DIR = path.resolve('src/i18n/locales')
const SRC_DIR = path.resolve('src')

// Ordered feature -> section mapping (first match wins when mapping a file path).
// Use simple substring includes against the repo-relative posix path.
const FEATURE_SECTIONS = [
  'auth',
  'dashboard',
  'channels',
  'usage-logs',
  'system-settings',
  'audit-logs',
  'users',
  'keys',
  'wallet',
  'models',
  'multimodal',
  'minimax',
  'rankings',
  'announcements',
  'multimodal-files',
  'pricing',
  'redemption-codes',
  'tickets',
  'subscriptions',
  'playground',
  'performance-metrics',
  'dynamic-ratio',
  'order-query',
  'key-query',
  'profile',
  'home',
  'about',
  'setup',
]

// Path-prefix hints that should go to layout.json (shell: header/footer/sidebar/route shell).
const LAYOUT_HINTS = [
  '/src/components/layout/',
  '/src/components/sidebar',
  '/src/components/header',
  '/src/components/footer',
  '/src/components/topbar',
  '/src/components/nav',
  '/src/routes/_authenticated/route',
  '/src/routes/__root',
  '/src/routes/_authenticated/layout',
]

// Files that aggregate constants/state rather than UI; route there to common.
const COMMON_HINTS = ['/src/stores/', '/src/lib/', '/src/hooks/', '/src/i18n/', '/src/constants']

function toPosix(p) {
  return p.split(path.sep).join('/')
}

function detectSection(filePath) {
  const p = toPosix(filePath)
  if (LAYOUT_HINTS.some((h) => p.includes(h))) return 'layout'
  for (const f of FEATURE_SECTIONS) {
    if (p.includes(`/features/${f}/`) || p.includes(`/routes/_authenticated/${f}/`)) return f
  }
  if (p.includes('/features/errors/')) return 'common'
  if (p.includes('/features/legal/')) return 'common'
  if (COMMON_HINTS.some((h) => p.includes(h))) return 'common'
  return 'common'
}

async function loadLocale(locale) {
  const raw = await fs.readFile(path.join(LOCALES_DIR, `${locale}.json`), 'utf8')
  const parsed = JSON.parse(raw)
  const trans = parsed?.translation
  if (!trans || typeof trans !== 'object' || Array.isArray(trans)) {
    throw new Error(`${locale}.json has no translation object`)
  }
  return trans
}

async function listSrcFiles() {
  const out = []
  async function walk(dir) {
    const entries = await fs.readdir(dir, { withFileTypes: true })
    for (const e of entries) {
      const full = path.join(dir, e.name)
      if (e.isDirectory()) {
        if (e.name === 'node_modules' || e.name === 'i18n') continue
        await walk(full)
      } else if (e.isFile() && (e.name.endsWith('.ts') || e.name.endsWith('.tsx'))) {
        out.push(full)
      }
    }
  }
  await walk(SRC_DIR)
  return out
}

// Build a map key -> section, by scanning source for t('key') usages.
async function classifyKeys(keys) {
  const files = await listSrcFiles()
  const keySet = new Set(keys)
  const keyToSection = new Map()

  // Regex captures the key inside t('...') / t("...") / i18n.t('...') / t(`...`).
  // Keep it conservative: only single-line string literals.
  const re = /\bt\(\s*['"`]([^'"`\n]+?)['"`]/g

  for (const file of files) {
    const text = await fs.readFile(file, 'utf8')
    if (!text.includes('t(')) continue
    let m
    while ((m = re.exec(text)) !== null) {
      const key = m[1]
      if (!keySet.has(key)) continue
      // Skip dynamic interpolations like {{x}} only-keys won't match; literals are fine.
      if (keyToSection.has(key)) continue // first-writer-wins
      keyToSection.set(key, detectSection(file))
    }
  }

  // Anything unmatched -> common.
  for (const k of keys) if (!keyToSection.has(k)) keyToSection.set(k, 'common')
  return keyToSection
}

function buildSections(trans, keyToSection) {
  const sections = {}
  for (const k of Object.keys(trans)) {
    const section = keyToSection.get(k) || 'common'
    if (!sections[section]) sections[section] = {}
    sections[section][k] = trans[k]
  }
  return sections
}

function verify(locale, original, sections) {
  // Re-flatten sections and compare keys/values for exact preservation.
  const merged = {}
  let dup = []
  for (const section of Object.keys(sections)) {
    const s = sections[section]
    for (const k of Object.keys(s)) {
      if (Object.prototype.hasOwnProperty.call(merged, k)) {
        dup.push(k)
        continue
      }
      merged[k] = s[k]
    }
  }
  const origKeys = Object.keys(original)
  const mergedKeys = Object.keys(merged)
  const missing = origKeys.filter((k) => !Object.prototype.hasOwnProperty.call(merged, k))
  const extra = mergedKeys.filter((k) => !Object.prototype.hasOwnProperty.call(original, k))
  const mismatched = origKeys.filter((k) => merged[k] !== original[k])
  return { dup, missing, extra, mismatched, origCount: origKeys.length, mergedCount: mergedKeys.length }
}

async function writeJson(file, obj) {
  const text = JSON.stringify(obj, null, 2) + '\n'
  await fs.writeFile(file, text, 'utf8')
}

async function main() {
  const locales = ['en', 'zh']
  // First, classify keys using the en file (canonical source of t('...') keys).
  const enTrans = await loadLocale('en')
  const keys = Object.keys(enTrans)
  console.log(`Total translation keys: ${keys.length}`)

  const keyToSection = await classifyKeys(keys)
  const sectionCounts = {}
  for (const s of keyToSection.values()) sectionCounts[s] = (sectionCounts[s] || 0) + 1
  console.log('Section distribution:')
  for (const [s, c] of Object.entries(sectionCounts).sort((a, b) => b[1] - a[1])) {
    console.log(`  ${s.padEnd(20)} ${c}`)
  }

  // Ensure the same set of section files exists for both locales (so config.ts imports stay aligned).
  const allSections = Array.from(new Set([...keyToSection.values(), 'common', 'layout'])).sort()

  for (const locale of locales) {
    const trans = locale === 'en' ? enTrans : await loadLocale(locale)
    const sections = buildSections(trans, keyToSection)
    // Make sure every locale has every section file even if empty (keeps imports stable).
    for (const s of allSections) if (!sections[s]) sections[s] = {}

    const v = verify(locale, trans, sections)
    if (v.dup.length || v.missing.length || v.extra.length || v.mismatched.length) {
      throw new Error(
        `Verification failed for ${locale}: dup=${v.dup.length} missing=${v.missing.length} extra=${v.extra.length} mismatched=${v.mismatched.length}`,
      )
    }
    console.log(`Verified ${locale}: ${v.mergedCount}/${v.origCount} keys preserved`)

    const outDir = path.join(LOCALES_DIR, locale)
    await fs.mkdir(outDir, { recursive: true })
    for (const s of Object.keys(sections)) {
      await writeJson(path.join(outDir, `${s}.json`), sections[s])
    }
    console.log(`Wrote ${Object.keys(sections).length} section files to ${toPosix(outDir)}`)
  }

  console.log('Split complete. Next: update config.ts and sync-i18n.mjs.')
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
