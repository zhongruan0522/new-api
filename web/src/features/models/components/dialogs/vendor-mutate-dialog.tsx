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
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { createVendor, updateVendor } from '../../api'
import { vendorsQueryKeys, modelsQueryKeys } from '../../lib'
import { vendorFormSchema, type Vendor, type VendorFormValues } from '../../types'

type VendorMutateDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentVendor?: Vendor | null
}

export function VendorMutateDialog({
  open,
  onOpenChange,
  currentVendor,
}: VendorMutateDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const isEdit = Boolean(currentVendor?.id)
  const [isSaving, setIsSaving] = useState(false)

  const form = useForm<VendorFormValues>({
    resolver: zodResolver(vendorFormSchema),
    defaultValues: {
      name: '',
      description: '',
      icon: '',
      data_retention_days: null,
      training_opt_out: null,
      status: 1,
    },
  })

  // Load vendor data for editing
  useEffect(() => {
    if (open && isEdit && currentVendor) {
      form.reset({
        id: currentVendor.id,
        name: currentVendor.name,
        description: currentVendor.description || '',
        icon: currentVendor.icon || '',
        data_retention_days: currentVendor.data_retention_days ?? null,
        training_opt_out: currentVendor.training_opt_out ?? null,
        status: currentVendor.status || 1,
      })
    } else if (open && !isEdit) {
      form.reset({
        name: '',
        description: '',
        icon: '',
        data_retention_days: null,
        training_opt_out: null,
        status: 1,
      })
    }
  }, [open, isEdit, currentVendor, form])

  const onSubmit = async (values: VendorFormValues) => {
    setIsSaving(true)
    try {
      const payload: Partial<Vendor> = {
        ...values,
        data_retention_days: values.data_retention_days ?? null,
      }
      const response = isEdit
        ? await updateVendor({ ...payload, id: currentVendor!.id })
        : await createVendor(payload)

      if (response.success) {
        toast.success(
          isEdit ? t('Vendor updated successfully') : t('Vendor created successfully')
        )
        queryClient.invalidateQueries({ queryKey: vendorsQueryKeys.lists() })
        queryClient.invalidateQueries({ queryKey: modelsQueryKeys.lists() })
        queryClient.invalidateQueries({ queryKey: ['pricing'] })
        onOpenChange(false)
      } else {
        toast.error(response.message || t('Operation failed'))
      }
    } catch (error: unknown) {
      toast.error((error as Error)?.message || t('Operation failed'))
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t('Edit Vendor') : t('Create Vendor')}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? t('Update vendor information for {{name}}', {
                  name: currentVendor?.name,
                })
              : t('Add a new vendor to the system')}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Vendor Name *')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('OpenAI, Anthropic, etc.')}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('The unique name for this vendor')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='description'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Description')}</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder={t('Describe this vendor...')}
                      rows={3}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='icon'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Icon')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('OpenAI, Anthropic, Google, etc.')}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('@lobehub/icons key name')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='data_retention_days'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Data retention')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        step={1}
                        placeholder={t('Provider-specific')}
                        value={field.value ?? ''}
                        onChange={(event) =>
                          field.onChange(
                            event.target.value === ''
                              ? null
                              : Number(event.target.value)
                          )
                        }
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Use 0 for zero retention; leave empty when unknown.')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='training_opt_out'
                render={({ field }) => (
                  <FormItem className='border-border/70 bg-muted/20 flex min-h-[5.75rem] items-start gap-3 rounded-md border p-3'>
                    <FormControl>
                      <Checkbox
                        checked={field.value === true}
                        onCheckedChange={(checked) =>
                          field.onChange(checked ? true : null)
                        }
                      />
                    </FormControl>
                    <div className='space-y-1'>
                      <FormLabel>{t('Training opt-out')}</FormLabel>
                      <FormDescription>
                        {t(
                          'Enable when requests are not used for upstream training by default.'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />
            </div>

            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={() => onOpenChange(false)}
                disabled={isSaving}
              >
                {t('Cancel')}
              </Button>
              <Button type='submit' disabled={isSaving}>
                {isSaving && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
                {isSaving ? t('Saving...') : isEdit ? t('Update') : t('Create')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
