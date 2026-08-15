<script lang="ts">
  type Props = {
    value: string
    current: unknown
    setValue: (value: string) => void
  }

  let { value, current, setValue }: Props = $props()

  $effect(() => {
    if (value && String(current ?? '') !== value) {
      setValue(value)
    }
  })
</script>
