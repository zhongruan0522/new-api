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
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

const oauthSchema = z.object({
  GitHubOAuthEnabled: z.boolean(),
  GitHubClientId: z.string(),
  GitHubClientSecret: z.string(),
  LinuxDOOAuthEnabled: z.boolean(),
  LinuxDOClientId: z.string(),
  LinuxDOClientSecret: z.string(),
  LinuxDOMinimumTrustLevel: z.string(),
})

type OAuthFormValues = z.infer<typeof oauthSchema>

type OAuthSectionProps = {
  defaultValues: OAuthFormValues
}

const secretFields: ReadonlySet<keyof OAuthFormValues> = new Set([
  'GitHubClientSecret',
  'LinuxDOClientSecret',
])

export function OAuthSection({ defaultValues }: OAuthSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<OAuthFormValues>({
    resolver: zodResolver(oauthSchema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  const onSubmit = async (values: OAuthFormValues) => {
    const updates = (
      Object.entries(values) as [
        keyof OAuthFormValues,
        OAuthFormValues[keyof OAuthFormValues],
      ][]
    ).filter(([key, value]) => {
      if (secretFields.has(key)) {
        return String(value ?? '').trim() !== ''
      }
      return value !== defaultValues[key]
    })

    if (updates.length === 0) {
      toast.info(t('channels.fields.noChangesToSave'))
      return
    }

    // Sort: put "Enabled" toggle updates last so that ID/Secret are saved
    // to the backend before the toggle validation checks them.
    updates.sort((a, b) => {
      const aIsEnabled = a[0].endsWith('Enabled') ? 1 : 0
      const bIsEnabled = b[0].endsWith('Enabled') ? 1 : 0
      return aIsEnabled - bIsEnabled
    })

    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key, value })
    }
  }

  return (
    <SettingsSection title={t('systemSettings.fields.oauthIntegrations')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />

          <Tabs defaultValue='github' className='space-y-6'>
            <TabsList className='grid w-full grid-cols-2'>
              <TabsTrigger value='github'>
                {t('profile.fields.gitHub')}
              </TabsTrigger>
              <TabsTrigger value='linuxdo'>
                {t('systemSettings.fields.linuxDo')}
              </TabsTrigger>
            </TabsList>

            <TabsContent value='github' className='space-y-6'>
              <FormField
                control={form.control}
                name='GitHubOAuthEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>
                        {t('systemSettings.actions.enableGitHubOauth')}
                      </FormLabel>
                      <FormDescription>
                        {t(
                          'systemSettings.actions.allowUsersToSignInAndRegisterWithGitHub'
                        )}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />

              <FormField
                control={form.control}
                name='GitHubClientId'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('systemSettings.fields.gitHubClientId')}
                    </FormLabel>
                    <FormControl>
                      <Input autoComplete='off' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='GitHubClientSecret'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('systemSettings.fields.gitHubClientSecret')}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        autoComplete='new-password'
                        placeholder={t(
                          'systemSettings.tips.sensitiveValuesAreNotReturned'
                        )}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'systemSettings.tips.leaveBlankUnlessYouWantToUpdateTheSecret'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </TabsContent>

            <TabsContent value='linuxdo' className='space-y-6'>
              <FormField
                control={form.control}
                name='LinuxDOOAuthEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>
                        {t('systemSettings.actions.enableLinuxDoOauth')}
                      </FormLabel>
                      <FormDescription>
                        {t(
                          'systemSettings.actions.allowUsersToSignInAndRegisterWithLinuxDo'
                        )}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />

              <FormField
                control={form.control}
                name='LinuxDOClientId'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('systemSettings.fields.linuxDoClientId')}
                    </FormLabel>
                    <FormControl>
                      <Input autoComplete='off' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='LinuxDOClientSecret'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('systemSettings.fields.linuxDoClientSecret')}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        autoComplete='new-password'
                        placeholder={t(
                          'systemSettings.tips.sensitiveValuesAreNotReturned'
                        )}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'systemSettings.tips.leaveBlankUnlessYouWantToUpdateTheSecret'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='LinuxDOMinimumTrustLevel'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('systemSettings.fields.linuxDoMinimumTrustLevel')}
                    </FormLabel>
                    <FormControl>
                      <Input autoComplete='off' placeholder='0' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </TabsContent>
          </Tabs>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
