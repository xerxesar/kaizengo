import { KAppStatus, SpecLayout } from '@/k'

/** Compiled page. Regions, span, and size come from the GreetingBoard layout tree. */
export default function GreetingBoard() {
  return (
    <>
      <SpecLayout />
      <KAppStatus />
    </>
  )
}
