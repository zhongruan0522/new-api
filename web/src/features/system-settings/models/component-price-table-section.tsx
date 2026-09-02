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
  type ReactNode,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, Copy, Loader2, Plus, Save, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { formatDateTimeStr } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { getEnabledModels } from '@/features/channels/api'
import {
  getModelPriceTableConfiguration,
  updateModelPriceTableConfiguration,
} from '../api'
import { SettingsPageActionsPortal } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import type {
  ModelPriceBillingMode,
  ModelPriceComponentName,
  ModelPriceGroupMultiplierSource,
  ModelPricePlan,
  ModelPriceRoundingMode,
  ModelPriceTableConfiguration,
} from '../types'

type DraftPlan = ModelPricePlan & {
  draftKey: string
}

type ComponentDefinition = {
  component: ModelPriceComponentName
  labelKey: string
}

const INPUT_CHILD_COMPONENTS: ModelPriceComponentName[] = [
  'text_input',
  'image_input',
  'audio_input',
  'video_input',
  'document_input',
]

const OUTPUT_CHILD_COMPONENTS: ModelPriceComponentName[] = [
  'text_output',
  'audio_output',
  'image_output',
]

const COMPONENT_GROUPS: Array<{
  titleKey: string
  components: ComponentDefinition[]
}> = [
  {
    titleKey: 'systemSettings.titles.inputComponents',
    components: [
      {
        component: 'input',
        labelKey: 'systemSettings.fields.priceComponentInput',
      },
      {
        component: 'text_input',
        labelKey: 'systemSettings.fields.priceComponentTextInput',
      },
      {
        component: 'image_input',
        labelKey: 'systemSettings.fields.priceComponentImageInput',
      },
      {
        component: 'audio_input',
        labelKey: 'systemSettings.fields.priceComponentAudioInput',
      },
      {
        component: 'video_input',
        labelKey: 'systemSettings.fields.priceComponentVideoInput',
      },
      {
        component: 'document_input',
        labelKey: 'systemSettings.fields.priceComponentDocumentInput',
      },
    ],
  },
  {
    titleKey: 'systemSettings.titles.outputComponents',
    components: [
      {
        component: 'output',
        labelKey: 'systemSettings.fields.priceComponentOutput',
      },
      {
        component: 'text_output',
        labelKey: 'systemSettings.fields.priceComponentTextOutput',
      },
      {
        component: 'audio_output',
        labelKey: 'systemSettings.fields.priceComponentAudioOutput',
      },
      {
        component: 'image_output',
        labelKey: 'systemSettings.fields.priceComponentImageOutput',
      },
    ],
  },
  {
    titleKey: 'systemSettings.titles.cacheComponents',
    components: [
      {
        component: 'cache_read',
        labelKey: 'systemSettings.fields.priceComponentCacheRead',
      },
      {
        component: 'cache_write_5m',
        labelKey: 'systemSettings.fields.priceComponentCacheWrite5m',
      },
      {
        component: 'cache_write_1h',
        labelKey: 'systemSettings.fields.priceComponentCacheWrite1h',
      },
    ],
  },
]

const DECIMAL_PATTERN = /^(?:\d+(?:\.\d+)?|\.\d+)$/
const DEFAULT_CONTEXT_TIER_WIDTH = 200000
const MAX_MODEL_NAME_LENGTH = 255
const MAX_SCOPE_LENGTH = 128
const MAX_PRICE_PLAN_PRECISION = 18
const MAX_PRICE_PLAN_VALUE = 1_000_000_000_000

function createEmptyPricePlan(): ModelPricePlan {
  return {
    model_name: '',
    endpoint: '',
    effective_group: '',
    service_tier: '',
    context_min_tokens: 0,
    billing_mode: 'token',
    currency: 'USD',
    exchange_rate: '1',
    price_precision: 12,
    rounding_mode: 'half_up',
    group_multiplier_source: 'inherit_group_ratio',
    group_multiplier: '',
    components: [],
  }
}

function normalizePriceTableConfiguration(
  configuration: ModelPriceTableConfiguration
): ModelPriceTableConfiguration {
  return {
    ...configuration,
    plans: configuration.plans ?? [],
    legacy_plans: configuration.legacy_plans ?? [],
  }
}

function clonePlan(plan: ModelPricePlan): ModelPricePlan {
  return {
    ...plan,
    endpoint: plan.endpoint ?? '',
    effective_group: plan.effective_group ?? '',
    service_tier: plan.service_tier ?? '',
    group_multiplier: plan.group_multiplier ?? '',
    components: (plan.components ?? []).map((component) => ({ ...component })),
  }
}

function toRequestPlan(plan: DraftPlan): ModelPricePlan {
  const {
    draftKey: _draftKey,
    id: _id,
    source: _source,
    read_only: _readOnly,
    created_at: _createdAt,
    updated_at: _updatedAt,
    ...request
  } = plan
  return {
    ...request,
    model_name: request.model_name.trim(),
    endpoint: request.endpoint?.trim() || undefined,
    effective_group: request.effective_group?.trim() || undefined,
    service_tier: request.service_tier?.trim() || undefined,
    currency: request.currency.trim().toUpperCase(),
    exchange_rate: request.exchange_rate.trim(),
    group_multiplier:
      request.group_multiplier_source === 'fixed'
        ? request.group_multiplier?.trim()
        : undefined,
    components: request.components.map((component) => ({
      ...component,
      unit_price: component.unit_price.trim(),
    })),
  }
}

function componentLabelKey(component: ModelPriceComponentName) {
  if (component === 'request') {
    return 'systemSettings.fields.priceComponentRequest'
  }
  for (const group of COMPONENT_GROUPS) {
    const definition = group.components.find(
      (candidate) => candidate.component === component
    )
    if (definition) return definition.labelKey
  }
  return 'systemSettings.fields.unknownPriceComponent'
}

function getComponent(
  plan: ModelPricePlan,
  component: ModelPriceComponentName
) {
  return plan.components.find((item) => item.component === component)
}

function formatPlanSummary(plan: ModelPricePlan, t: (key: string) => string) {
  const scopes = [] as string[]
  if (
    plan.context_min_tokens !== 0 ||
    plan.context_max_tokens !== undefined
  ) {
    const contextRange =
      plan.context_max_tokens === undefined
        ? `${plan.context_min_tokens}+`
        : `${plan.context_min_tokens} - ${plan.context_max_tokens}`
    scopes.push(
      `${t('systemSettings.fields.contextMinTokens')}: ${contextRange}`
    )
  }
  if (plan.effective_from !== undefined) {
    scopes.push(
      `${t('systemSettings.fields.effectiveFrom')}: ${formatDateTimeStr(
        new Date(plan.effective_from * 1000)
      )}`
    )
  }
  if (plan.effective_until !== undefined) {
    scopes.push(
      `${t('systemSettings.fields.effectiveUntil')}: ${formatDateTimeStr(
        new Date(plan.effective_until * 1000)
      )}`
    )
  }
  if (plan.endpoint) {
    scopes.push(`${t('systemSettings.fields.endpoint')}: ${plan.endpoint}`)
  }
  if (plan.effective_group) {
    scopes.push(
      `${t('systemSettings.fields.effectiveGroup')}: ${plan.effective_group}`
    )
  }
  if (plan.service_tier) {
    scopes.push(
      `${t('systemSettings.fields.serviceTier')}: ${plan.service_tier}`
    )
  }
  if (scopes.length > 0) return scopes.join(' · ')
  return t('systemSettings.tips.defaultPriceScope')
}

function timestampToLocalInput(timestamp?: number) {
  if (timestamp === undefined) return ''
  const date = new Date(timestamp * 1000)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function localInputToTimestamp(value: string) {
  if (!value) return undefined
  const timestamp = new Date(value).getTime()
  return Number.isNaN(timestamp) ? undefined : Math.floor(timestamp / 1000)
}

function updateTokenComponent<T extends ModelPricePlan>(
  plan: T,
  component: ModelPriceComponentName,
  checked: boolean
): T {
  let components = plan.components.filter(
    (item) => item.component !== component
  )
  if (!checked) return { ...plan, components }

  if (component === 'input') {
    components = components.filter(
      (item) => !INPUT_CHILD_COMPONENTS.includes(item.component)
    )
  } else if (INPUT_CHILD_COMPONENTS.includes(component)) {
    components = components.filter((item) => item.component !== 'input')
  }
  if (component === 'output') {
    components = components.filter(
      (item) => !OUTPUT_CHILD_COMPONENTS.includes(item.component)
    )
  } else if (OUTPUT_CHILD_COMPONENTS.includes(component)) {
    components = components.filter((item) => item.component !== 'output')
  }

  return {
    ...plan,
    components: [
      ...components,
      {
        component,
        unit: 'per_1m_tokens',
        unit_price: '',
      },
    ],
  }
}

function updateComponentPrice<T extends ModelPricePlan>(
  plan: T,
  component: ModelPriceComponentName,
  unitPrice: string
): T {
  return {
    ...plan,
    components: plan.components.map((item) =>
      item.component === component ? { ...item, unit_price: unitPrice } : item
    ),
  }
}

function updateBillingMode<T extends ModelPricePlan>(
  plan: T,
  billingMode: ModelPriceBillingMode
): T {
  if (billingMode === 'token') {
    return { ...plan, billing_mode: billingMode, components: [] }
  }
  if (billingMode === 'per_request') {
    return {
      ...plan,
      billing_mode: billingMode,
      components: [
        {
          component: 'request',
          unit: 'per_request',
          unit_price: '',
        },
      ],
    }
  }
  return { ...plan, billing_mode: billingMode, components: [] }
}

function hasParentChildConflict(plan: ModelPricePlan) {
  const components = new Set(plan.components.map((item) => item.component))
  return (
    (components.has('input') &&
      INPUT_CHILD_COMPONENTS.some((component) => components.has(component))) ||
    (components.has('output') &&
      OUTPUT_CHILD_COMPONENTS.some((component) => components.has(component)))
  )
}

function isValidDecimal(value: string, scale: number, positive = false) {
  const trimmed = value.trim()
  if (!DECIMAL_PATTERN.test(trimmed)) return false
  const decimal = Number(trimmed)
  if (
    !Number.isFinite(decimal) ||
    Math.abs(decimal) > MAX_PRICE_PLAN_VALUE ||
    (positive ? decimal <= 0 : decimal < 0)
  ) {
    return false
  }
  const fraction = trimmed.split('.')[1]
  return !fraction || fraction.length <= scale
}

function isValidPricePlanScope(value: string) {
  const trimmed = value.trim()
  for (const character of trimmed) {
    const code = character.charCodeAt(0)
    if (character === ',' || code <= 0x1f || code === 0x7f) {
      return false
    }
  }
  return new TextEncoder().encode(trimmed).length <= MAX_SCOPE_LENGTH
}

function rangesOverlap(
  leftStart: number,
  leftEnd: number | undefined,
  rightStart: number,
  rightEnd: number | undefined
) {
  if (leftEnd !== undefined && leftEnd <= rightStart) return false
  if (rightEnd !== undefined && rightEnd <= leftStart) return false
  return true
}

function validateDraftPlans(plans: DraftPlan[], t: (key: string) => string) {
  const errors: string[] = []
  for (const [index, plan] of plans.entries()) {
    const label = `${t('systemSettings.fields.pricePlan')} ${index + 1}`
    if (!plan.model_name.trim()) {
      errors.push(
        `${label}: ${t('systemSettings.errors.priceTableModelRequired')}`
      )
    } else if (
      new TextEncoder().encode(plan.model_name.trim()).length >
      MAX_MODEL_NAME_LENGTH
    ) {
      errors.push(
        `${label}: ${t('systemSettings.errors.priceTableModelTooLong')}`
      )
    }
    for (const value of [
      plan.endpoint,
      plan.effective_group,
      plan.service_tier,
    ]) {
      if (value && !isValidPricePlanScope(value)) {
        errors.push(
          `${label}: ${t('systemSettings.errors.priceTableScopeInvalid')}`
        )
        break
      }
    }
    if (
      !Number.isInteger(plan.context_min_tokens) ||
      plan.context_min_tokens < 0 ||
      (plan.context_max_tokens !== undefined &&
        (!Number.isInteger(plan.context_max_tokens) ||
          plan.context_max_tokens <= plan.context_min_tokens))
    ) {
      errors.push(
        `${label}: ${t('systemSettings.errors.priceTableContextRangeInvalid')}`
      )
    }
    if (
      (plan.effective_from !== undefined &&
        (!Number.isInteger(plan.effective_from) || plan.effective_from < 0)) ||
      (plan.effective_until !== undefined &&
        (!Number.isInteger(plan.effective_until) ||
          plan.effective_until < 0)) ||
      (plan.effective_from !== undefined &&
        plan.effective_until !== undefined &&
        plan.effective_until <= plan.effective_from)
    ) {
      errors.push(
        `${label}: ${t('systemSettings.errors.priceTableTimeRangeInvalid')}`
      )
    }
    if (!/^[A-Za-z]{3}$/.test(plan.currency.trim())) {
      errors.push(
        `${label}: ${t('systemSettings.errors.priceTableCurrencyInvalid')}`
      )
    }
    if (
      !Number.isInteger(plan.price_precision) ||
      plan.price_precision < 0 ||
      plan.price_precision > MAX_PRICE_PLAN_PRECISION
    ) {
      errors.push(
        `${label}: ${t('systemSettings.errors.priceTablePrecisionInvalid')}`
      )
    }
    if (!isValidDecimal(plan.exchange_rate, plan.price_precision, true)) {
      errors.push(
        `${label}: ${t('systemSettings.errors.priceTableExchangeRateInvalid')}`
      )
    }
    if (
      plan.group_multiplier_source === 'fixed' &&
      !isValidDecimal(plan.group_multiplier ?? '', plan.price_precision)
    ) {
      errors.push(
        `${label}: ${t('systemSettings.errors.priceTableMultiplierInvalid')}`
      )
    }

    if (plan.billing_mode === 'free' && plan.components.length > 0) {
      errors.push(
        `${label}: ${t('systemSettings.errors.priceTableFreeModeInvalid')}`
      )
    }
    if (plan.billing_mode === 'per_request') {
      const request = getComponent(plan, 'request')
      if (
        plan.components.length !== 1 ||
        !request ||
        request.unit !== 'per_request' ||
        !isValidDecimal(request.unit_price, plan.price_precision, true)
      ) {
        errors.push(
          `${label}: ${t('systemSettings.errors.priceTableRequestPriceInvalid')}`
        )
      }
    }
    if (plan.billing_mode === 'token') {
      if (plan.components.length === 0) {
        errors.push(
          `${label}: ${t('systemSettings.errors.priceTableTokenComponentRequired')}`
        )
      }
      if (hasParentChildConflict(plan)) {
        errors.push(
          `${label}: ${t('systemSettings.errors.priceTableParentChildConflict')}`
        )
      }
      for (const component of plan.components) {
        if (
          component.unit !== 'per_1m_tokens' ||
          !isValidDecimal(component.unit_price, plan.price_precision)
        ) {
          errors.push(
            `${label} / ${t(componentLabelKey(component.component))}: ${t(
              'systemSettings.errors.priceTableTokenPriceInvalid'
            )}`
          )
          break
        }
      }
    }
  }

  for (let left = 0; left < plans.length; left += 1) {
    for (let right = left + 1; right < plans.length; right += 1) {
      const first = plans[left]
      const second = plans[right]
      if (
        first.model_name.trim() !== second.model_name.trim() ||
        (first.endpoint ?? '') !== (second.endpoint ?? '') ||
        (first.effective_group ?? '') !== (second.effective_group ?? '') ||
        (first.service_tier ?? '') !== (second.service_tier ?? '')
      ) {
        continue
      }
      if (
        rangesOverlap(
          first.context_min_tokens,
          first.context_max_tokens,
          second.context_min_tokens,
          second.context_max_tokens
        ) &&
        rangesOverlap(
          first.effective_from ?? 0,
          first.effective_until,
          second.effective_from ?? 0,
          second.effective_until
        )
      ) {
        errors.push(t('systemSettings.errors.priceTablePlansOverlap'))
      }
    }
  }
  return [...new Set(errors)]
}

export function ComponentPriceTableSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const nextDraftID = useRef(0)
  const [plans, setPlans] = useState<DraftPlan[]>([])
  const [selectedDraftKey, setSelectedDraftKey] = useState<string | null>(null)
  const [isDirty, setIsDirty] = useState(false)

  const configurationQuery = useQuery({
    queryKey: ['system-settings', 'model-price-table'],
    queryFn: async () => {
      const response = await getModelPriceTableConfiguration()
      if (!response.success) {
        throw new Error(response.message)
      }
      return normalizePriceTableConfiguration(response.data)
    },
  })

  const enabledModelsQuery = useQuery({
    queryKey: ['channels', 'enabled-models'],
    queryFn: async () => {
      const response = await getEnabledModels()
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data ?? []
    },
    staleTime: 60_000,
  })

  useEffect(() => {
    // Keep an in-progress draft intact when React Query refetches on focus or
    // network recovery. A successful save clears isDirty before its refreshed
    // configuration is applied below.
    if (!configurationQuery.data || isDirty) return
    const nextPlans = configurationQuery.data.plans.map((plan) => ({
      ...clonePlan(plan),
      draftKey: `price-plan-${nextDraftID.current++}`,
    }))
    setPlans(nextPlans)
    setSelectedDraftKey(nextPlans[0]?.draftKey ?? null)
    setIsDirty(false)
  }, [configurationQuery.data, isDirty])

  const saveMutation = useMutation({
    mutationFn: async (nextPlans: DraftPlan[]) => {
      const response = await updateModelPriceTableConfiguration({
        plans: nextPlans.map(toRequestPlan),
      })
      if (!response.success) {
        throw new Error(response.message)
      }
      return response
    },
    onSuccess: async (response) => {
      setIsDirty(false)
      queryClient.setQueryData<ModelPriceTableConfiguration | undefined>(
        ['system-settings', 'model-price-table'],
        (current) =>
          current
            ? { ...current, plans: response.data.plans ?? [] }
            : normalizePriceTableConfiguration(response.data)
      )
      await queryClient.invalidateQueries({
        queryKey: ['system-settings', 'model-price-table'],
      })
      await queryClient.invalidateQueries({ queryKey: ['pricing'] })
      toast.success(
        response.message || t('systemSettings.status.priceTableSaved')
      )
    },
    onError: (error) => {
      toast.error(
        error instanceof Error
          ? error.message || t('systemSettings.errors.priceTableSaveFailed')
          : t('systemSettings.errors.priceTableSaveFailed')
      )
    },
  })

  const selectedPlan = plans.find((plan) => plan.draftKey === selectedDraftKey)
  const validationErrors = useMemo(
    () => validateDraftPlans(plans, t),
    [plans, t]
  )
  const enabledModels = enabledModelsQuery.data ?? []
  const canEditConfiguration =
    configurationQuery.isSuccess && !saveMutation.isPending

  const updatePlan = (
    draftKey: string,
    updater: (plan: DraftPlan) => DraftPlan
  ) => {
    setPlans((current) =>
      current.map((plan) => (plan.draftKey === draftKey ? updater(plan) : plan))
    )
    setIsDirty(true)
  }

  const addPlan = () => {
    if (!canEditConfiguration) return
    const nextPlan: DraftPlan = {
      ...createEmptyPricePlan(),
      draftKey: `price-plan-${nextDraftID.current++}`,
    }
    setPlans((current) => [...current, nextPlan])
    setSelectedDraftKey(nextPlan.draftKey)
    setIsDirty(true)
  }

  const removePlan = (draftKey: string) => {
    if (!canEditConfiguration) return
    setPlans((current) => {
      const nextPlans = current.filter((plan) => plan.draftKey !== draftKey)
      if (selectedDraftKey === draftKey) {
        setSelectedDraftKey(nextPlans[0]?.draftKey ?? null)
      }
      return nextPlans
    })
    setIsDirty(true)
  }

  const addContextTier = () => {
    if (!selectedPlan || !canEditConfiguration) return
    const boundary =
      selectedPlan.context_max_tokens ??
      selectedPlan.context_min_tokens + DEFAULT_CONTEXT_TIER_WIDTH
    const duplicate: DraftPlan = {
      ...clonePlan(selectedPlan),
      id: undefined,
      context_min_tokens: boundary,
      context_max_tokens: undefined,
      draftKey: `price-plan-${nextDraftID.current++}`,
    }
    setPlans((current) =>
      current.flatMap((plan) => {
        if (plan.draftKey !== selectedPlan.draftKey) return [plan]
        const original =
          plan.context_max_tokens === undefined
            ? { ...plan, context_max_tokens: boundary }
            : plan
        return [original, duplicate]
      })
    )
    setSelectedDraftKey(duplicate.draftKey)
    setIsDirty(true)
  }

  const handleSave = () => {
    if (!canEditConfiguration) return
    if (validationErrors.length > 0) {
      toast.error(t('systemSettings.errors.invalidComponentPricePlans'))
      return
    }
    saveMutation.mutate(plans)
  }

  const saveActionLabel = saveMutation.isPending
    ? t('systemSettings.tips.savingComponentPriceTable')
    : t('systemSettings.actions.saveComponentPriceTable')

  return (
    <SettingsSection title={t('systemSettings.titles.componentPriceTable')}>
      <SettingsPageActionsPortal>
        <Button
          type='button'
          size='sm'
          variant='outline'
          className='max-sm:size-8 max-sm:px-0'
          aria-label={t('systemSettings.actions.addPricePlan')}
          title={t('systemSettings.actions.addPricePlan')}
          onClick={addPlan}
          disabled={!canEditConfiguration}
        >
          <Plus data-icon='inline-start' />
          <span className='max-sm:sr-only'>
            {t('systemSettings.actions.addPricePlan')}
          </span>
        </Button>
        <Button
          type='button'
          size='sm'
          className='max-sm:size-8 max-sm:px-0'
          aria-label={saveActionLabel}
          title={saveActionLabel}
          onClick={handleSave}
          disabled={
            !canEditConfiguration || !isDirty || validationErrors.length > 0
          }
        >
          {saveMutation.isPending ? (
            <Loader2 data-icon='inline-start' className='animate-spin' />
          ) : (
            <Save data-icon='inline-start' />
          )}
          <span className='max-sm:sr-only'>{saveActionLabel}</span>
        </Button>
      </SettingsPageActionsPortal>

      {configurationQuery.isPending ? (
        <div className='text-muted-foreground flex min-h-48 items-center justify-center gap-2 text-sm'>
          <Loader2 className='size-4 animate-spin' />
          <span>{t('common.tips.loading')}</span>
        </div>
      ) : configurationQuery.isError ? (
        <Alert variant='destructive'>
          <AlertCircle />
          <AlertTitle>
            {t('systemSettings.errors.priceTableLoadFailed')}
          </AlertTitle>
          <AlertDescription>
            {configurationQuery.error instanceof Error
              ? configurationQuery.error.message ||
                t('systemSettings.errors.priceTableLoadFailed')
              : t('systemSettings.errors.priceTableLoadFailed')}
          </AlertDescription>
        </Alert>
      ) : (
        <>
          <p className='text-muted-foreground text-sm'>
            {t('systemSettings.tips.componentPriceTableFallback')}
          </p>

          {validationErrors.length > 0 ? (
            <Alert variant='destructive' aria-live='polite'>
              <AlertCircle />
              <AlertTitle>
                {t('systemSettings.errors.invalidComponentPricePlans')}
              </AlertTitle>
              <AlertDescription>
                <ul className='list-disc space-y-1 pl-4'>
                  {validationErrors.map((error) => (
                    <li key={error}>{error}</li>
                  ))}
                </ul>
              </AlertDescription>
            </Alert>
          ) : null}

          <div className='grid min-w-0 border lg:grid-cols-[minmax(15rem,22rem)_minmax(0,1fr)]'>
            <div className='border-b lg:border-r lg:border-b-0'>
              <div className='flex items-center justify-between gap-2 border-b px-3 py-2'>
                <h4 className='text-sm font-medium'>
                  {t('systemSettings.titles.pricePlans')}
                </h4>
                <Button
                  type='button'
                  size='icon-xs'
                  variant='ghost'
                  onClick={addPlan}
                  disabled={!canEditConfiguration}
                  aria-label={t('systemSettings.actions.addPricePlan')}
                  title={t('systemSettings.actions.addPricePlan')}
                >
                  <Plus />
                </Button>
              </div>
              {plans.length === 0 ? (
                <div className='text-muted-foreground flex min-h-44 flex-col items-center justify-center gap-3 px-4 text-center text-sm'>
                  <span>{t('systemSettings.tips.noComponentPricePlans')}</span>
                  <Button
                    type='button'
                    size='sm'
                    variant='outline'
                    onClick={addPlan}
                    disabled={!canEditConfiguration}
                  >
                    <Plus data-icon='inline-start' />
                    <span>{t('systemSettings.actions.addPricePlan')}</span>
                  </Button>
                </div>
              ) : (
                <div className='max-h-[34rem] overflow-y-auto p-1.5'>
                  {plans.map((plan, index) => (
                    <div
                      key={plan.draftKey}
                      className={cn(
                        'group flex min-w-0 items-center gap-1 rounded-md',
                        selectedDraftKey === plan.draftKey && 'bg-muted'
                      )}
                    >
                      <button
                        type='button'
                        className='focus-visible:ring-ring/60 min-w-0 flex-1 px-2.5 py-2 text-left outline-none focus-visible:ring-2'
                        onClick={() => setSelectedDraftKey(plan.draftKey)}
                      >
                        <div className='truncate text-sm font-medium'>
                          {plan.model_name ||
                            `${t('systemSettings.fields.pricePlan')} ${index + 1}`}
                        </div>
                        <div className='text-muted-foreground truncate text-xs'>
                          {formatPlanSummary(plan, t)}
                        </div>
                      </button>
                      <Button
                        type='button'
                        size='icon-xs'
                        variant='ghost'
                        className='mr-1 opacity-0 group-hover:opacity-100 focus-visible:opacity-100'
                        onClick={() => removePlan(plan.draftKey)}
                        disabled={!canEditConfiguration}
                        aria-label={t('systemSettings.actions.deletePricePlan')}
                        title={t('systemSettings.actions.deletePricePlan')}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className='min-w-0 p-4'>
              {selectedPlan ? (
                <PricePlanEditor
                  key={selectedPlan.draftKey}
                  plan={selectedPlan}
                  enabledModels={enabledModels}
                  onChange={(updater) =>
                    updatePlan(selectedPlan.draftKey, updater)
                  }
                  onAddContextTier={addContextTier}
                  onDelete={() => removePlan(selectedPlan.draftKey)}
                  disabled={!canEditConfiguration}
                />
              ) : (
                <div className='text-muted-foreground flex min-h-52 items-center justify-center text-center text-sm'>
                  {t('systemSettings.tips.selectComponentPricePlan')}
                </div>
              )}
            </div>
          </div>

          <LegacyPricePlans
            plans={configurationQuery.data?.legacy_plans ?? []}
          />
        </>
      )}
    </SettingsSection>
  )
}

function PricePlanEditor({
  plan,
  enabledModels,
  onChange,
  onAddContextTier,
  onDelete,
  disabled,
}: {
  plan: DraftPlan
  enabledModels: string[]
  onChange: (updater: (plan: DraftPlan) => DraftPlan) => void
  onAddContextTier: () => void
  onDelete: () => void
  disabled: boolean
}) {
  const { t } = useTranslation()
  const modelListID = `model-price-table-models-${plan.draftKey}`

  const update = (patch: Partial<ModelPricePlan>) =>
    onChange((current) => ({ ...current, ...patch }))

  return (
    <div className='space-y-6'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <h4 className='text-sm font-semibold'>
          {t('systemSettings.titles.pricePlanDetails')}
        </h4>
        <div className='flex items-center gap-1.5'>
          <Button
            type='button'
            size='sm'
            variant='outline'
            onClick={onAddContextTier}
            disabled={disabled}
          >
            <Copy data-icon='inline-start' />
            <span>{t('systemSettings.actions.addContextTier')}</span>
          </Button>
          <Button
            type='button'
            size='icon-sm'
            variant='ghost'
            onClick={onDelete}
            disabled={disabled}
            aria-label={t('systemSettings.actions.deletePricePlan')}
            title={t('systemSettings.actions.deletePricePlan')}
          >
            <Trash2 />
          </Button>
        </div>
      </div>

      <div className='grid gap-x-4 gap-y-4 sm:grid-cols-2 xl:grid-cols-3'>
        <PriceField label={t('systemSettings.fields.modelName')}>
          <Input
            list={modelListID}
            value={plan.model_name}
            onChange={(event) => update({ model_name: event.target.value })}
            autoComplete='off'
            disabled={disabled}
          />
          <datalist id={modelListID}>
            {enabledModels.map((model) => (
              <option key={model} value={model} />
            ))}
          </datalist>
        </PriceField>
        <PriceField label={t('systemSettings.fields.endpoint')}>
          <Input
            value={plan.endpoint ?? ''}
            onChange={(event) => update({ endpoint: event.target.value })}
            disabled={disabled}
          />
        </PriceField>
        <PriceField label={t('systemSettings.fields.effectiveGroup')}>
          <Input
            value={plan.effective_group ?? ''}
            onChange={(event) =>
              update({ effective_group: event.target.value })
            }
            disabled={disabled}
          />
        </PriceField>
        <PriceField label={t('systemSettings.fields.serviceTier')}>
          <Input
            value={plan.service_tier ?? ''}
            onChange={(event) => update({ service_tier: event.target.value })}
            disabled={disabled}
          />
        </PriceField>
        <PriceField label={t('systemSettings.fields.contextMinTokens')}>
          <Input
            type='number'
            min={0}
            step={1}
            value={plan.context_min_tokens}
            onChange={(event) =>
              update({ context_min_tokens: Number(event.target.value) || 0 })
            }
            disabled={disabled}
          />
        </PriceField>
        <PriceField label={t('systemSettings.fields.contextMaxTokens')}>
          <Input
            type='number'
            min={0}
            step={1}
            value={plan.context_max_tokens ?? ''}
            onChange={(event) =>
              update({
                context_max_tokens: event.target.value
                  ? Number(event.target.value)
                  : undefined,
              })
            }
            disabled={disabled}
          />
        </PriceField>
        <PriceField label={t('systemSettings.fields.effectiveFrom')}>
          <Input
            type='datetime-local'
            step={1}
            value={timestampToLocalInput(plan.effective_from)}
            onChange={(event) =>
              update({
                effective_from: localInputToTimestamp(event.target.value),
              })
            }
            disabled={disabled}
          />
        </PriceField>
        <PriceField label={t('systemSettings.fields.effectiveUntil')}>
          <Input
            type='datetime-local'
            step={1}
            value={timestampToLocalInput(plan.effective_until)}
            onChange={(event) =>
              update({
                effective_until: localInputToTimestamp(event.target.value),
              })
            }
            disabled={disabled}
          />
        </PriceField>
      </div>

      <div className='space-y-2'>
        <Label>{t('systemSettings.fields.billingMode')}</Label>
        <ToggleGroup
          value={[plan.billing_mode]}
          onValueChange={(values) => {
            const nextMode = values[0] as ModelPriceBillingMode | undefined
            if (nextMode)
              onChange((current) => updateBillingMode(current, nextMode))
          }}
          variant='outline'
          size='sm'
          disabled={disabled}
        >
          <ToggleGroupItem value='token'>
            {t('systemSettings.fields.tokenBilling')}
          </ToggleGroupItem>
          <ToggleGroupItem value='per_request'>
            {t('systemSettings.fields.perRequestBilling')}
          </ToggleGroupItem>
          <ToggleGroupItem value='free'>
            {t('systemSettings.fields.freeBilling')}
          </ToggleGroupItem>
        </ToggleGroup>
      </div>

      <div className='grid gap-x-4 gap-y-4 sm:grid-cols-2 xl:grid-cols-3'>
        <PriceField label={t('systemSettings.fields.currency')}>
          <Input
            value={plan.currency}
            maxLength={3}
            onChange={(event) => update({ currency: event.target.value })}
            disabled={disabled}
          />
        </PriceField>
        <PriceField label={t('systemSettings.fields.exchangeRate')}>
          <Input
            inputMode='decimal'
            value={plan.exchange_rate}
            onChange={(event) => update({ exchange_rate: event.target.value })}
            disabled={disabled}
          />
        </PriceField>
        <PriceField label={t('systemSettings.fields.pricePrecision')}>
          <Input
            type='number'
            min={0}
            max={MAX_PRICE_PLAN_PRECISION}
            step={1}
            value={plan.price_precision}
            onChange={(event) =>
              update({ price_precision: Number(event.target.value) || 0 })
            }
            disabled={disabled}
          />
        </PriceField>
        <PriceField label={t('systemSettings.fields.roundingMode')}>
          <Select
            value={plan.rounding_mode}
            onValueChange={(value) =>
              update({ rounding_mode: value as ModelPriceRoundingMode })
            }
            disabled={disabled}
          >
            <SelectTrigger className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='half_up'>
                {t('systemSettings.fields.roundHalfUp')}
              </SelectItem>
              <SelectItem value='half_even'>
                {t('systemSettings.fields.roundHalfEven')}
              </SelectItem>
              <SelectItem value='floor'>
                {t('systemSettings.fields.roundFloor')}
              </SelectItem>
              <SelectItem value='ceil'>
                {t('systemSettings.fields.roundCeil')}
              </SelectItem>
            </SelectContent>
          </Select>
        </PriceField>
        <PriceField label={t('systemSettings.fields.groupMultiplierSource')}>
          <Select
            value={plan.group_multiplier_source}
            onValueChange={(value) =>
              update({
                group_multiplier_source:
                  value as ModelPriceGroupMultiplierSource,
                group_multiplier:
                  value === 'fixed' ? (plan.group_multiplier ?? '') : '',
              })
            }
            disabled={disabled}
          >
            <SelectTrigger className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='inherit_group_ratio'>
                {t('systemSettings.fields.inheritGroupRatio')}
              </SelectItem>
              <SelectItem value='fixed'>
                {t('systemSettings.fields.fixedMultiplier')}
              </SelectItem>
            </SelectContent>
          </Select>
        </PriceField>
        {plan.group_multiplier_source === 'fixed' ? (
          <PriceField label={t('systemSettings.fields.groupMultiplier')}>
            <Input
              inputMode='decimal'
              value={plan.group_multiplier ?? ''}
              onChange={(event) =>
                update({ group_multiplier: event.target.value })
              }
              disabled={disabled}
            />
          </PriceField>
        ) : null}
      </div>

      {plan.billing_mode === 'token' ? (
        <TokenComponentEditor
          plan={plan}
          onChange={(nextPlan) => onChange(() => nextPlan)}
          disabled={disabled}
        />
      ) : null}

      {plan.billing_mode === 'per_request' ? (
        <div className='max-w-sm'>
          <PriceField label={t('systemSettings.fields.pricePerRequest')}>
            <Input
              inputMode='decimal'
              value={getComponent(plan, 'request')?.unit_price ?? ''}
              onChange={(event) =>
                onChange((current) =>
                  updateComponentPrice(current, 'request', event.target.value)
                )
              }
              disabled={disabled}
            />
          </PriceField>
        </div>
      ) : null}
    </div>
  )
}

function TokenComponentEditor({
  plan,
  onChange,
  disabled,
}: {
  plan: DraftPlan
  onChange: (plan: DraftPlan) => void
  disabled: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-baseline justify-between gap-2'>
        <h5 className='text-sm font-semibold'>
          {t('systemSettings.titles.componentPrices')}
        </h5>
        <span className='text-muted-foreground text-xs'>
          {t('systemSettings.fields.pricePerMillionTokens')}
        </span>
      </div>
      <div className='grid gap-4 xl:grid-cols-3'>
        {COMPONENT_GROUPS.map((group) => (
          <div key={group.titleKey} className='min-w-0 space-y-2'>
            <h6 className='text-muted-foreground text-xs font-medium'>
              {t(group.titleKey)}
            </h6>
            <div className='divide-y border'>
              {group.components.map((definition) => {
                const configured = getComponent(plan, definition.component)
                const inputID = `${plan.draftKey}-${definition.component}`
                return (
                  <div
                    key={definition.component}
                    className='flex min-w-0 items-center gap-2 px-2 py-2'
                  >
                    <Checkbox
                      id={inputID}
                      checked={configured !== undefined}
                      onCheckedChange={(checked) =>
                        onChange(
                          updateTokenComponent(
                            plan,
                            definition.component,
                            checked === true
                          )
                        )
                      }
                      disabled={disabled}
                    />
                    <Label
                      htmlFor={inputID}
                      className='min-w-0 flex-1 truncate text-xs font-normal'
                    >
                      {t(definition.labelKey)}
                    </Label>
                    {configured ? (
                      <Input
                        className='h-7 w-24 shrink-0 px-2 text-xs tabular-nums'
                        inputMode='decimal'
                        value={configured.unit_price}
                        onChange={(event) =>
                          onChange(
                            updateComponentPrice(
                              plan,
                              definition.component,
                              event.target.value
                            )
                          )
                        }
                        disabled={disabled}
                      />
                    ) : null}
                  </div>
                )
              })}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function LegacyPricePlans({ plans }: { plans: ModelPricePlan[] }) {
  const { t } = useTranslation()
  return (
    <details className='border' open={plans.length > 0}>
      <summary className='cursor-pointer px-3 py-2 text-sm font-medium'>
        {t('systemSettings.titles.legacyPriceFallback')}
      </summary>
      <div className='border-t'>
        {plans.length === 0 ? (
          <div className='text-muted-foreground px-3 py-4 text-sm'>
            {t('systemSettings.tips.noLegacyPricePlans')}
          </div>
        ) : (
          <div className='divide-y'>
            {plans.map((plan, index) => (
              <div
                key={`${plan.model_name}-${plan.context_min_tokens}-${index}`}
                className='grid gap-2 px-3 py-3 lg:grid-cols-[minmax(12rem,18rem)_minmax(0,1fr)]'
              >
                <div className='min-w-0'>
                  <div className='truncate text-sm font-medium'>
                    {plan.model_name}
                  </div>
                  <div className='text-muted-foreground text-xs'>
                    {formatPlanSummary(plan, t)}
                  </div>
                </div>
                <div className='flex flex-wrap gap-x-3 gap-y-1 text-xs'>
                  {(plan.components ?? []).map((component) => (
                    <span key={component.component} className='tabular-nums'>
                      {t(componentLabelKey(component.component))}:{' '}
                      {component.unit_price}{' '}
                      {component.unit === 'per_request'
                        ? t('systemSettings.fields.perRequestUnit')
                        : t('systemSettings.fields.perMillionTokenUnit')}
                    </span>
                  ))}
                  {plan.billing_mode === 'free' ? (
                    <span>{t('systemSettings.fields.freeBilling')}</span>
                  ) : null}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </details>
  )
}

function PriceField({
  label,
  children,
}: {
  label: string
  children: ReactNode
}) {
  return (
    <Label className='flex min-w-0 flex-col items-start gap-1.5'>
      <span>{label}</span>
      {children}
    </Label>
  )
}
