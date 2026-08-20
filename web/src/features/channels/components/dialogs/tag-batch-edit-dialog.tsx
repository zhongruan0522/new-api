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
import { useState, useEffect, useMemo } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, AlertCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { MultiSelect } from '@/components/multi-select'
import {
  getTagModels,
  editTagChannels,
  getAllModels,
  getGroups,
} from '../../api'
import { channelsQueryKeys } from '../../lib'
import type { TagOperationParams } from '../../types'
import { useChannels } from '../channels-provider'
import { ModelMappingEditor } from '../model-mapping-editor'

type TagBatchEditDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function TagBatchEditDialog({
  open,
  onOpenChange,
}: TagBatchEditDialogProps) {
  const { t } = useTranslation()
  const { currentTag } = useChannels()
  const queryClient = useQueryClient()
  const [isLoading, setIsLoading] = useState(false)
  const [isSaving, setIsSaving] = useState(false)

  // Form fields
  const [newTag, setNewTag] = useState('')
  const [models, setModels] = useState('')
  const [modelMapping, setModelMapping] = useState('')
  const [groups, setGroups] = useState<string[]>([])

  // Fetch available groups
  const { data: groupsData, isLoading: isLoadingGroups } = useQuery({
    queryKey: ['groups'],
    queryFn: getGroups,
  })

  // Transform groups to multi-select options
  const groupOptions = useMemo(() => {
    if (!groupsData?.data) return []
    const allGroups = new Set([...groupsData.data, ...groups])
    return Array.from(allGroups).map((group) => ({
      value: group,
      label: group,
    }))
  }, [groupsData, groups])

  useEffect(() => {
    if (open && currentTag) {
      loadTagData()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, currentTag])

  const loadTagData = async () => {
    if (!currentTag) return

    setIsLoading(true)
    try {
      // Fetch current tag models
      const tagModelsResponse = await getTagModels(currentTag)
      if (tagModelsResponse.success && tagModelsResponse.data) {
        setModels(tagModelsResponse.data)
      }

      // Fetch all available models (for future use if needed)
      const allModelsResponse = await getAllModels()
      if (allModelsResponse.success && allModelsResponse.data) {
        // Available models could be used for autocomplete in the future
      }

      // Initialize new tag with current tag name
      setNewTag(currentTag)
    } catch (_error: unknown) {
      toast.error(
        _error instanceof Error
          ? _error.message
          : t('channels.errors.failedToLoadTagData')
      )
    } finally {
      setIsLoading(false)
    }
  }

  const handleSave = async () => {
    if (!currentTag) return

    // Validate model mapping JSON if provided
    if (modelMapping.trim()) {
      try {
        JSON.parse(modelMapping)
      } catch (_error) {
        toast.error(t('channels.errors.modelMappingMustBeValidJson'))
        return
      }
    }

    setIsSaving(true)
    try {
      const params: Record<string, string | undefined> = {
        tag: currentTag,
      }

      if (newTag !== currentTag) {
        params.new_tag = newTag || undefined
      }

      if (models.trim()) {
        params.models = models
      }

      if (modelMapping.trim()) {
        params.model_mapping = modelMapping
      }

      if (groups.length > 0) {
        params.groups = groups.join(',')
      }

      // Check if there are any changes
      if (Object.keys(params).length === 1) {
        toast.warning(t('channels.fields.noChangesMade'))
        return
      }

      const response = await editTagChannels(
        params as unknown as TagOperationParams
      )
      if (response.success) {
        toast.success(t('channels.status.tagUpdatedSuccessfully'))
        queryClient.invalidateQueries({ queryKey: channelsQueryKeys.lists() })
        handleClose()
      } else {
        toast.error(response.message || t('channels.errors.failedToUpdateTag'))
      }
    } catch (error: unknown) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('channels.errors.failedToUpdateTag')
      )
    } finally {
      setIsSaving(false)
    }
  }

  const handleClose = () => {
    setNewTag('')
    setModels('')
    setModelMapping('')
    setGroups([])
    onOpenChange(false)
  }

  if (!currentTag) return null

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className='max-h-[90vh] max-w-2xl overflow-y-auto'>
        <DialogHeader>
          <DialogTitle>{t('channels.fields.batchEditByTag')}</DialogTitle>
          <DialogDescription>
            {t('channels.actions.editAllChannelsWithTag')}{' '}
            <strong>{currentTag}</strong>
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className='flex items-center justify-center py-12'>
            <Loader2 className='text-muted-foreground h-8 w-8 animate-spin' />
          </div>
        ) : (
          <>
            <div className='space-y-4 py-4'>
              <Alert>
                <AlertCircle className='h-4 w-4' />
                <AlertDescription>
                  {t(
                    'channels.tips.allEditsAreOverwriteOperationsLeaveFieldsEmptyTo'
                  )}
                </AlertDescription>
              </Alert>

              {/* Tag Name */}
              <div className='space-y-2'>
                <Label htmlFor='new-tag'>{t('channels.fields.tagName')}</Label>
                <Input
                  id='new-tag'
                  placeholder={t(
                    'channels.placeholders.enterNewTagNameLeaveEmptyToDisbandTag'
                  )}
                  value={newTag}
                  onChange={(e) => setNewTag(e.target.value)}
                  disabled={isSaving}
                />
                <p className='text-muted-foreground text-xs'>
                  {t('channels.fields.leaveEmptyToDisbandTheTag')}
                </p>
              </div>

              {/* Models */}
              <div className='space-y-2'>
                <Label htmlFor='models'>{t('channels.titles.models')}</Label>
                <Textarea
                  id='models'
                  placeholder={t(
                    'channels.tips.commaSeparatedModelNamesLeaveEmptyToKeepCurrent'
                  )}
                  value={models}
                  onChange={(e) => setModels(e.target.value)}
                  disabled={isSaving}
                  rows={3}
                />
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'channels.tips.currentModelsForTheLongestChannelInThisTag'
                  )}
                </p>
              </div>

              {/* Model Mapping */}
              <div className='space-y-2'>
                <Label htmlFor='model-mapping'>
                  {t('channels.fields.modelMapping')}
                </Label>
                <ModelMappingEditor
                  value={modelMapping}
                  onChange={setModelMapping}
                  disabled={isSaving}
                />
              </div>

              {/* Groups */}
              <div className='space-y-2'>
                <Label htmlFor='groups'>{t('channels.fields.groups')}</Label>
                {isLoadingGroups ? (
                  <Skeleton className='h-10 w-full' />
                ) : (
                  <MultiSelect
                    options={groupOptions}
                    selected={groups}
                    onChange={setGroups}
                    placeholder={t(
                      'channels.placeholders.selectGroupsLeaveEmptyToKeepCurrent'
                    )}
                  />
                )}
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'channels.tips.userGroupsThatCanAccessChannelsWithThisTag'
                  )}
                </p>
              </div>
            </div>

            <DialogFooter>
              <Button
                variant='outline'
                onClick={handleClose}
                disabled={isSaving}
              >
                {t('common.actions.cancel')}
              </Button>
              <Button onClick={handleSave} disabled={isSaving}>
                {isSaving && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
                {isSaving
                  ? t('channels.tips.saving')
                  : t('channels.actions.saveChanges')}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
