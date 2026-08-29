/** Parse gettext .po bytes into msgid → msgstr (matches Go i18n.ParsePO). */

export function parsePO(source: string): Record<string, string> {
  const out: Record<string, string> = {}
  let msgid = ''
  let msgstr = ''
  let state: '' | 'msgid' | 'msgstr' = ''
  let fuzzy = false

  const flush = () => {
    if (msgid && msgstr && !fuzzy) {
      out[msgid] = msgstr
    }
    msgid = ''
    msgstr = ''
    state = ''
    fuzzy = false
  }

  for (const raw of source.split(/\r?\n/)) {
    const trim = raw.trim()
    if (!trim) {
      flush()
      continue
    }
    if (trim.startsWith('#,') && trim.includes('fuzzy')) {
      fuzzy = true
      continue
    }
    if (trim.startsWith('#')) continue
    if (trim.startsWith('msgctxt ')) {
      state = ''
      continue
    }
    if (trim.startsWith('msgid ')) {
      if (state === 'msgstr') flush()
      state = 'msgid'
      msgid = unquotePO(trim.slice('msgid '.length))
      continue
    }
    if (trim.startsWith('msgstr ')) {
      state = 'msgstr'
      msgstr = unquotePO(trim.slice('msgstr '.length))
      continue
    }
    if (trim.startsWith('"')) {
      const chunk = unquotePO(trim)
      if (state === 'msgid') msgid += chunk
      else if (state === 'msgstr') msgstr += chunk
    }
  }
  flush()
  delete out['']
  return out
}

function unquotePO(s: string): string {
  const t = s.trim()
  if (!t) return ''
  try {
    return JSON.parse(t) as string
  } catch {
    return t.replace(/^"|"$/g, '')
  }
}
