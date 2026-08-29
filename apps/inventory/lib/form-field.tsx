import { FormField, Select, t, type KFormFieldContext } from '@kaizengo/sdk-solid/ui'
import { ENUMS, enumKey } from './enums'
import { RelationPicker } from './RelationPicker'

export function enumOptions(model: string, fieldKey: string) {
  const values = ENUMS[enumKey(model, fieldKey)] ?? []
  return values.map((value) => {
    const key = `inventory.enum.${enumKey(model, fieldKey)}.${value}`
    const label = t(key)
    return { value, label: label === key ? value.replaceAll('_', ' ') : label }
  })
}

export function renderInventoryField(
  ctx: KFormFieldContext,
  model: string,
  fromApp: string,
) {
  const enums = enumOptions(model, ctx.field.key)
  if (ctx.field.relation) {
    return (
      <FormField label={ctx.label} required={ctx.field.required}>
        <RelationPicker
          relation={ctx.field.relation}
          fromApp={fromApp}
          value={String(ctx.draft[ctx.field.key] ?? '')}
          placeholder={ctx.placeholder}
          onChange={(id) => ctx.setValue(id)}
        />
      </FormField>
    )
  }
  if (enums.length) {
    return (
      <FormField label={ctx.label} required={ctx.field.required}>
        <Select
          value={String(ctx.draft[ctx.field.key] ?? '')}
          options={enums}
          onChange={(value) => ctx.setValue(value)}
        />
      </FormField>
    )
  }
  return ctx.default()
}
