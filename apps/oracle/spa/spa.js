/**
 * Oracle SPA — vanilla JS + JSON API.
 */
const plugin = {
  async mount(el) {
    el.innerHTML = `
      <div class="oracle">
        <div class="orb" aria-hidden="true">✦</div>
        <h1>The Oracle</h1>
        <p>Full SPA. Answers come from <code>POST /api/oracle/ask</code>.</p>
        <form class="form">
          <label for="q">Your question</label>
          <input id="q" name="q" type="text" placeholder="Should I refactor everything today?" autocomplete="off"/>
          <button type="submit">Consult the void</button>
        </form>
        <div class="out idle">Ask a yes/no question. The oracle is… theatrical.</div>
      </div>
      <style>
        .oracle { text-align: center; }
        .orb {
          width: 5.5rem; height: 5.5rem; margin: 0 auto 1rem;
          border-radius: 50%;
          background: radial-gradient(circle at 35% 30%, #5b8def, #1a1a2e 70%);
          display: grid; place-items: center; color: #fff; font-size: 1.75rem;
        }
        .form { display: grid; gap: 0.6rem; text-align: left; margin-top: 1rem; }
        .form input { padding: 0.55rem 0.7rem; border: 1px solid #ccc; border-radius: 0.25rem; }
        .form button {
          padding: 0.45rem 0.75rem; border: 1px solid #111; border-radius: 0.25rem;
          background: #111; color: #fff; cursor: pointer;
        }
        .out {
          margin-top: 1rem; padding: 1rem; background: #f0f0f0;
          border-radius: 0.5rem; text-align: left; min-height: 4rem;
        }
        .out.idle { color: #666; }
        .out .q { font-style: italic; color: #444; margin: 0 0 0.4rem; }
        .out .a { margin: 0; font-weight: 600; font-size: 1.1rem; }
      </style>
    `

    const form = el.querySelector('form')
    const out = el.querySelector('.out')
    form.addEventListener('submit', async (e) => {
      e.preventDefault()
      const q = new FormData(form).get('q') || ''
      out.classList.remove('idle')
      out.textContent = 'Consulting…'
      try {
        const res = await fetch('/api/oracle/ask', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ q }),
        })
        const data = await res.json()
        out.innerHTML = `<p class="q">“${escapeHtml(data.q)}”</p><p class="a">${escapeHtml(data.answer)}</p>`
      } catch (err) {
        out.textContent = err instanceof Error ? err.message : String(err)
      }
    })
  },
  unmount(el) {
    el.innerHTML = ''
  },
}

function escapeHtml(s) {
  return String(s)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

export default plugin
