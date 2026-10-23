declare module 'virtual:kaizengo-i18n' {
  export const catalogs: Record<string, Record<string, string>>
  export const localeDirs: Record<string, 'ltr' | 'rtl'>
}
