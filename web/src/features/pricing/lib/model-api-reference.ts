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
import type { PricingModel } from '../types'

export type SupportedParameter = {
  name: string
  type:
    | 'number'
    | 'integer'
    | 'boolean'
    | 'string'
    | 'object'
    | 'array'
    | 'enum'
  defaultValue?: string | number | boolean
  range?: string
  enumValues?: string[]
  descriptionKey: string
  required?: boolean
}

const COMMON_CHAT_PARAMS: SupportedParameter[] = [
  {
    name: 'temperature',
    type: 'number',
    defaultValue: 1,
    range: '0 ~ 2',
    descriptionKey: 'common.tips.samplingTemperatureLowerIsMoreDeterministic',
  },
  {
    name: 'top_p',
    type: 'number',
    defaultValue: 1,
    range: '0 ~ 1',
    descriptionKey: 'common.tips.nucleusSamplingProbabilityMass',
  },
  {
    name: 'max_tokens',
    type: 'integer',
    range: '>= 1',
    descriptionKey: 'common.tips.maximumNumberOfTokensInTheResponse',
  },
  {
    name: 'frequency_penalty',
    type: 'number',
    defaultValue: 0,
    range: '-2 ~ 2',
    descriptionKey: 'common.tips.penalisesRepetitionOfFrequentTokens',
  },
  {
    name: 'presence_penalty',
    type: 'number',
    defaultValue: 0,
    range: '-2 ~ 2',
    descriptionKey: 'common.tips.encouragesIntroducingNewTopics',
  },
  {
    name: 'stop',
    type: 'array',
    descriptionKey: 'common.tips.upTo4StringsThatStopGeneration',
  },
  {
    name: 'stream',
    type: 'boolean',
    defaultValue: false,
    descriptionKey: 'common.tips.streamTokensViaServerSentEvents',
  },
  {
    name: 'response_format',
    type: 'object',
    descriptionKey: 'common.tips.forceJsonObjectOrSchemaConformingOutput',
  },
  {
    name: 'tools',
    type: 'array',
    descriptionKey: 'common.tips.toolFunctionDeclarationsTheModelMayCall',
  },
  {
    name: 'tool_choice',
    type: 'string',
    enumValues: ['auto', 'none', 'required'],
    descriptionKey: 'common.tips.toolChoicePolicyOrSpecificToolName',
  },
  {
    name: 'user',
    type: 'string',
    descriptionKey: 'common.tips.endUserIdentifierForAbuseMonitoring',
  },
]

const REASONING_PARAMS: SupportedParameter[] = [
  {
    name: 'reasoning_effort',
    type: 'enum',
    enumValues: ['low', 'medium', 'high'],
    defaultValue: 'medium',
    descriptionKey: 'common.tips.controlsHowMuchTheModelThinksBeforeAnswering',
  },
  {
    name: 'max_completion_tokens',
    type: 'integer',
    range: '>= 1',
    descriptionKey: 'common.tips.maximumTokensIncludingHiddenReasoningTokens',
  },
  {
    name: 'stream',
    type: 'boolean',
    defaultValue: false,
    descriptionKey: 'common.tips.streamTokensViaServerSentEvents',
  },
  {
    name: 'response_format',
    type: 'object',
    descriptionKey: 'common.tips.forceJsonObjectOrSchemaConformingOutput',
  },
  {
    name: 'tools',
    type: 'array',
    descriptionKey: 'common.tips.toolFunctionDeclarationsTheModelMayCall',
  },
  {
    name: 'tool_choice',
    type: 'string',
    enumValues: ['auto', 'none', 'required'],
    descriptionKey: 'common.tips.toolChoicePolicyOrSpecificToolName',
  },
]

const EMBEDDING_PARAMS: SupportedParameter[] = [
  {
    name: 'input',
    type: 'string',
    required: true,
    descriptionKey: 'common.fields.textOrArrayOfTextsToEmbed',
  },
  {
    name: 'dimensions',
    type: 'integer',
    range: '>= 1',
    descriptionKey: 'common.tips.truncateEmbeddingsToThisManyDimensions',
  },
  {
    name: 'encoding_format',
    type: 'enum',
    enumValues: ['float', 'base64'],
    defaultValue: 'float',
    descriptionKey: 'common.tips.wireEncodingForTheEmbeddingVectors',
  },
]

const IMAGE_PARAMS: SupportedParameter[] = [
  {
    name: 'prompt',
    type: 'string',
    required: true,
    descriptionKey: 'common.tips.textDescriptionOfTheDesiredImage',
  },
  {
    name: 'size',
    type: 'enum',
    enumValues: ['256x256', '512x512', '1024x1024', '1024x1792', '1792x1024'],
    defaultValue: '1024x1024',
    descriptionKey: 'common.fields.outputImageSize',
  },
  {
    name: 'quality',
    type: 'enum',
    enumValues: ['standard', 'hd'],
    defaultValue: 'standard',
    descriptionKey: 'common.fields.generationQualityPreset',
  },
  {
    name: 'n',
    type: 'integer',
    defaultValue: 1,
    range: '1 ~ 10',
    descriptionKey: 'common.fields.numberOfImagesToGenerate',
  },
]

const VIDEO_PARAMS: SupportedParameter[] = [
  {
    name: 'prompt',
    type: 'string',
    required: true,
    descriptionKey: 'common.tips.textDescriptionOfTheDesiredVideo',
  },
  {
    name: 'duration',
    type: 'integer',
    range: '1 ~ 60',
    descriptionKey: 'common.fields.videoLengthInSeconds',
  },
  {
    name: 'aspect_ratio',
    type: 'enum',
    enumValues: ['16:9', '9:16', '1:1'],
    defaultValue: '16:9',
    descriptionKey: 'common.fields.outputAspectRatio',
  },
]

type ApiCategory = 'reasoning' | 'embedding' | 'image' | 'video' | 'chat'

function apiCategoryOf(model: PricingModel): ApiCategory {
  const endpoints = model.supported_endpoint_types ?? []
  const name = model.model_name ?? ''
  if (endpoints.includes('embeddings') || endpoints.includes('jina-rerank')) {
    return 'embedding'
  }
  if (endpoints.includes('image-generation')) return 'image'
  if (endpoints.includes('openai-video')) return 'video'
  if (/^o[1-4](?:[-:_].+)?$|reasoning|thinking|qwq|deepseek-r/i.test(name)) {
    return 'reasoning'
  }
  if (/sora|veo|kling|pika|video|wan-|hunyuanvideo/i.test(name)) {
    return 'video'
  }
  if (/image|dall|imagen|jimeng/i.test(name)) return 'image'
  return 'chat'
}

export function buildSupportedParameters(
  model: PricingModel
): SupportedParameter[] {
  const category = apiCategoryOf(model)
  if (category === 'reasoning') return REASONING_PARAMS
  if (category === 'embedding') return EMBEDDING_PARAMS
  if (category === 'image') return IMAGE_PARAMS
  if (category === 'video') return VIDEO_PARAMS
  return COMMON_CHAT_PARAMS
}
