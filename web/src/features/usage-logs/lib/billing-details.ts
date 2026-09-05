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
import { z } from 'zod'
import type { UsageLog } from '../data/schema'
import type { LogOtherData } from '../types'

export const BILLING_TOKEN_FIELDS = [
  'text_input',
  'image_input',
  'audio_input',
  'video_input',
  'document_input',
  'text_output',
  'audio_output',
  'image_output',
  'reasoning_output',
  'accepted_prediction',
  'rejected_prediction',
  'read_cache',
  'write_cache',
  'write_cache_5m',
  'write_cache_1h',
] as const

export type BillingTokenField = (typeof BILLING_TOKEN_FIELDS)[number]
export type BillingDetailsErrorCode =
  | 'malformed_json'
  | 'unknown_version'
  | 'invalid_fields'
  | 'invalid_cache_splits'

export type BillingTokenValues = Partial<
  Record<BillingTokenField, number | null>
>

export type BillingDetails =
  | { status: 'legacy' }
  | {
      status: 'invalid'
      code: BillingDetailsErrorCode
      errorKey: string
    }
  | { status: 'valid'; tokens: BillingTokenValues }

const nonnegativeToken = z
  .number()
  .int()
  .nonnegative()
  .refine(Number.isSafeInteger)
  .nullable()
  .optional()

const billingDetailsSchema = z
  .object({
    schema_version: z.literal(1),
    tokens: z
      .object({
        input: z
          .object({
            text_input: nonnegativeToken,
            image_input: nonnegativeToken,
            audio_input: nonnegativeToken,
            video_input: nonnegativeToken,
            document_input: nonnegativeToken,
          })
          .strict(),
        output: z
          .object({
            text_output: nonnegativeToken,
            audio_output: nonnegativeToken,
            image_output: nonnegativeToken,
            reasoning_output: nonnegativeToken,
            accepted_prediction: nonnegativeToken,
            rejected_prediction: nonnegativeToken,
          })
          .strict(),
        cache: z
          .object({
            read_cache: nonnegativeToken,
            write_cache: nonnegativeToken,
            write_cache_5m: nonnegativeToken,
            write_cache_1h: nonnegativeToken,
          })
          .strict(),
      })
      .strict(),
  })
  .strict()

const parseCache = new Map<string, BillingDetails>()
const parseCacheLimit = 256

function cacheResult(raw: string, result: BillingDetails): BillingDetails {
  parseCache.delete(raw)
  parseCache.set(raw, result)
  if (parseCache.size > parseCacheLimit) {
    const oldest = parseCache.keys().next().value
    if (oldest !== undefined) parseCache.delete(oldest)
  }
  return result
}

function invalid(code: BillingDetailsErrorCode): BillingDetails {
  return {
    status: 'invalid',
    code,
    errorKey: `usageLogs.errors.billingDetails.${code}`,
  }
}

export function parseBillingDetails(raw: unknown): BillingDetails {
  if (raw == null || raw === '') return { status: 'legacy' }
  if (typeof raw !== 'string') return invalid('invalid_fields')

  const cached = parseCache.get(raw)
  if (cached) return cached

  let data: unknown
  try {
    data = JSON.parse(raw)
  } catch {
    return cacheResult(raw, invalid('malformed_json'))
  }

  const parsed = billingDetailsSchema.safeParse(data)
  if (!parsed.success) {
    const unknownVersion = parsed.error.issues.some(
      (issue) => issue.path[0] === 'schema_version'
    )
    return cacheResult(
      raw,
      unknownVersion ? invalid('unknown_version') : invalid('invalid_fields')
    )
  }

  const tokens: BillingTokenValues = {}
  for (const group of [
    parsed.data.tokens.input,
    parsed.data.tokens.output,
    parsed.data.tokens.cache,
  ]) {
    for (const [field, value] of Object.entries(group)) {
      if (value != null) {
        tokens[field as BillingTokenField] = value
      }
    }
  }
  const writeCache = tokens.write_cache
  const writeCache5m = tokens.write_cache_5m
  const writeCache1h = tokens.write_cache_1h
  if (
    (writeCache == null && (writeCache5m != null || writeCache1h != null)) ||
    (writeCache != null &&
      (writeCache5m ?? 0) + (writeCache1h ?? 0) > writeCache)
  ) {
    return cacheResult(raw, invalid('invalid_cache_splits'))
  }

  return cacheResult(raw, { status: 'valid', tokens })
}

export function getBillingToken(
  tokens: BillingTokenValues | undefined,
  field: BillingTokenField
): number | null {
  const value = tokens?.[field]
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
    ? value
    : null
}

export function hasOfficialBillingTokens(
  tokens: BillingTokenValues | undefined
): boolean {
  return BILLING_TOKEN_FIELDS.some((field) => tokens?.[field] != null)
}

export function hasOfficialCacheTokens(
  tokens: BillingTokenValues | undefined
): boolean {
  return (
    tokens?.read_cache != null ||
    tokens?.write_cache != null ||
    tokens?.write_cache_5m != null ||
    tokens?.write_cache_1h != null
  )
}

export interface DisplayTokenValues {
  input: number | null
  output: number | null
  cacheRead: number | null
  cacheWrite: number | null
  cacheWriteUnallocated: number | null
  cacheWrite5m: number | null
  cacheWrite1h: number | null
  textInput: number | null
  imageInput: number | null
  audioInput: number | null
  videoInput: number | null
  documentInput: number | null
  textOutput: number | null
  audioOutput: number | null
  imageOutput: number | null
  reasoningOutput: number | null
  acceptedPrediction: number | null
  rejectedPrediction: number | null
  fromBillingDetails: boolean
  hasValues: boolean
}

function legacyCacheCreationTotal(other: LogOtherData | null): number {
  if (!other) return 0
  const splitTotal =
    (other.cache_creation_tokens_5m || 0) +
    (other.cache_creation_tokens_1h || 0)
  return splitTotal > 0 ? splitTotal : other.cache_creation_tokens || 0
}

function legacyOrdinaryInput(log: UsageLog, other: LogOtherData | null) {
  if ((other?.audio || other?.ws) && other.text_input != null) {
    return Math.max(other.text_input, 0)
  }

  const cacheRead = other?.cache_tokens ?? 0
  const cacheCreation = legacyCacheCreationTotal(other)
  const audioInput = other?.audio_input_seperate_price
    ? other.audio_input_token_count || 0
    : 0
  return Math.max(
    (log.prompt_tokens || 0) - cacheRead - cacheCreation - audioInput,
    0
  )
}

export function resolveDisplayTokens(
  log: UsageLog,
  billing: BillingDetails,
  other: LogOtherData | null
): DisplayTokenValues {
  if (billing.status !== 'valid') {
    const cacheRead = other?.cache_tokens || 0
    const cacheWrite = legacyCacheCreationTotal(other)
    const audioInput =
      other?.audio_input ??
      (other?.audio_input_seperate_price
        ? other.audio_input_token_count || 0
        : 0)
    return {
      input: legacyOrdinaryInput(log, other),
      output: log.completion_tokens || 0,
      cacheRead,
      cacheWrite,
      cacheWriteUnallocated: 0,
      cacheWrite5m: other?.cache_creation_tokens_5m || 0,
      cacheWrite1h: other?.cache_creation_tokens_1h || 0,
      textInput: other?.text_input ?? null,
      imageInput: null,
      audioInput,
      videoInput: null,
      documentInput: null,
      textOutput: other?.text_output ?? null,
      audioOutput: other?.audio_output || 0,
      imageOutput: other?.image_output || 0,
      reasoningOutput: null,
      acceptedPrediction: null,
      rejectedPrediction: null,
      fromBillingDetails: false,
      hasValues: Boolean(
        log.prompt_tokens ||
        log.completion_tokens ||
        cacheRead ||
        cacheWrite ||
        audioInput ||
        other?.audio_output ||
        other?.image_output ||
        other?.text_input
      ),
    }
  }

  return {
    input: getBillingToken(billing.tokens, 'text_input'),
    output: getBillingToken(billing.tokens, 'text_output'),
    cacheRead: getBillingToken(billing.tokens, 'read_cache'),
    cacheWrite: getBillingToken(billing.tokens, 'write_cache'),
    cacheWriteUnallocated: officialCacheWriteUnallocated(billing.tokens),
    cacheWrite5m: getBillingToken(billing.tokens, 'write_cache_5m'),
    cacheWrite1h: getBillingToken(billing.tokens, 'write_cache_1h'),
    textInput: getBillingToken(billing.tokens, 'text_input'),
    imageInput: getBillingToken(billing.tokens, 'image_input'),
    audioInput: getBillingToken(billing.tokens, 'audio_input'),
    videoInput: getBillingToken(billing.tokens, 'video_input'),
    documentInput: getBillingToken(billing.tokens, 'document_input'),
    textOutput: getBillingToken(billing.tokens, 'text_output'),
    audioOutput: getBillingToken(billing.tokens, 'audio_output'),
    imageOutput: getBillingToken(billing.tokens, 'image_output'),
    reasoningOutput: getBillingToken(billing.tokens, 'reasoning_output'),
    acceptedPrediction: getBillingToken(billing.tokens, 'accepted_prediction'),
    rejectedPrediction: getBillingToken(billing.tokens, 'rejected_prediction'),
    fromBillingDetails: true,
    hasValues: hasOfficialBillingTokens(billing.tokens),
  }
}

function officialCacheWriteUnallocated(
  tokens: BillingTokenValues
): number | null {
  const total = getBillingToken(tokens, 'write_cache')
  if (total == null) return null
  const allocated =
    (getBillingToken(tokens, 'write_cache_5m') ?? 0) +
    (getBillingToken(tokens, 'write_cache_1h') ?? 0)
  return Math.max(total - allocated, 0)
}
