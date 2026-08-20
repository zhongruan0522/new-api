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
import {
  BookOpenCheck,
  Braces,
  Code2,
  Database,
  FileCode,
  Globe,
  type LucideIcon,
  PanelTopOpen,
  ScanEye,
  Settings2,
  Sparkles,
  Workflow,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import type { ModelCapability } from '../types'

type CapabilityMeta = {
  icon: LucideIcon
  labelKey: string
  descriptionKey: string
}

const CAPABILITY_META: Record<ModelCapability, CapabilityMeta> = {
  function_calling: {
    icon: Workflow,
    labelKey: 'common.fields.functionCalling',
    descriptionKey:
      'common.tips.invokeDeveloperDefinedFunctionsWithStructuredArguments',
  },
  streaming: {
    icon: Zap,
    labelKey: 'common.fields.streaming',
    descriptionKey: 'common.tips.streamTokensIncrementallyAsTheyAreGenerated',
  },
  vision: {
    icon: ScanEye,
    labelKey: 'common.fields.vision',
    descriptionKey: 'common.tips.understandImageInputsAlongsideText',
  },
  json_mode: {
    icon: Braces,
    labelKey: 'common.fields.jsonMode',
    descriptionKey: 'common.tips.forceASyntacticallyValidJsonResponse',
  },
  structured_output: {
    icon: FileCode,
    labelKey: 'common.fields.structuredOutput',
    descriptionKey: 'common.tips.returnDataConformingToAJsonSchema',
  },
  reasoning: {
    icon: Sparkles,
    labelKey: 'common.fields.reasoning',
    descriptionKey: 'common.tips.multiStepThinkingBeforeFinalAnswer',
  },
  tools: {
    icon: Settings2,
    labelKey: 'common.fields.tools',
    descriptionKey: 'common.actions.useExternalToolsToExtendCapabilities',
  },
  system_prompt: {
    icon: PanelTopOpen,
    labelKey: 'usageLogs.titles.systemPrompt',
    descriptionKey: 'common.tips.steerBehaviourWithASystemInstruction',
  },
  web_search: {
    icon: Globe,
    labelKey: 'common.fields.webSearch',
    descriptionKey: 'common.actions.searchThePublicWebAtInferenceTime',
  },
  code_interpreter: {
    icon: Code2,
    labelKey: 'common.fields.codeInterpreter',
    descriptionKey: 'common.tips.executeCodeInASandboxDuringTheResponse',
  },
  caching: {
    icon: Database,
    labelKey: 'common.fields.promptCaching',
    descriptionKey:
      'common.tips.cacheRepeatedPromptPrefixesForCheaperFasterReuse',
  },
  embeddings: {
    icon: BookOpenCheck,
    labelKey: 'pricing.fields.embeddings',
    descriptionKey: 'common.tips.returnVectorEmbeddingsForInputs',
  },
}

/**
 * Order capabilities for display. We put the most user-facing capabilities
 * first, then the rest. Anything not listed sinks to the bottom in a stable
 * order so the layout looks tidy across models.
 */
const CAPABILITY_ORDER: ModelCapability[] = [
  'streaming',
  'function_calling',
  'tools',
  'json_mode',
  'structured_output',
  'vision',
  'reasoning',
  'caching',
  'system_prompt',
  'web_search',
  'code_interpreter',
  'embeddings',
]

function orderCapabilities(capabilities: ModelCapability[]): ModelCapability[] {
  const set = new Set(capabilities)
  const ordered = CAPABILITY_ORDER.filter((c) => set.has(c))
  for (const c of capabilities) {
    if (!ordered.includes(c)) ordered.push(c)
  }
  return ordered
}

export function ModelDetailsCapabilities(props: {
  capabilities: ModelCapability[]
}) {
  const { t } = useTranslation()
  const ordered = orderCapabilities(props.capabilities)

  if (ordered.length === 0) {
    return (
      <p className='text-muted-foreground text-sm'>
        {t('pricing.tips.noCapabilitiesReportedForThisModel')}
      </p>
    )
  }

  return (
    <div className='grid grid-cols-2 gap-2 @md/details:grid-cols-3 @2xl/details:grid-cols-4'>
      {ordered.map((capability) => {
        const meta = CAPABILITY_META[capability]
        if (!meta) return null
        const Icon = meta.icon
        return (
          <div
            key={capability}
            className={cn(
              'group flex items-start gap-2 rounded-lg border p-3 transition-colors',
              'hover:bg-muted/30'
            )}
          >
            <span className='bg-muted text-foreground inline-flex size-7 shrink-0 items-center justify-center rounded-md transition-colors group-hover:bg-emerald-100 group-hover:text-emerald-700 dark:group-hover:bg-emerald-500/20 dark:group-hover:text-emerald-300'>
              <Icon className='size-3.5' />
            </span>
            <div className='min-w-0 flex-1'>
              <div className='text-foreground truncate text-xs font-semibold'>
                {t(meta.labelKey)}
              </div>
              <p className='text-muted-foreground mt-0.5 line-clamp-2 text-[11px] leading-snug'>
                {t(meta.descriptionKey)}
              </p>
            </div>
          </div>
        )
      })}
    </div>
  )
}
