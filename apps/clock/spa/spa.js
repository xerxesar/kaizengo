/**
 * Analog clock SPA — uses platform calendars via GraphQL.
 */
const plugin = {
  id: 'clock',
  title: 'Clock',
  route: 'clock',
  _raf: 0,
  _timer: 0,
  _calendar: 'gregorian',

  async mount(el) {
    el.innerHTML = `
      <h1 id="clock-title">Clock</h1>
      <p class="clock-sub" id="clock-subtitle">Local time, ticking live.</p>
      <label class="cal-label">
        <span id="clock-cal-label">Calendar</span>
        <select id="clock-cal"></select>
      </label>
      <div class="clock-wrap">
        <svg class="clock-face" viewBox="0 0 200 200" aria-label="Analog clock">
          <circle cx="100" cy="100" r="96" fill="#fff" stroke="#111" stroke-width="4"/>
          <circle cx="100" cy="100" r="88" fill="none" stroke="#eee" stroke-width="1"/>
          ${ticks()}
          <line id="hour-hand" x1="100" y1="100" x2="100" y2="55" stroke="#111" stroke-width="5" stroke-linecap="round"/>
          <line id="minute-hand" x1="100" y1="100" x2="100" y2="35" stroke="#333" stroke-width="3.5" stroke-linecap="round"/>
          <line id="second-hand" x1="100" y1="110" x2="100" y2="28" stroke="#c0392b" stroke-width="1.5" stroke-linecap="round"/>
          <circle cx="100" cy="100" r="5" fill="#111"/>
          <circle cx="100" cy="100" r="2" fill="#c0392b"/>
        </svg>
        <p class="clock-digital" id="clock-digital">--:--:--</p>
      </div>
      <style>
        .clock-sub { color: #555; margin-top: -0.25rem; }
        .cal-label {
          display: flex; gap: 0.5rem; align-items: center; justify-content: center;
          margin: 0.75rem 0 0; font-size: 0.9rem;
        }
        .cal-label select {
          padding: 0.25rem 0.4rem; border: 1px solid #ccc; border-radius: 0.25rem;
        }
        .clock-wrap { display: grid; justify-items: center; gap: 0.75rem; margin-top: 1rem; }
        .clock-face { width: min(220px, 70vw); height: auto; }
        .clock-digital { font-variant-numeric: tabular-nums; font-size: 1.1rem; margin: 0; letter-spacing: 0.04em; }
      </style>
    `

    const hour = el.querySelector('#hour-hand')
    const minute = el.querySelector('#minute-hand')
    const second = el.querySelector('#second-hand')
    const digital = el.querySelector('#clock-digital')
    const select = el.querySelector('#clock-cal')

    try {
      const meta = await gql(`{
        clockCopy { title subtitle calendarLabel }
        clockCalendars { id name }
      }`)
      el.querySelector('#clock-title').textContent = meta.clockCopy.title
      el.querySelector('#clock-subtitle').textContent = meta.clockCopy.subtitle
      el.querySelector('#clock-cal-label').textContent = meta.clockCopy.calendarLabel
      for (const c of meta.clockCalendars) {
        const opt = document.createElement('option')
        opt.value = c.id
        opt.textContent = c.name
        select.appendChild(opt)
      }
      if (meta.clockCalendars.some((c) => c.id === 'persian')) {
        // both calendars available when persian driver is loaded
      }
      this._calendar = select.value || 'gregorian'
      try {
        const pref = await gql(`{ settings { defaultCalendar } }`)
        const def = pref.settings?.defaultCalendar
        if (def && [...select.options].some((o) => o.value === def)) {
          select.value = def
          this._calendar = def
        }
      } catch {
        // settings app may be disabled
      }
    } catch (e) {
      digital.textContent = e instanceof Error ? e.message : String(e)
    }

    select.addEventListener('change', () => {
      this._calendar = select.value
      void this._refreshDigital(digital)
    })

    const tickHands = () => {
      const now = new Date()
      const ms = now.getMilliseconds()
      const s = now.getSeconds() + ms / 1000
      const m = now.getMinutes() + s / 60
      const h = (now.getHours() % 12) + m / 60
      second.setAttribute('transform', `rotate(${s * 6} 100 100)`)
      minute.setAttribute('transform', `rotate(${m * 6} 100 100)`)
      hour.setAttribute('transform', `rotate(${h * 30} 100 100)`)
      this._raf = requestAnimationFrame(tickHands)
    }
    tickHands()

    const refresh = () => void this._refreshDigital(digital)
    refresh()
    this._timer = setInterval(refresh, 1000)
  },

  async _refreshDigital(digital) {
    try {
      const data = await gql(
        `query ($calendar: String) { clockNow(calendar: $calendar) }`,
        { calendar: this._calendar },
      )
      digital.textContent = data.clockNow
    } catch (e) {
      digital.textContent = e instanceof Error ? e.message : String(e)
    }
  },

  unmount(el) {
    cancelAnimationFrame(this._raf)
    clearInterval(this._timer)
    this._raf = 0
    this._timer = 0
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

function ticks() {
  let out = ''
  for (let i = 0; i < 12; i++) {
    const a = (i * 30 * Math.PI) / 180
    const x1 = 100 + Math.sin(a) * 78
    const y1 = 100 - Math.cos(a) * 78
    const x2 = 100 + Math.sin(a) * 88
    const y2 = 100 - Math.cos(a) * 88
    out += `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="#111" stroke-width="2.5"/>`
  }
  for (let i = 0; i < 60; i++) {
    if (i % 5 === 0) continue
    const a = (i * 6 * Math.PI) / 180
    const x1 = 100 + Math.sin(a) * 84
    const y1 = 100 - Math.cos(a) * 84
    const x2 = 100 + Math.sin(a) * 88
    const y2 = 100 - Math.cos(a) * 88
    out += `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="#999" stroke-width="1"/>`
  }
  return out
}

export default plugin
