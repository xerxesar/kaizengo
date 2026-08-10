/**
 * Status — vanilla SPA module (static-ish HTML via mount).
 */
const plugin = {
  async mount(el) {
    el.innerHTML = `
      <h1>Status</h1>
      <p>Vanilla JS module mounted with <code>import()</code> + <code>mount(el)</code>.</p>
      <p>Loaded: <time datetime="${new Date().toISOString()}">${new Date().toLocaleString()}</time></p>
    `
  },
  unmount(el) {
    el.innerHTML = ''
  },
}

export default plugin
