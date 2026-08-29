import { type JSX, type ParentProps } from 'solid-js'
import type { ThemeId } from './theme'

type Props = ParentProps & {
  theme?: ThemeId
}

export function Page(props: Props): JSX.Element {
  return (
    <div class="kg-page kg-app flex min-h-screen w-full flex-col bg-[var(--kg-bg)]" data-kg-theme={props.theme}>
      {props.children}
    </div>
  )
}
