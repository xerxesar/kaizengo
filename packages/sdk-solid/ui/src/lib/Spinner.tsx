import { Progress } from '@ark-ui/solid/progress'
import { type JSX } from 'solid-js'
import { progressRangeClass, progressTrackClass } from './ark/styles'
import { cn } from './cn'

type Props = {
  class?: string
}

export function Spinner(props: Props): JSX.Element {
  return (
    <div class={cn('flex items-center justify-center py-8', props.class)} role="status" aria-label="Loading">
      <Progress.Root value={null} class="w-48 max-w-full">
        <Progress.Track class={progressTrackClass}>
          <Progress.Range class={progressRangeClass} />
        </Progress.Track>
      </Progress.Root>
    </div>
  )
}
