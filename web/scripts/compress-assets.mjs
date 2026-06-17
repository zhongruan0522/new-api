/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { createReadStream, createWriteStream } from 'node:fs'
import fs from 'node:fs/promises'
import path from 'node:path'
import { pipeline } from 'node:stream/promises'
import zlib from 'node:zlib'

const DIST_DIR = path.resolve('dist/static')
const MIN_SIZE_BYTES = 1024
const COMPRESSIBLE_EXTENSIONS = new Set([
  '.css',
  '.js',
  '.json',
  '.mjs',
  '.svg',
  '.txt',
  '.xml',
])

function shouldCompress(filePath, size) {
  if (size < MIN_SIZE_BYTES) return false
  if (filePath.endsWith('.gz') || filePath.endsWith('.br')) return false
  return COMPRESSIBLE_EXTENSIONS.has(path.extname(filePath))
}

async function collectFiles(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true })
  const files = []

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...(await collectFiles(fullPath)))
    } else if (entry.isFile()) {
      files.push(fullPath)
    }
  }

  return files
}

async function compressFile(filePath) {
  const brotliPath = `${filePath}.br`

  await pipeline(
    createReadStream(filePath),
    zlib.createBrotliCompress({
      params: {
        [zlib.constants.BROTLI_PARAM_QUALITY]: 11,
      },
    }),
    createWriteStream(brotliPath),
  )

  const [original, brotli] = await Promise.all([
    fs.stat(filePath),
    fs.stat(brotliPath),
  ])

  return {
    brotliBytes: brotli.size,
    originalBytes: original.size,
  }
}

async function removeGeneratedCompressionFiles(files) {
  await Promise.all(
    files
      .filter((file) => file.endsWith('.br') || file.endsWith('.gz'))
      .map((file) => fs.rm(file, { force: true })),
  )
}

async function main() {
  const files = await collectFiles(DIST_DIR)
  await removeGeneratedCompressionFiles(files)
  const targets = []

  for (const file of files) {
    const stat = await fs.stat(file)
    if (shouldCompress(file, stat.size)) {
      targets.push(file)
    }
  }

  let originalBytes = 0
  let brotliBytes = 0

  for (const file of targets) {
    const result = await compressFile(file)
    originalBytes += result.originalBytes
    brotliBytes += result.brotliBytes
  }

  const formatKB = (bytes) => `${(bytes / 1024).toFixed(1)} kB`
  console.log(
    `Compressed ${targets.length} assets: ${formatKB(originalBytes)} -> br ${formatKB(
      brotliBytes,
    )}`,
  )
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
