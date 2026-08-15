/** Catalog entry from GET /api/apps */
export type NavEntry = {
  id: string
  title: string
  route: string
  order?: number
}
