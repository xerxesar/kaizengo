/**
 * Settings — configure platform (locale, calendar) and core shell title.
 */
const plugin = {
  async mount(el) {
    el.innerHTML = `<p class="muted">Loading settings…</p>`
    try {
      const data = await gql(`{
        settings {
          locale locales defaultCalendar shellTitle
          calendars { id name }
          labels { title locale calendar shell save saved }
        }
      }`)
      const s = data.settings
      el.innerHTML = `
        <h1>${escapeHtml(s.labels.title)}</h1>
        <form class="form" id="settings-form">
          <label>
            <span>${escapeHtml(s.labels.locale)}</span>
            <select name="locale">
              ${s.locales.map((id) => `<option value="${escapeHtml(id)}" ${id === s.locale ? 'selected' : ''}>${escapeHtml(id)}</option>`).join('')}
            </select>
          </label>
          <label>
            <span>${escapeHtml(s.labels.calendar)}</span>
            <select name="defaultCalendar">
              ${s.calendars.map((c) => `<option value="${escapeHtml(c.id)}" ${c.id === s.defaultCalendar ? 'selected' : ''}>${escapeHtml(c.name)}</option>`).join('')}
            </select>
          </label>
          <label>
            <span>${escapeHtml(s.labels.shell)}</span>
            <input name="shellTitle" type="text" value="${escapeHtml(s.shellTitle)}" />
          </label>
          <button type="submit">${escapeHtml(s.labels.save)}</button>
          <p class="status" id="settings-status" hidden>${escapeHtml(s.labels.saved)}</p>
        </form>
        <style>
          .form { display: grid; gap: 0.85rem; max-width: 22rem; margin: 1rem auto 0; text-align: left; }
          .form label { display: grid; gap: 0.3rem; font-size: 0.9rem; }
          .form select, .form input {
            padding: 0.45rem 0.55rem; border: 1px solid #ccc; border-radius: 0.25rem;
          }
          .form button {
            padding: 0.5rem 0.75rem; border: 1px solid #111; border-radius: 0.25rem;
            background: #111; color: #fff; cursor: pointer;
          }
          .status { color: #0a7; margin: 0; }
          .muted { color: #666; }
          .err { color: #b00020; }
        </style>
      `

      const form = el.querySelector('#settings-form')
      const status = el.querySelector('#settings-status')
      form.addEventListener('submit', async (e) => {
        e.preventDefault()
        status.hidden = true
        const fd = new FormData(form)
        try {
          await gql(
            `mutation ($locale: String, $defaultCalendar: String, $shellTitle: String) {
              updateSettings(locale: $locale, defaultCalendar: $defaultCalendar, shellTitle: $shellTitle) {
                locale defaultCalendar shellTitle
              }
            }`,
            {
              locale: fd.get('locale'),
              defaultCalendar: fd.get('defaultCalendar'),
              shellTitle: fd.get('shellTitle'),
            },
          )
          status.hidden = false
          // Notify shell to refresh brand title if listening.
          window.dispatchEvent(new CustomEvent('kaizengo:settings', {
            detail: { shellTitle: fd.get('shellTitle') },
          }))
        } catch (err) {
          status.hidden = false
          status.className = 'status err'
          status.textContent = err instanceof Error ? err.message : String(err)
        }
      })
    } catch (e) {
      el.innerHTML = `<p class="err">${escapeHtml(e instanceof Error ? e.message : String(e))}</p>`
    }
  },
  unmount(el) {
    el.innerHTML = ''
  },
}

async function gql(query, variables) {
  const res = await fetch('/graphql', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
  if (!res.ok) throw new Error(`GraphQL HTTP ${res.status}`)
  const body = await res.json()
  if (body.errors?.length) {
    throw new Error(body.errors.map((e) => e.message).join(', '))
  }
  return body.data
}

function escapeHtml(s) {
  return String(s)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

export default plugin
