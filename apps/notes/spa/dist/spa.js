var fr = Object.defineProperty;
var un = (e) => {
  throw TypeError(e);
};
var ur = (e, t, n) => t in e ? fr(e, t, { enumerable: !0, configurable: !0, writable: !0, value: n }) : e[t] = n;
var L = (e, t, n) => ur(e, typeof t != "symbol" ? t + "" : t, n), It = (e, t, n) => t.has(e) || un("Cannot " + n);
var s = (e, t, n) => (It(e, t, "read from private field"), n ? n.call(e) : t.get(e)), m = (e, t, n) => t.has(e) ? un("Cannot add the same private member more than once") : t instanceof WeakSet ? t.add(e) : t.set(e, n), g = (e, t, n, r) => (It(e, t, "write to private field"), r ? r.call(e, n) : t.set(e, n), n), b = (e, t, n) => (It(e, t, "access private method"), n);
var or = Array.isArray, ar = Array.prototype.indexOf, St = Array.prototype.includes, cr = Array.from, hr = Object.defineProperty, it = Object.getOwnPropertyDescriptor, dr = Object.prototype, vr = Array.prototype, _r = Object.getPrototypeOf, on = Object.isExtensible;
const pr = () => {
};
function gr(e) {
  for (var t = 0; t < e.length; t++)
    e[t]();
}
function Sn() {
  var e, t, n = new Promise((r, i) => {
    e = r, t = i;
  });
  return { promise: n, resolve: e, reject: t };
}
const N = 2, ut = 4, Ft = 8, Tn = 1 << 24, X = 16, re = 32, pe = 64, Ht = 128, $ = 512, R = 1024, C = 2048, te = 4096, K = 8192, ne = 16384, Je = 32768, an = 1 << 25, Ke = 65536, Tt = 1 << 17, wr = 1 << 18, Ze = 1 << 19, mr = 1 << 20, Ne = 65536, xt = 1 << 21, qe = 1 << 22, ye = 1 << 23, Lt = Symbol("$state"), yr = Symbol("attributes"), Er = Symbol("class"), br = Symbol("style"), Xe = Symbol("text"), _t = new class extends Error {
  constructor() {
    super(...arguments);
    L(this, "name", "StaleReactionError");
    L(this, "message", "The reaction that called `getAbortSignal()` was re-run or destroyed");
  }
}();
function kr(e) {
  throw new Error("https://svelte.dev/e/lifecycle_outside_component");
}
function Sr() {
  throw new Error("https://svelte.dev/e/async_derived_orphan");
}
function Tr(e) {
  throw new Error("https://svelte.dev/e/effect_in_teardown");
}
function xr() {
  throw new Error("https://svelte.dev/e/effect_in_unowned_derived");
}
function Ar(e) {
  throw new Error("https://svelte.dev/e/effect_orphan");
}
function Rr() {
  throw new Error("https://svelte.dev/e/effect_update_depth_exceeded");
}
function Cr() {
  throw new Error("https://svelte.dev/e/state_descriptors_fixed");
}
function Or() {
  throw new Error("https://svelte.dev/e/state_prototype_fixed");
}
function Nr() {
  throw new Error("https://svelte.dev/e/state_unsafe_mutation");
}
function Dr() {
  throw new Error("https://svelte.dev/e/svelte_boundary_reset_onerror");
}
const Pr = 2, A = Symbol("uninitialized");
function Fr() {
  console.warn("https://svelte.dev/e/derived_inert");
}
function Mr() {
  console.warn("https://svelte.dev/e/svelte_boundary_reset_noop");
}
function xn(e) {
  return e === this.v;
}
let F = null;
function We(e) {
  F = e;
}
function An(e, t = !1, n) {
  F = {
    p: F,
    i: !1,
    c: null,
    e: null,
    s: e,
    x: null,
    r: (
      /** @type {Effect} */
      E
    ),
    l: null
  };
}
function Rn(e) {
  var t = (
    /** @type {ComponentContext} */
    F
  ), n = t.e;
  if (n !== null) {
    t.e = null;
    for (var r of n)
      $n(r);
  }
  return t.i = !0, F = t.p, /** @type {T} */
  {};
}
function Cn() {
  return !0;
}
let Ie = [];
function jr() {
  var e = Ie;
  Ie = [], gr(e);
}
function Be(e) {
  if (Ie.length === 0) {
    var t = Ie;
    queueMicrotask(() => {
      t === Ie && jr();
    });
  }
  Ie.push(e);
}
function On(e) {
  var t = E;
  if (t === null)
    return y.f |= ye, e;
  if ((t.f & Je) === 0 && (t.f & ut) === 0)
    throw e;
  me(e, t);
}
function me(e, t) {
  if (!(t !== null && (t.f & ne) !== 0)) {
    for (; t !== null; ) {
      if ((t.f & Ht) !== 0) {
        if ((t.f & Je) === 0)
          throw e;
        try {
          t.b.error(e);
          return;
        } catch (n) {
          e = n;
        }
      }
      t = t.parent;
    }
    throw e;
  }
}
const Ir = -7169;
function T(e, t) {
  e.f = e.f & Ir | t;
}
function Xt(e) {
  (e.f & $) !== 0 || e.deps === null ? T(e, R) : T(e, te);
}
function Nn(e) {
  if (e !== null)
    for (const t of e)
      (t.f & N) === 0 || (t.f & Ne) === 0 || (t.f ^= Ne, Nn(
        /** @type {Derived} */
        t.deps
      ));
}
function Dn(e, t, n) {
  (e.f & C) !== 0 ? t.add(e) : (e.f & te) !== 0 && n.add(e), Nn(e.deps), T(e, R);
}
function Mt(e) {
  var t = y, n = E;
  W(null), oe(null);
  try {
    return e();
  } finally {
    W(t), oe(n);
  }
}
function Lr(e) {
  let t = 0, n = jt(0), r;
  return () => {
    sn() && (fe(n), ci(() => (t === 0 && (r = lr(() => e(() => lt(n)))), t += 1, () => {
      Be(() => {
        t -= 1, t === 0 && (r == null || r(), r = void 0, lt(n));
      });
    })));
  };
}
var qr = Ke | Ze;
function Br(e, t, n, r) {
  new Vr(e, t, n, r);
}
var H, Zt, z, Se, M, G, P, B, he, Te, ge, Ve, at, ct, de, Nt, x, Yr, Ur, Hr, zt, yt, Et, Gt, $t;
class Vr {
  /**
   * @param {TemplateNode} node
   * @param {BoundaryProps} props
   * @param {((anchor: Node) => void)} children
   * @param {((error: unknown) => unknown) | undefined} [transform_error]
   */
  constructor(t, n, r, i) {
    m(this, x);
    /** @type {Boundary | null} */
    L(this, "parent");
    L(this, "is_pending", !1);
    /**
     * API-level transformError transform function. Transforms errors before they reach the `failed` snippet.
     * Inherited from parent boundary, or defaults to identity.
     * @type {(error: unknown) => unknown}
     */
    L(this, "transform_error");
    /** @type {TemplateNode} */
    m(this, H);
    /** @type {TemplateNode | null} */
    m(this, Zt, null);
    /** @type {BoundaryProps} */
    m(this, z);
    /** @type {((anchor: Node) => void)} */
    m(this, Se);
    /** @type {Effect} */
    m(this, M);
    /** @type {Effect | null} */
    m(this, G, null);
    /** @type {Effect | null} */
    m(this, P, null);
    /** @type {Effect | null} */
    m(this, B, null);
    /** @type {DocumentFragment | null} */
    m(this, he, null);
    m(this, Te, 0);
    m(this, ge, 0);
    m(this, Ve, !1);
    /** @type {Set<Effect>} */
    m(this, at, /* @__PURE__ */ new Set());
    /** @type {Set<Effect>} */
    m(this, ct, /* @__PURE__ */ new Set());
    /**
     * A source containing the number of pending async deriveds/expressions.
     * Only created if `$effect.pending()` is used inside the boundary,
     * otherwise updating the source results in needless `Batch.ensure()`
     * calls followed by no-op flushes
     * @type {Source<number> | null}
     */
    m(this, de, null);
    m(this, Nt, Lr(() => (g(this, de, jt(s(this, Te))), () => {
      g(this, de, null);
    })));
    var l;
    g(this, H, t), g(this, z, n), g(this, Se, (o) => {
      var u = (
        /** @type {Effect} */
        E
      );
      u.b = this, u.f |= Ht, r(o);
    }), this.parent = /** @type {Effect} */
    E.b, this.transform_error = i ?? ((l = this.parent) == null ? void 0 : l.transform_error) ?? ((o) => o), g(this, M, Kn(() => {
      b(this, x, zt).call(this);
    }, qr));
  }
  /**
   * Defer an effect inside a pending boundary until the boundary resolves
   * @param {Effect} effect
   */
  defer_effect(t) {
    Dn(t, s(this, at), s(this, ct));
  }
  /**
   * Returns `false` if the effect exists inside a boundary whose pending snippet is shown
   * @returns {boolean}
   */
  is_rendered() {
    return !this.is_pending && (!this.parent || this.parent.is_rendered());
  }
  has_pending_snippet() {
    return !!s(this, z).pending;
  }
  /**
   * Update the source that powers `$effect.pending()` inside this boundary,
   * and controls when the current `pending` snippet (if any) is removed.
   * Do not call from inside the class
   * @param {1 | -1} d
   * @param {Batch} batch
   */
  update_pending_count(t, n) {
    b(this, x, Gt).call(this, t, n), g(this, Te, s(this, Te) + t), !(!s(this, de) || s(this, Ve)) && (g(this, Ve, !0), Be(() => {
      g(this, Ve, !1), s(this, de) && Ct(s(this, de), s(this, Te));
    }));
  }
  get_effect_pending() {
    return s(this, Nt).call(this), fe(
      /** @type {Source<number>} */
      s(this, de)
    );
  }
  /** @param {unknown} error */
  error(t) {
    if (!s(this, z).onerror && !s(this, z).failed)
      throw t;
    w != null && w.is_fork ? (s(this, G) && w.skip_effect(s(this, G)), s(this, P) && w.skip_effect(s(this, P)), s(this, B) && w.skip_effect(s(this, B)), w.oncommit(() => {
      b(this, x, $t).call(this, t);
    })) : b(this, x, $t).call(this, t);
  }
}
H = new WeakMap(), Zt = new WeakMap(), z = new WeakMap(), Se = new WeakMap(), M = new WeakMap(), G = new WeakMap(), P = new WeakMap(), B = new WeakMap(), he = new WeakMap(), Te = new WeakMap(), ge = new WeakMap(), Ve = new WeakMap(), at = new WeakMap(), ct = new WeakMap(), de = new WeakMap(), Nt = new WeakMap(), x = new WeakSet(), Yr = function() {
  try {
    g(this, G, ae(() => s(this, Se).call(this, s(this, H))));
  } catch (t) {
    this.error(t);
  }
}, /**
 * @param {unknown} error The deserialized error from the server's hydration comment
 */
Ur = function(t) {
  const n = s(this, z).failed;
  n && g(this, B, ae(() => {
    n(
      s(this, H),
      () => t,
      () => () => {
      }
    );
  }));
}, Hr = function() {
  const t = s(this, z).pending;
  t && (this.is_pending = !0, g(this, P, ae(() => t(s(this, H)))), Be(() => {
    var n = g(this, he, document.createDocumentFragment()), r = Ot();
    n.append(r), g(this, G, b(this, x, Et).call(this, () => ae(() => s(this, Se).call(this, r)))), s(this, ge) === 0 && (s(this, H).before(n), g(this, he, null), ft(
      /** @type {Effect} */
      s(this, P),
      () => {
        g(this, P, null);
      }
    ), b(this, x, yt).call(
      this,
      /** @type {Batch} */
      w
    ));
  }));
}, zt = function() {
  try {
    if (this.is_pending = this.has_pending_snippet(), g(this, ge, 0), g(this, Te, 0), g(this, G, ae(() => {
      s(this, Se).call(this, s(this, H));
    })), s(this, ge) > 0) {
      var t = g(this, he, document.createDocumentFragment());
      Xn(s(this, G), t);
      const n = (
        /** @type {(anchor: Node) => void} */
        s(this, z).pending
      );
      g(this, P, ae(() => n(s(this, H))));
    } else
      b(this, x, yt).call(
        this,
        /** @type {Batch} */
        w
      );
  } catch (n) {
    this.error(n);
  }
}, /**
 * @param {Batch} batch
 */
yt = function(t) {
  this.is_pending = !1, t.transfer_effects(s(this, at), s(this, ct));
}, /**
 * @template T
 * @param {() => T} fn
 */
Et = function(t) {
  var n = E, r = y, i = F;
  oe(s(this, M)), W(s(this, M)), We(s(this, M).ctx);
  try {
    return De.ensure(), t();
  } catch (l) {
    return On(l), null;
  } finally {
    oe(n), W(r), We(i);
  }
}, /**
 * Updates the pending count associated with the currently visible pending snippet,
 * if any, such that we can replace the snippet with content once work is done
 * @param {1 | -1} d
 * @param {Batch} batch
 */
Gt = function(t, n) {
  var r;
  if (!this.has_pending_snippet()) {
    this.parent && b(r = this.parent, x, Gt).call(r, t, n);
    return;
  }
  g(this, ge, s(this, ge) + t), s(this, ge) === 0 && (b(this, x, yt).call(this, n), s(this, P) && ft(s(this, P), () => {
    g(this, P, null);
  }), s(this, he) && (s(this, H).before(s(this, he)), g(this, he, null)));
}, /**
 * @param {unknown} error
 */
$t = function(t) {
  s(this, G) && (Y(s(this, G)), g(this, G, null)), s(this, P) && (Y(s(this, P)), g(this, P, null)), s(this, B) && (Y(s(this, B)), g(this, B, null));
  var n = s(this, z).onerror;
  let r = s(this, z).failed;
  var i = !1, l = !1;
  const o = () => {
    if (i) {
      Mr();
      return;
    }
    i = !0, l && Dr(), s(this, B) !== null && ft(s(this, B), () => {
      g(this, B, null);
    }), b(this, x, Et).call(this, () => {
      b(this, x, zt).call(this);
    });
  }, u = (f) => {
    try {
      l = !0, n == null || n(f, o), l = !1;
    } catch (a) {
      me(a, s(this, M) && s(this, M).parent);
    }
    r && g(this, B, b(this, x, Et).call(this, () => {
      try {
        return ae(() => {
          var a = (
            /** @type {Effect} */
            E
          );
          a.b = this, a.f |= Ht, r(
            s(this, H),
            () => f,
            () => o
          );
        });
      } catch (a) {
        return me(
          a,
          /** @type {Effect} */
          s(this, M).parent
        ), null;
      }
    }));
  };
  Be(() => {
    var f;
    try {
      f = this.transform_error(t);
    } catch (a) {
      me(a, s(this, M) && s(this, M).parent);
      return;
    }
    f !== null && typeof f == "object" && typeof /** @type {any} */
    f.then == "function" ? f.then(
      u,
      /** @param {unknown} e */
      (a) => me(a, s(this, M) && s(this, M).parent)
    ) : u(f);
  });
};
function zr(e, t, n, r) {
  const i = $r;
  var l = e.filter((h) => !h.settled), o = t.map(i);
  if (n.length === 0 && l.length === 0) {
    r(o);
    return;
  }
  var u = (
    /** @type {Effect} */
    E
  ), f = Gr(), a = l.length === 1 ? l[0].promise : l.length > 1 ? Promise.all(l.map((h) => h.promise)) : null;
  function d(h) {
    if ((u.f & ne) === 0) {
      f();
      try {
        r([...o, ...h]);
      } catch (v) {
        me(v, u);
      }
      At();
    }
  }
  var _ = Pn();
  if (n.length === 0) {
    a.then(() => d([])).finally(_);
    return;
  }
  function c() {
    Promise.all(n.map((h) => /* @__PURE__ */ Kr(h))).then(d).catch((h) => me(h, u)).finally(_);
  }
  a ? a.then(() => {
    f(), c(), At();
  }) : c();
}
function Gr() {
  var e = (
    /** @type {Effect} */
    E
  ), t = y, n = F, r = (
    /** @type {Batch} */
    w
  );
  return function(l = !0) {
    oe(e), W(t), We(n), l && (e.f & ne) === 0 && (r == null || r.activate(), r == null || r.apply());
  };
}
function At(e = !0) {
  oe(null), W(null), We(null), e && (w == null || w.deactivate());
}
function Pn() {
  var e = (
    /** @type {Effect} */
    E
  ), t = e.b, n = (
    /** @type {Batch} */
    w
  ), r = !!(t != null && t.is_rendered());
  return t == null || t.update_pending_count(1, n), n.increment(r, e), () => {
    t == null || t.update_pending_count(-1, n), n.decrement(r, e);
  };
}
// @__NO_SIDE_EFFECTS__
function $r(e) {
  var t = N | C;
  return E !== null && (E.f |= Ze), {
    ctx: F,
    deps: null,
    effects: null,
    equals: xn,
    f: t,
    fn: e,
    reactions: null,
    rv: 0,
    v: (
      /** @type {V} */
      A
    ),
    wv: 0,
    parent: E,
    ac: null
  };
}
const et = Symbol("obsolete");
// @__NO_SIDE_EFFECTS__
function Kr(e, t, n) {
  let r = (
    /** @type {Effect | null} */
    E
  );
  r === null && Sr();
  var i = (
    /** @type {Promise<V>} */
    /** @type {unknown} */
    void 0
  ), l = jt(
    /** @type {V} */
    A
  ), o = !y, u = /* @__PURE__ */ new Set();
  return ai(() => {
    var h, v;
    var f = (
      /** @type {Effect} */
      E
    ), a = Sn();
    i = a.promise;
    try {
      Promise.resolve(e()).then(a.resolve, (p) => {
        p !== _t && a.reject(p);
      }).finally(At);
    } catch (p) {
      a.reject(p), At();
    }
    var d = (
      /** @type {Batch} */
      w
    );
    if (o) {
      if ((f.f & Je) !== 0)
        var _ = Pn();
      if (
        // boundary can be null if the async derived is inside an $effect.root not connected to the component render tree
        (h = r.b) != null && h.is_rendered()
      )
        (v = d.async_deriveds.get(f)) == null || v.reject(et);
      else
        for (const p of u.values())
          p.reject(et);
      u.add(a), d.async_deriveds.set(f, a);
    }
    const c = (p, S = void 0) => {
      _ == null || _(), u.delete(a), S !== et && (d.activate(), S ? (l.f |= ye, Ct(l, S)) : ((l.f & ye) !== 0 && (l.f ^= ye), Ct(l, p)), d.deactivate());
    };
    a.promise.then(c, (p) => c(null, p || "unknown"));
  }), fi(() => {
    for (const f of u)
      f.reject(et);
  }), new Promise((f) => {
    function a(d) {
      function _() {
        d === i ? f(l) : a(i);
      }
      d.then(_, _);
    }
    a(i);
  });
}
function Wr(e) {
  var t = e.effects;
  if (t !== null) {
    e.effects = null;
    for (var n = 0; n < t.length; n += 1)
      Y(
        /** @type {Effect} */
        t[n]
      );
  }
}
function en(e) {
  var t, n = E, r = e.parent;
  if (!Ee && r !== null && e.v !== A && // if it was never evaluated before, it's guaranteed to fail downstream, so we try to execute instead
  (r.f & (ne | K)) !== 0)
    return Fr(), e.v;
  oe(r);
  try {
    e.f &= ~Ne, Wr(e), t = rr(e);
  } finally {
    oe(n);
  }
  return t;
}
function Fn(e) {
  var t = en(e);
  if (!e.equals(t) && (e.wv = tr(), (!(w != null && w.is_fork) || e.deps === null) && (w !== null ? (w.capture(e, t, !0), st == null || st.capture(e, t, !0)) : e.v = t, e.deps === null))) {
    T(e, R);
    return;
  }
  Ee || (D !== null ? (sn() || w != null && w.is_fork) && D.set(e, t) : Xt(e));
}
function Qr(e) {
  var t;
  if (e.effects !== null)
    for (const n of e.effects)
      (n.teardown || n.ac) && ((t = n.teardown) == null || t.call(n), n.ac !== null && Mt(() => {
        n.ac.abort(_t), n.ac = null;
      }), n.fn !== null && (n.teardown = pr), ot(n, 0), ln(n));
}
function Mn(e) {
  if (e.effects !== null)
    for (const t of e.effects)
      t.teardown && t.fn !== null && Qe(t);
}
let qt = null, Me = null, w = null, st = null, D = null, Kt = null, Bt = !1, Le = null, bt = null;
var cn = 0;
let Jr = 1;
var Ye, we, xe, Ue, He, ze, ve, Ge, j, ht, _e, J, se, $e, Ae, k, Wt, tt, Qt, jn, In, je, Zr, nt;
const Dt = class Dt {
  constructor() {
    m(this, k);
    L(this, "id", Jr++);
    /** True as soon as `#process` was called */
    m(this, Ye, !1);
    L(this, "linked", !0);
    /** @type {Batch | null} */
    m(this, we, null);
    /** @type {Batch | null} */
    m(this, xe, null);
    /** @type {Map<Effect, ReturnType<typeof deferred<any>>>} */
    L(this, "async_deriveds", /* @__PURE__ */ new Map());
    /**
     * The current values of any signals that are updated in this batch.
     * Tuple format: [value, is_derived] (note: is_derived is false for deriveds, too, if they were overridden via assignment)
     * They keys of this map are identical to `this.#previous`
     * @type {Map<Value, [any, boolean]>}
     */
    L(this, "current", /* @__PURE__ */ new Map());
    /**
     * The values of any signals (sources and deriveds) that are updated in this batch _before_ those updates took place.
     * They keys of this map are identical to `this.#current`
     * @type {Map<Value, any>}
     */
    L(this, "previous", /* @__PURE__ */ new Map());
    /**
     * When the batch is committed (and the DOM is updated), we need to remove old branches
     * and append new ones by calling the functions added inside (if/each/key/etc) blocks
     * @type {Set<(batch: Batch) => void>}
     */
    m(this, Ue, /* @__PURE__ */ new Set());
    /**
     * If a fork is discarded, we need to destroy any effects that are no longer needed
     * @type {Set<(batch: Batch) => void>}
     */
    m(this, He, /* @__PURE__ */ new Set());
    /**
     * The number of async effects that are currently in flight
     */
    m(this, ze, 0);
    /**
     * Async effects that are currently in flight, _not_ inside a pending boundary
     * @type {Map<Effect, number>}
     */
    m(this, ve, /* @__PURE__ */ new Map());
    /**
     * A deferred that resolves when the batch is committed, used with `settled()`
     * TODO replace with Promise.withResolvers once supported widely enough
     * @type {{ promise: Promise<void>, resolve: (value?: any) => void, reject: (reason: unknown) => void } | null}
     */
    m(this, Ge, null);
    /**
     * The root effects that need to be flushed
     * @type {Effect[]}
     */
    m(this, j, []);
    /**
     * Effects created while this batch was active.
     * @type {Effect[]}
     */
    m(this, ht, []);
    /**
     * Deferred effects (which run after async work has completed) that are DIRTY
     * @type {Set<Effect>}
     */
    m(this, _e, /* @__PURE__ */ new Set());
    /**
     * Deferred effects that are MAYBE_DIRTY
     * @type {Set<Effect>}
     */
    m(this, J, /* @__PURE__ */ new Set());
    /**
     * A map of branches that still exist, but will be destroyed when this batch
     * is committed — we skip over these during `process`.
     * The value contains child effects that were dirty/maybe_dirty before being reset,
     * so they can be rescheduled if the branch survives.
     * @type {Map<Effect, { d: Effect[], m: Effect[] }>}
     */
    m(this, se, /* @__PURE__ */ new Map());
    /**
     * Inverse of #skipped_branches which we need to tell prior batches to unskip them when committing
     * @type {Set<Effect>}
     */
    m(this, $e, /* @__PURE__ */ new Set());
    L(this, "is_fork", !1);
    m(this, Ae, !1);
    Me === null ? qt = Me = this : (g(Me, xe, this), g(this, we, Me)), Me = this;
  }
  /**
   * Add an effect to the #skipped_branches map and reset its children
   * @param {Effect} effect
   */
  skip_effect(t) {
    s(this, se).has(t) || s(this, se).set(t, { d: [], m: [] }), s(this, $e).delete(t);
  }
  /**
   * Remove an effect from the #skipped_branches map and reschedule
   * any tracked dirty/maybe_dirty child effects
   * @param {Effect} effect
   * @param {(e: Effect) => void} callback
   */
  unskip_effect(t, n = (r) => this.schedule(r)) {
    var r = s(this, se).get(t);
    if (r) {
      s(this, se).delete(t);
      for (var i of r.d)
        T(i, C), n(i);
      for (i of r.m)
        T(i, te), n(i);
    }
    s(this, $e).add(t);
  }
  /**
   * Associate a change to a given source with the current
   * batch, noting its previous and current values
   * @param {Value} source
   * @param {any} value
   * @param {boolean} [is_derived]
   */
  capture(t, n, r = !1) {
    t.v !== A && !this.previous.has(t) && this.previous.set(t, t.v), (t.f & ye) === 0 && (this.current.set(t, [n, r]), D == null || D.set(t, n)), this.is_fork || (t.v = n);
  }
  activate() {
    w = this;
  }
  deactivate() {
    w = null, D = null;
  }
  flush() {
    try {
      Bt = !0, w = this, b(this, k, tt).call(this);
    } finally {
      cn = 0, Kt = null, Le = null, bt = null, Bt = !1, w = null, D = null, Ce.clear();
    }
  }
  discard() {
    var t;
    for (const n of s(this, He)) n(this);
    s(this, He).clear();
    for (const n of this.async_deriveds.values())
      n.reject(et);
    b(this, k, nt).call(this), (t = s(this, Ge)) == null || t.resolve();
  }
  /**
   * @param {Effect} effect
   */
  register_created_effect(t) {
    s(this, ht).push(t);
  }
  /**
   * @param {boolean} blocking
   * @param {Effect} effect
   */
  increment(t, n) {
    if (g(this, ze, s(this, ze) + 1), t) {
      let r = s(this, ve).get(n) ?? 0;
      s(this, ve).set(n, r + 1);
    }
  }
  /**
   * @param {boolean} blocking
   * @param {Effect} effect
   */
  decrement(t, n) {
    if (g(this, ze, s(this, ze) - 1), t) {
      let r = s(this, ve).get(n) ?? 0;
      r === 1 ? s(this, ve).delete(n) : s(this, ve).set(n, r - 1);
    }
    s(this, Ae) || (g(this, Ae, !0), Be(() => {
      g(this, Ae, !1), this.linked && this.flush();
    }));
  }
  /**
   * @param {Set<Effect>} dirty_effects
   * @param {Set<Effect>} maybe_dirty_effects
   */
  transfer_effects(t, n) {
    for (const r of t)
      s(this, _e).add(r);
    for (const r of n)
      s(this, J).add(r);
    t.clear(), n.clear();
  }
  /** @param {(batch: Batch) => void} fn */
  oncommit(t) {
    s(this, Ue).add(t);
  }
  /** @param {(batch: Batch) => void} fn */
  ondiscard(t) {
    s(this, He).add(t);
  }
  settled() {
    return (s(this, Ge) ?? g(this, Ge, Sn())).promise;
  }
  static ensure() {
    if (w === null) {
      const t = w = new Dt();
      Bt || Be(() => {
        s(t, Ye) || t.flush();
      });
    }
    return w;
  }
  apply() {
    {
      D = null;
      return;
    }
  }
  /**
   *
   * @param {Effect} effect
   */
  schedule(t) {
    var i;
    if (Kt = t, (i = t.b) != null && i.is_pending && (t.f & (ut | Ft | Tn)) !== 0 && (t.f & Je) === 0) {
      t.b.defer_effect(t);
      return;
    }
    for (var n = t; n.parent !== null; ) {
      n = n.parent;
      var r = n.f;
      if (Le !== null && n === E && (y === null || (y.f & N) === 0))
        return;
      if ((r & (pe | re)) !== 0) {
        if ((r & R) === 0)
          return;
        n.f ^= R;
      }
    }
    s(this, j).push(n);
  }
};
Ye = new WeakMap(), we = new WeakMap(), xe = new WeakMap(), Ue = new WeakMap(), He = new WeakMap(), ze = new WeakMap(), ve = new WeakMap(), Ge = new WeakMap(), j = new WeakMap(), ht = new WeakMap(), _e = new WeakMap(), J = new WeakMap(), se = new WeakMap(), $e = new WeakMap(), Ae = new WeakMap(), k = new WeakSet(), Wt = function() {
  if (this.is_fork) return !0;
  for (const r of s(this, ve).keys()) {
    for (var t = r, n = !1; t.parent !== null; ) {
      if (s(this, se).has(t)) {
        n = !0;
        break;
      }
      t = t.parent;
    }
    if (!n)
      return !0;
  }
  return !1;
}, tt = function() {
  var f, a, d, _;
  g(this, Ye, !0), cn++ > 1e3 && (b(this, k, nt).call(this), Xr());
  for (const c of s(this, _e))
    s(this, J).delete(c), T(c, C), this.schedule(c);
  for (const c of s(this, J))
    T(c, te), this.schedule(c);
  const t = s(this, j);
  g(this, j, []), this.apply();
  var n = Le = [], r = [], i = bt = [];
  for (const c of t)
    try {
      b(this, k, Qt).call(this, c, n, r);
    } catch (h) {
      throw Bn(c), b(this, k, Wt).call(this) || this.discard(), h;
    }
  if (w = null, i.length > 0) {
    var l = Dt.ensure();
    for (const c of i)
      l.schedule(c);
  }
  if (Le = null, bt = null, b(this, k, Wt).call(this)) {
    b(this, k, je).call(this, r), b(this, k, je).call(this, n);
    for (const [c, h] of s(this, se))
      qn(c, h);
    i.length > 0 && /** @type {unknown} */
    b(f = w, k, tt).call(f);
    return;
  }
  const o = b(this, k, jn).call(this);
  if (o) {
    b(this, k, je).call(this, r), b(this, k, je).call(this, n), b(a = o, k, In).call(a, this);
    return;
  }
  s(this, _e).clear(), s(this, J).clear();
  for (const c of s(this, Ue)) c(this);
  s(this, Ue).clear(), st = this, hn(r), hn(n), st = null, (d = s(this, Ge)) == null || d.resolve();
  var u = (
    /** @type {Batch | null} */
    /** @type {unknown} */
    w
  );
  if (s(this, ze) === 0 && (s(this, j).length === 0 || u !== null) && b(this, k, nt).call(this), s(this, j).length > 0)
    if (u !== null) {
      const c = u;
      s(c, j).push(...s(this, j).filter((h) => !s(c, j).includes(h)));
    } else
      u = this;
  u !== null && b(_ = u, k, tt).call(_);
}, /**
 * Traverse the effect tree, executing effects or stashing
 * them for later execution as appropriate
 * @param {Effect} root
 * @param {Effect[]} effects
 * @param {Effect[]} render_effects
 */
Qt = function(t, n, r) {
  t.f ^= R;
  for (var i = t.first; i !== null; ) {
    var l = i.f, o = (l & (re | pe)) !== 0, u = o && (l & R) !== 0, f = u || (l & K) !== 0 || s(this, se).has(i);
    if (!f && i.fn !== null) {
      o ? i.f ^= R : (l & ut) !== 0 ? n.push(i) : pt(i) && ((l & X) !== 0 && s(this, J).add(i), Qe(i));
      var a = i.first;
      if (a !== null) {
        i = a;
        continue;
      }
    }
    for (; i !== null; ) {
      var d = i.next;
      if (d !== null) {
        i = d;
        break;
      }
      i = i.parent;
    }
  }
}, jn = function() {
  for (var t = s(this, we); t !== null; ) {
    if (!t.is_fork) {
      for (const [n, [, r]] of this.current)
        if (t.current.has(n) && !r)
          return t;
    }
    t = s(t, we);
  }
  return null;
}, /**
 * @param {Batch} batch
 */
In = function(t) {
  var r;
  for (const [i, l] of t.current)
    !this.previous.has(i) && t.previous.has(i) && this.previous.set(i, t.previous.get(i)), this.current.set(i, l);
  for (const [i, l] of t.async_deriveds) {
    const o = this.async_deriveds.get(i);
    o && l.promise.then(o.resolve).catch(o.reject);
  }
  t.async_deriveds.clear(), this.transfer_effects(s(t, _e), s(t, J));
  const n = (i) => {
    var l = i.reactions;
    if (l !== null && !((i.f & N) !== 0 && (i.f & (C | te)) === 0))
      for (const f of l) {
        var o = f.f;
        if ((o & N) !== 0)
          n(
            /** @type {Derived} */
            f
          );
        else {
          var u = (
            /** @type {Effect} */
            f
          );
          o & (qe | X) && !this.async_deriveds.has(u) && (s(this, J).delete(u), T(u, C), this.schedule(u));
        }
      }
  };
  for (const i of this.current.keys())
    n(i);
  this.oncommit(() => t.discard()), b(r = t, k, nt).call(r), w = this, b(this, k, tt).call(this);
}, /**
 * @param {Effect[]} effects
 */
je = function(t) {
  for (var n = 0; n < t.length; n += 1)
    Dn(t[n], s(this, _e), s(this, J));
}, Zr = function() {
  var _;
  for (let c = qt; c !== null; c = s(c, xe)) {
    var t = c.id < this.id, n = [];
    for (const [h, [v, p]] of this.current) {
      if (c.current.has(h)) {
        var r = (
          /** @type {[any, boolean]} */
          c.current.get(h)[0]
        );
        if (t && v !== r)
          c.current.set(h, [v, p]);
        else
          continue;
      }
      n.push(h);
    }
    if (t)
      for (const [h, v] of this.async_deriveds) {
        const p = c.async_deriveds.get(h);
        p && v.promise.then(p.resolve).catch(p.reject);
      }
    var i = [...c.current.keys()].filter(
      (h) => !/** @type {[any, boolean]} */
      c.current.get(h)[1]
    );
    if (!(!s(c, Ye) || i.length === 0)) {
      var l = i.filter((h) => !this.current.has(h));
      if (l.length === 0)
        t && c.discard();
      else if (n.length > 0) {
        if (t)
          for (const h of s(this, $e))
            c.unskip_effect(h, (v) => {
              var p;
              (v.f & (X | qe)) !== 0 ? c.schedule(v) : b(p = c, k, je).call(p, [v]);
            });
        c.activate();
        var o = /* @__PURE__ */ new Set(), u = /* @__PURE__ */ new Map();
        for (var f of n)
          Ln(f, l, o, u);
        u = /* @__PURE__ */ new Map();
        var a = [...c.current].filter(([h, v]) => {
          const p = this.current.get(h);
          return p ? p[0] !== v[0] || p[1] !== v[1] : !0;
        }).map(([h]) => h);
        if (a.length > 0)
          for (const h of s(this, ht))
            (h.f & (ne | K | Tt)) === 0 && tn(h, a, u) && ((h.f & (qe | X)) !== 0 ? (T(h, C), c.schedule(h)) : s(c, _e).add(h));
        if (s(c, j).length > 0 && !s(c, Ae)) {
          c.apply();
          for (var d of s(c, j))
            b(_ = c, k, Qt).call(_, d, [], []);
          g(c, j, []);
        }
        c.deactivate();
      }
    }
  }
}, nt = function() {
  if (this.linked) {
    var t = s(this, we), n = s(this, xe);
    t === null ? qt = n : g(t, xe, n), n === null ? Me = t : g(n, we, t), this.linked = !1;
  }
};
let De = Dt;
function Xr() {
  try {
    Rr();
  } catch (e) {
    me(e, Kt);
  }
}
let Q = null;
function hn(e) {
  var t = e.length;
  if (t !== 0) {
    for (var n = 0; n < t; ) {
      var r = e[n++];
      if ((r.f & (ne | K)) === 0 && pt(r) && (Q = /* @__PURE__ */ new Set(), Qe(r), r.deps === null && r.first === null && r.nodes === null && r.teardown === null && r.ac === null && Qn(r), (Q == null ? void 0 : Q.size) > 0)) {
        Ce.clear();
        for (const i of Q) {
          if ((i.f & (ne | K)) !== 0) continue;
          const l = [i];
          let o = i.parent;
          for (; o !== null; )
            Q.has(o) && (Q.delete(o), l.push(o)), o = o.parent;
          for (let u = l.length - 1; u >= 0; u--) {
            const f = l[u];
            (f.f & (ne | K)) === 0 && Qe(f);
          }
        }
        Q.clear();
      }
    }
    Q = null;
  }
}
function Ln(e, t, n, r) {
  if (!n.has(e) && (n.add(e), e.reactions !== null))
    for (const i of e.reactions) {
      const l = i.f;
      (l & N) !== 0 ? Ln(
        /** @type {Derived} */
        i,
        t,
        n,
        r
      ) : (l & (qe | X)) !== 0 && (l & C) === 0 && tn(i, t, r) && (T(i, C), nn(
        /** @type {Effect} */
        i
      ));
    }
}
function tn(e, t, n) {
  const r = n.get(e);
  if (r !== void 0) return r;
  if (e.deps !== null)
    for (const i of e.deps) {
      if (St.call(t, i))
        return !0;
      if ((i.f & N) !== 0 && tn(
        /** @type {Derived} */
        i,
        t,
        n
      ))
        return n.set(
          /** @type {Derived} */
          i,
          !0
        ), !0;
    }
  return n.set(e, !1), !1;
}
function nn(e) {
  w.schedule(e);
}
function qn(e, t) {
  if (!((e.f & re) !== 0 && (e.f & R) !== 0)) {
    (e.f & C) !== 0 ? t.d.push(e) : (e.f & te) !== 0 && t.m.push(e), T(e, R);
    for (var n = e.first; n !== null; )
      qn(n, t), n = n.next;
  }
}
function Bn(e) {
  T(e, R);
  for (var t = e.first; t !== null; )
    Bn(t), t = t.next;
}
let Rt = /* @__PURE__ */ new Set();
const Ce = /* @__PURE__ */ new Map();
let Vn = !1;
function jt(e, t) {
  var n = {
    f: 0,
    // TODO ideally we could skip this altogether, but it causes type errors
    v: e,
    reactions: null,
    equals: xn,
    rv: 0,
    wv: 0
  };
  return n;
}
// @__NO_SIDE_EFFECTS__
function ie(e, t) {
  const n = jt(e);
  return vi(n), n;
}
function ce(e, t, n = !1) {
  y !== null && // since we are untracking the function inside `$inspect.with` we need to add this check
  // to ensure we error if state is set inside an inspect effect
  (!ee || (y.f & Tt) !== 0) && Cn() && (y.f & (N | X | qe | Tt)) !== 0 && (ue === null || !ue.has(e)) && Nr();
  let r = n ? rt(t) : t;
  return Ct(e, r, bt);
}
function Ct(e, t, n = null) {
  if (!e.equals(t)) {
    Ce.set(e, Ee ? t : e.v);
    var r = De.ensure();
    if (r.capture(e, t), (e.f & N) !== 0) {
      const i = (
        /** @type {Derived} */
        e
      );
      (e.f & C) !== 0 && en(i), D === null && Xt(i);
    }
    e.wv = tr(), Yn(e, C, n), E !== null && (E.f & R) !== 0 && (E.f & (re | pe)) === 0 && (U === null ? _i([e]) : U.push(e)), !r.is_fork && Rt.size > 0 && !Vn && ei();
  }
  return t;
}
function ei() {
  Vn = !1;
  for (const e of Rt) {
    (e.f & R) !== 0 && T(e, te);
    let t;
    try {
      t = pt(e);
    } catch {
      t = !0;
    }
    t && Qe(e);
  }
  Rt.clear();
}
function lt(e) {
  ce(e, e.v + 1);
}
function Yn(e, t, n) {
  var r = e.reactions;
  if (r !== null)
    for (var i = r.length, l = 0; l < i; l++) {
      var o = r[l], u = o.f, f = (u & C) === 0;
      if (f && T(o, t), (u & Tt) !== 0)
        Rt.add(
          /** @type {Effect} */
          o
        );
      else if ((u & N) !== 0) {
        var a = (
          /** @type {Derived} */
          o
        );
        D == null || D.delete(a), (u & Ne) === 0 && (u & $ && (E === null || (E.f & xt) === 0) && (o.f |= Ne), Yn(a, te, n));
      } else if (f) {
        var d = (
          /** @type {Effect} */
          o
        );
        (u & X) !== 0 && Q !== null && Q.add(d), n !== null ? n.push(d) : nn(d);
      }
    }
}
function rt(e) {
  if (typeof e != "object" || e === null || Lt in e)
    return e;
  const t = _r(e);
  if (t !== dr && t !== vr)
    return e;
  var n = /* @__PURE__ */ new Map(), r = or(e), i = /* @__PURE__ */ ie(0), l = Oe, o = (u) => {
    if (Oe === l)
      return u();
    var f = y, a = Oe;
    W(null), gn(l);
    var d = u();
    return W(f), gn(a), d;
  };
  return r && n.set("length", /* @__PURE__ */ ie(
    /** @type {any[]} */
    e.length
  )), new Proxy(
    /** @type {any} */
    e,
    {
      defineProperty(u, f, a) {
        (!("value" in a) || a.configurable === !1 || a.enumerable === !1 || a.writable === !1) && Cr();
        var d = n.get(f);
        return d === void 0 ? o(() => {
          var _ = /* @__PURE__ */ ie(a.value);
          return n.set(f, _), _;
        }) : ce(d, a.value, !0), !0;
      },
      deleteProperty(u, f) {
        var a = n.get(f);
        if (a === void 0) {
          if (f in u) {
            const d = o(() => /* @__PURE__ */ ie(A));
            n.set(f, d), lt(i);
          }
        } else
          ce(a, A), lt(i);
        return !0;
      },
      get(u, f, a) {
        var h;
        if (f === Lt)
          return e;
        var d = n.get(f), _ = f in u;
        if (d === void 0 && (!_ || (h = it(u, f)) != null && h.writable) && (d = o(() => {
          var v = rt(_ ? u[f] : A), p = /* @__PURE__ */ ie(v);
          return p;
        }), n.set(f, d)), d !== void 0) {
          var c = fe(d);
          return c === A ? void 0 : c;
        }
        return Reflect.get(u, f, a);
      },
      getOwnPropertyDescriptor(u, f) {
        var a = Reflect.getOwnPropertyDescriptor(u, f);
        if (a && "value" in a) {
          var d = n.get(f);
          d && (a.value = fe(d));
        } else if (a === void 0) {
          var _ = n.get(f), c = _ == null ? void 0 : _.v;
          if (_ !== void 0 && c !== A)
            return {
              enumerable: !0,
              configurable: !0,
              value: c,
              writable: !0
            };
        }
        return a;
      },
      has(u, f) {
        var c;
        if (f === Lt)
          return !0;
        var a = n.get(f), d = a !== void 0 && a.v !== A || Reflect.has(u, f);
        if (a !== void 0 || E !== null && (!d || (c = it(u, f)) != null && c.writable)) {
          a === void 0 && (a = o(() => {
            var h = d ? rt(u[f]) : A, v = /* @__PURE__ */ ie(h);
            return v;
          }), n.set(f, a));
          var _ = fe(a);
          if (_ === A)
            return !1;
        }
        return d;
      },
      set(u, f, a, d) {
        var Fe;
        var _ = n.get(f), c = f in u;
        if (r && f === "length")
          for (var h = a; h < /** @type {Source<number>} */
          _.v; h += 1) {
            var v = n.get(h + "");
            v !== void 0 ? ce(v, A) : h in u && (v = o(() => /* @__PURE__ */ ie(A)), n.set(h + "", v));
          }
        if (_ === void 0)
          (!c || (Fe = it(u, f)) != null && Fe.writable) && (_ = o(() => /* @__PURE__ */ ie(void 0)), ce(_, rt(a)), n.set(f, _));
        else {
          c = _.v !== A;
          var p = o(() => rt(a));
          ce(_, p);
        }
        var S = Reflect.getOwnPropertyDescriptor(u, f);
        if (S != null && S.set && S.set.call(d, a), !c) {
          if (r && typeof f == "string") {
            var O = (
              /** @type {Source<number>} */
              n.get("length")
            ), Pe = Number(f);
            Number.isInteger(Pe) && Pe >= O.v && ce(O, Pe + 1);
          }
          lt(i);
        }
        return !0;
      },
      ownKeys(u) {
        fe(i);
        var f = Reflect.ownKeys(u).filter((_) => {
          var c = n.get(_);
          return c === void 0 || c.v !== A;
        });
        for (var [a, d] of n)
          d.v !== A && !(a in u) && f.push(a);
        return f;
      },
      setPrototypeOf() {
        Or();
      }
    }
  );
}
var dn, Un, Hn, zn;
function ti() {
  if (dn === void 0) {
    dn = window, Un = /Firefox/.test(navigator.userAgent);
    var e = Element.prototype, t = Node.prototype, n = Text.prototype;
    Hn = it(t, "firstChild").get, zn = it(t, "nextSibling").get, on(e) && (e[Er] = void 0, e[yr] = null, e[br] = void 0, e.__e = void 0), on(n) && (n[Xe] = void 0);
  }
}
function Ot(e = "") {
  return document.createTextNode(e);
}
// @__NO_SIDE_EFFECTS__
function Gn(e) {
  return (
    /** @type {TemplateNode | null} */
    Hn.call(e)
  );
}
// @__NO_SIDE_EFFECTS__
function rn(e) {
  return (
    /** @type {TemplateNode | null} */
    zn.call(e)
  );
}
function Vt(e, t) {
  return /* @__PURE__ */ Gn(e);
}
function ni(e, t = 1, n = !1) {
  let r = e;
  for (; t--; )
    r = /** @type {TemplateNode} */
    /* @__PURE__ */ rn(r);
  return r;
}
function ri() {
  return !1;
}
function ii(e, t, n) {
  return (
    /** @type {T extends keyof HTMLElementTagNameMap ? HTMLElementTagNameMap[T] : Element} */
    n ? document.createElement(e, { is: n }) : document.createElement(e)
  );
}
function si(e) {
  E === null && (y === null && Ar(), xr()), Ee && Tr();
}
function li(e, t) {
  var n = t.last;
  n === null ? t.last = t.first = e : (n.next = e, e.prev = n, t.last = e);
}
function be(e, t) {
  var n = E;
  n !== null && (n.f & K) !== 0 && (e |= K);
  var r = {
    ctx: F,
    deps: null,
    nodes: null,
    f: e | C | $,
    first: null,
    fn: t,
    last: null,
    next: null,
    parent: n,
    b: n && n.b,
    prev: null,
    teardown: null,
    wv: 0,
    ac: null
  };
  w == null || w.register_created_effect(r);
  var i = r;
  if ((e & ut) !== 0)
    Le !== null ? Le.push(r) : De.ensure().schedule(r);
  else if (t !== null) {
    try {
      Qe(r);
    } catch (o) {
      throw Y(r), o;
    }
    i.deps === null && i.teardown === null && i.nodes === null && i.first === i.last && // either `null`, or a singular child
    (i.f & Ze) === 0 && (i = i.first, (e & X) !== 0 && (e & Ke) !== 0 && i !== null && (i.f |= Ke));
  }
  if (i !== null && (i.parent = n, n !== null && li(i, n), y !== null && (y.f & N) !== 0 && (e & pe) === 0)) {
    var l = (
      /** @type {Derived} */
      y
    );
    (l.effects ?? (l.effects = [])).push(i);
  }
  return r;
}
function sn() {
  return y !== null && !ee;
}
function fi(e) {
  const t = be(Ft, null);
  return T(t, R), t.teardown = e, t;
}
function ui(e) {
  si();
  var t = (
    /** @type {Effect} */
    E.f
  ), n = !y && (t & re) !== 0 && F !== null && !F.i;
  if (n) {
    var r = (
      /** @type {ComponentContext} */
      F
    );
    (r.e ?? (r.e = [])).push(e);
  } else
    return $n(e);
}
function $n(e) {
  return be(ut | mr, e);
}
function oi(e) {
  De.ensure();
  const t = be(pe | Ze, e);
  return (n = {}) => new Promise((r) => {
    n.outro ? ft(t, () => {
      Y(t), r(void 0);
    }) : (Y(t), r(void 0));
  });
}
function ai(e) {
  return be(qe | Ze, e);
}
function ci(e, t = 0) {
  return be(Ft | t, e);
}
function vn(e, t = [], n = [], r = []) {
  zr(r, t, n, (i) => {
    be(Ft, () => {
      e(...i.map(fe));
    });
  });
}
function Kn(e, t = 0) {
  var n = be(X | t, e);
  return n;
}
function ae(e) {
  return be(re | Ze, e);
}
function Wn(e) {
  var t = e.teardown;
  if (t !== null) {
    const n = Ee, r = y;
    pn(!0), W(null);
    try {
      t.call(null);
    } finally {
      pn(n), W(r);
    }
  }
}
function ln(e, t = !1) {
  var n = e.first;
  for (e.first = e.last = null; n !== null; ) {
    const i = n.ac;
    i !== null && Mt(() => {
      i.abort(_t);
    });
    var r = n.next;
    (n.f & pe) !== 0 ? n.parent = null : Y(n, t), n = r;
  }
}
function hi(e) {
  for (var t = e.first; t !== null; ) {
    var n = t.next;
    (t.f & re) === 0 && Y(t), t = n;
  }
}
function Y(e, t = !0) {
  var n = !1;
  (t || (e.f & wr) !== 0) && e.nodes !== null && e.nodes.end !== null && (di(
    e.nodes.start,
    /** @type {TemplateNode} */
    e.nodes.end
  ), n = !0), e.f |= an, ln(e, t && !n), ot(e, 0);
  var r = e.nodes && e.nodes.t;
  if (r !== null)
    for (const l of r)
      l.stop();
  Wn(e), e.f ^= an, e.f |= ne;
  var i = e.parent;
  i !== null && i.first !== null && Qn(e), e.next = e.prev = e.teardown = e.ctx = e.deps = e.fn = e.nodes = e.ac = e.b = null;
}
function di(e, t) {
  for (; e !== null; ) {
    var n = e === t ? null : /* @__PURE__ */ rn(e);
    e.remove(), e = n;
  }
}
function Qn(e) {
  var t = e.parent, n = e.prev, r = e.next;
  n !== null && (n.next = r), r !== null && (r.prev = n), t !== null && (t.first === e && (t.first = r), t.last === e && (t.last = n));
}
function ft(e, t, n = !0) {
  var r = [];
  Jn(e, r, !0);
  var i = () => {
    n && Y(e), t && t();
  }, l = r.length;
  if (l > 0) {
    var o = () => --l || i();
    for (var u of r)
      u.out(o);
  } else
    i();
}
function Jn(e, t, n) {
  if ((e.f & K) === 0) {
    e.f ^= K;
    var r = e.nodes && e.nodes.t;
    if (r !== null)
      for (const u of r)
        (u.is_global || n) && t.push(u);
    for (var i = e.first; i !== null; ) {
      var l = i.next;
      if ((i.f & pe) === 0) {
        var o = (i.f & Ke) !== 0 || // If this is a branch effect without a block effect parent,
        // it means the parent block effect was pruned. In that case,
        // transparency information was transferred to the branch effect.
        (i.f & re) !== 0 && (e.f & X) !== 0;
        Jn(i, t, o ? n : !1);
      }
      i = l;
    }
  }
}
function _n(e) {
  Zn(e, !0);
}
function Zn(e, t) {
  if ((e.f & K) !== 0) {
    e.f ^= K, (e.f & R) === 0 && (T(e, C), De.ensure().schedule(e));
    for (var n = e.first; n !== null; ) {
      var r = n.next, i = (n.f & Ke) !== 0 || (n.f & re) !== 0;
      Zn(n, i ? t : !1), n = r;
    }
    var l = e.nodes && e.nodes.t;
    if (l !== null)
      for (const o of l)
        (o.is_global || t) && o.in();
  }
}
function Xn(e, t) {
  if (e.nodes)
    for (var n = e.nodes.start, r = e.nodes.end; n !== null; ) {
      var i = n === r ? null : /* @__PURE__ */ rn(n);
      t.append(n), n = i;
    }
}
let kt = !1, Ee = !1;
function pn(e) {
  Ee = e;
}
let y = null, ee = !1;
function W(e) {
  y = e;
}
let E = null;
function oe(e) {
  E = e;
}
let ue = null;
function vi(e) {
  y !== null && (ue ?? (ue = /* @__PURE__ */ new Set())).add(e);
}
let I = null, q = 0, U = null;
function _i(e) {
  U = e;
}
let er = 1, ke = 0, Oe = ke;
function gn(e) {
  Oe = e;
}
function tr() {
  return ++er;
}
function pt(e) {
  var t = e.f;
  if ((t & C) !== 0)
    return !0;
  if (t & N && (e.f &= ~Ne), (t & te) !== 0) {
    for (var n = (
      /** @type {Value[]} */
      e.deps
    ), r = n.length, i = 0; i < r; i++) {
      var l = n[i];
      if (pt(
        /** @type {Derived} */
        l
      ) && Fn(
        /** @type {Derived} */
        l
      ), l.wv > e.wv)
        return !0;
    }
    (t & $) !== 0 && // During time traveling we don't want to reset the status so that
    // traversal of the graph in the other batches still happens
    D === null && T(e, R);
  }
  return !1;
}
function nr(e, t, n = !0) {
  var r = e.reactions;
  if (r !== null && !(ue !== null && ue.has(e)))
    for (var i = 0; i < r.length; i++) {
      var l = r[i];
      (l.f & N) !== 0 ? nr(
        /** @type {Derived} */
        l,
        t,
        !1
      ) : t === l && (n ? T(l, C) : (l.f & R) !== 0 && T(l, te), nn(
        /** @type {Effect} */
        l
      ));
    }
}
function rr(e) {
  var p;
  var t = I, n = q, r = U, i = y, l = ue, o = F, u = ee, f = Oe, a = e.f;
  I = /** @type {null | Value[]} */
  null, q = 0, U = null, y = (a & (re | pe)) === 0 ? e : null, ue = null, We(e.ctx), ee = !1, Oe = ++ke, e.ac !== null && (Mt(() => {
    e.ac.abort(_t);
  }), e.ac = null);
  try {
    e.f |= xt;
    var d = (
      /** @type {Function} */
      e.fn
    ), _ = d();
    e.f |= Je;
    var c = e.deps, h = w == null ? void 0 : w.is_fork;
    if (I !== null) {
      var v;
      if (h || ot(e, q), c !== null && q > 0)
        for (c.length = q + I.length, v = 0; v < I.length; v++)
          c[q + v] = I[v];
      else
        e.deps = c = I;
      if (sn() && (e.f & $) !== 0)
        for (v = q; v < c.length; v++)
          ((p = c[v]).reactions ?? (p.reactions = [])).push(e);
    } else !h && c !== null && q < c.length && (ot(e, q), c.length = q);
    if (Cn() && U !== null && !ee && c !== null && (e.f & (N | te | C)) === 0)
      for (v = 0; v < /** @type {Source[]} */
      U.length; v++)
        nr(
          U[v],
          /** @type {Effect} */
          e
        );
    if (i !== null && i !== e) {
      if (ke++, i.deps !== null)
        for (let S = 0; S < n; S += 1)
          i.deps[S].rv = ke;
      if (t !== null)
        for (const S of t)
          S.rv = ke;
      U !== null && (r === null ? r = U : r.push(.../** @type {Source[]} */
      U));
    }
    return (e.f & ye) !== 0 && (e.f ^= ye), _;
  } catch (S) {
    return On(S);
  } finally {
    e.f ^= xt, I = t, q = n, U = r, y = i, ue = l, We(o), ee = u, Oe = f;
  }
}
function pi(e, t) {
  let n = t.reactions;
  if (n !== null) {
    var r = ar.call(n, e);
    if (r !== -1) {
      var i = n.length - 1;
      i === 0 ? n = t.reactions = null : (n[r] = n[i], n.pop());
    }
  }
  if (n === null && (t.f & N) !== 0 && // Destroying a child effect while updating a parent effect can cause a dependency to appear
  // to be unused, when in fact it is used by the currently-updating parent. Checking `new_deps`
  // allows us to skip the expensive work of disconnecting and immediately reconnecting it
  (I === null || !St.call(I, t))) {
    var l = (
      /** @type {Derived} */
      t
    );
    (l.f & $) !== 0 && (l.f ^= $, l.f &= ~Ne), l.v !== A && Xt(l), l.ac !== null && Mt(() => {
      l.ac.abort(_t), l.ac = null, T(l, C);
    }), Qr(l), ot(l, 0);
  }
}
function ot(e, t) {
  var n = e.deps;
  if (n !== null)
    for (var r = t; r < n.length; r++)
      pi(e, n[r]);
}
function Qe(e) {
  var t = e.f;
  if ((t & ne) === 0) {
    T(e, R);
    var n = E, r = kt;
    E = e, kt = (t & (re | pe)) === 0;
    try {
      (t & (X | Tn)) !== 0 ? hi(e) : ln(e), Wn(e);
      var i = rr(e);
      e.teardown = typeof i == "function" ? i : null, e.wv = er;
      var l;
    } finally {
      kt = r, E = n;
    }
  }
}
function fe(e) {
  var t = e.f, n = (t & N) !== 0;
  if (y !== null && !ee) {
    var r = E !== null && (E.f & ne) !== 0;
    if (!r && (ue === null || !ue.has(e))) {
      var i = y.deps;
      if ((y.f & xt) !== 0)
        e.rv < ke && (e.rv = ke, I === null && i !== null && i[q] === e ? q++ : I === null ? I = [e] : I.push(e));
      else {
        y.deps ?? (y.deps = []), St.call(y.deps, e) || y.deps.push(e);
        var l = e.reactions;
        l === null ? e.reactions = [y] : St.call(l, y) || l.push(y);
      }
    }
  }
  if (Ee && Ce.has(e))
    return Ce.get(e);
  if (n) {
    var o = (
      /** @type {Derived} */
      e
    );
    if (Ee) {
      var u = o.v;
      return ((o.f & R) === 0 && o.reactions !== null || sr(o)) && (u = en(o)), Ce.set(o, u), u;
    }
    var f = (o.f & $) === 0 && !ee && y !== null && (kt || (y.f & $) !== 0), a = (o.f & Je) === 0;
    pt(o) && (f && (o.f |= $), Fn(o)), f && !a && (Mn(o), ir(o));
  }
  if (D != null && D.has(e))
    return D.get(e);
  if ((e.f & ye) !== 0)
    throw e.v;
  return e.v;
}
function ir(e) {
  if (e.f |= $, e.deps !== null)
    for (const t of e.deps)
      (t.reactions ?? (t.reactions = [])).push(e), (t.f & N) !== 0 && (t.f & $) === 0 && (Mn(
        /** @type {Derived} */
        t
      ), ir(
        /** @type {Derived} */
        t
      ));
}
function sr(e) {
  if (e.v === A) return !0;
  if (e.deps === null) return !1;
  for (const t of e.deps)
    if (Ce.has(t) || (t.f & N) !== 0 && sr(
      /** @type {Derived} */
      t
    ))
      return !0;
  return !1;
}
function lr(e) {
  var t = ee;
  try {
    return ee = !0, e();
  } finally {
    ee = t;
  }
}
const gi = ["touchstart", "touchmove"];
function wi(e) {
  return gi.includes(e);
}
const gt = Symbol("events"), mi = /* @__PURE__ */ new Set(), wn = /* @__PURE__ */ new Set();
let mn = null;
function yn(e) {
  var p, S;
  var t = this, n = (
    /** @type {Node} */
    t.ownerDocument
  ), r = e.type, i = ((p = e.composedPath) == null ? void 0 : p.call(e)) || [], l = (
    /** @type {null | Element} */
    i[0] || e.target
  );
  mn = e;
  var o = 0, u = mn === e && e[gt];
  if (u) {
    var f = i.indexOf(u);
    if (f !== -1 && (t === document || t === /** @type {any} */
    window)) {
      e[gt] = t;
      return;
    }
    var a = i.indexOf(t);
    if (a === -1)
      return;
    f <= a && (o = f);
  }
  if (l = /** @type {Element} */
  i[o] || e.target, l !== t) {
    hr(e, "currentTarget", {
      configurable: !0,
      get() {
        return l || n;
      }
    });
    var d = y, _ = E;
    W(null), oe(null);
    try {
      for (var c, h = []; l !== null && l !== t; ) {
        try {
          var v = (S = l[gt]) == null ? void 0 : S[r];
          v != null && (!/** @type {any} */
          l.disabled || // DOM could've been updated already by the time this is reached, so we check this as well
          // -> the target could not have been disabled because it emits the event in the first place
          e.target === l) && v.call(l, e);
        } catch (O) {
          c ? h.push(O) : c = O;
        }
        if (e.cancelBubble) break;
        o++, l = o < i.length ? (
          /** @type {Element} */
          i[o]
        ) : null;
      }
      if (c) {
        for (let O of h)
          queueMicrotask(() => {
            throw O;
          });
        throw c;
      }
    } finally {
      e[gt] = t, delete e.currentTarget, W(d), oe(_);
    }
  }
}
var bn;
const Yt = (
  // We gotta write it like this because after downleveling the pure comment may end up in the wrong location
  ((bn = globalThis == null ? void 0 : globalThis.window) == null ? void 0 : bn.trustedTypes) && /* @__PURE__ */ globalThis.window.trustedTypes.createPolicy("svelte-trusted-html", {
    /** @param {string} html */
    createHTML: (e) => e
  })
);
function yi(e) {
  return (
    /** @type {string} */
    (Yt == null ? void 0 : Yt.createHTML(e)) ?? e
  );
}
function Ei(e) {
  var t = ii("template");
  return t.innerHTML = yi(e.replaceAll("<!>", "<!---->")), t.content;
}
function bi(e, t) {
  var n = (
    /** @type {Effect} */
    E
  );
  n.nodes === null && (n.nodes = { start: e, end: t, a: null, t: null });
}
// @__NO_SIDE_EFFECTS__
function fn(e, t) {
  var n = (t & Pr) !== 0, r, i = !e.startsWith("<!>");
  return () => {
    r === void 0 && (r = Ei(i ? e : "<!>" + e), r = /** @type {TemplateNode} */
    /* @__PURE__ */ Gn(r));
    var l = (
      /** @type {TemplateNode} */
      n || Un ? document.importNode(r, !0) : r.cloneNode(!0)
    );
    return bi(l, l), l;
  };
}
function Ut(e, t) {
  e !== null && e.before(
    /** @type {Node} */
    t
  );
}
function En(e, t) {
  var n = t == null ? "" : typeof t == "object" ? `${t}` : t;
  n !== /** @type {any} */
  (e[Xe] ?? (e[Xe] = e.nodeValue)) && (e[Xe] = n, e.nodeValue = `${n}`);
}
function ki(e, t) {
  return Si(e, t);
}
const wt = /* @__PURE__ */ new Map();
function Si(e, { target: t, anchor: n, props: r = {}, events: i, context: l, intro: o = !0, transformError: u }) {
  ti();
  var f = void 0, a = oi(() => {
    var d = n ?? t.appendChild(Ot());
    Br(
      /** @type {TemplateNode} */
      d,
      {
        pending: () => {
        }
      },
      (h) => {
        An({});
        var v = (
          /** @type {ComponentContext} */
          F
        );
        l && (v.c = l), i && (r.$$events = i), f = e(h, r) || {}, Rn();
      },
      u
    );
    var _ = /* @__PURE__ */ new Set(), c = (h) => {
      for (var v = 0; v < h.length; v++) {
        var p = h[v];
        if (!_.has(p)) {
          _.add(p);
          var S = wi(p);
          for (const Fe of [t, document]) {
            var O = wt.get(Fe);
            O === void 0 && (O = /* @__PURE__ */ new Map(), wt.set(Fe, O));
            var Pe = O.get(p);
            Pe === void 0 ? (Fe.addEventListener(p, yn, { passive: S }), O.set(p, 1)) : O.set(p, Pe + 1);
          }
        }
      }
    };
    return c(cr(mi)), wn.add(c), () => {
      var S;
      for (var h of _)
        for (const O of [t, document]) {
          var v = (
            /** @type {Map<string, number>} */
            wt.get(O)
          ), p = (
            /** @type {number} */
            v.get(h)
          );
          --p == 0 ? (O.removeEventListener(h, yn), v.delete(h), v.size === 0 && wt.delete(O)) : v.set(h, p);
        }
      wn.delete(c), d !== n && ((S = d.parentNode) == null || S.removeChild(d));
    };
  });
  return Jt.set(f, a), f;
}
let Jt = /* @__PURE__ */ new WeakMap();
function Ti(e, t) {
  const n = Jt.get(e);
  return n ? (Jt.delete(e), n(t)) : Promise.resolve();
}
var Z, le, V, Re, dt, vt, Pt;
class xi {
  /**
   * @param {TemplateNode} anchor
   * @param {boolean} transition
   */
  constructor(t, n = !0) {
    /** @type {TemplateNode} */
    L(this, "anchor");
    /** @type {Map<Batch, Key>} */
    m(this, Z, /* @__PURE__ */ new Map());
    /**
     * Map of keys to effects that are currently rendered in the DOM.
     * These effects are visible and actively part of the document tree.
     * Example:
     * ```
     * {#if condition}
     * 	foo
     * {:else}
     * 	bar
     * {/if}
     * ```
     * Can result in the entries `true->Effect` and `false->Effect`
     * @type {Map<Key, Effect>}
     */
    m(this, le, /* @__PURE__ */ new Map());
    /**
     * Similar to #onscreen with respect to the keys, but contains branches that are not yet
     * in the DOM, because their insertion is deferred.
     * @type {Map<Key, Branch>}
     */
    m(this, V, /* @__PURE__ */ new Map());
    /**
     * Keys of effects that are currently outroing
     * @type {Set<Key>}
     */
    m(this, Re, /* @__PURE__ */ new Set());
    /**
     * Whether to pause (i.e. outro) on change, or destroy immediately.
     * This is necessary for `<svelte:element>`
     */
    m(this, dt, !0);
    /**
     * @param {Batch} batch
     */
    m(this, vt, (t) => {
      if (s(this, Z).has(t)) {
        var n = (
          /** @type {Key} */
          s(this, Z).get(t)
        ), r = s(this, le).get(n);
        if (r)
          _n(r), s(this, Re).delete(n);
        else {
          var i = s(this, V).get(n);
          i && (_n(i.effect), s(this, le).set(n, i.effect), s(this, V).delete(n), i.fragment.lastChild.remove(), this.anchor.before(i.fragment), r = i.effect);
        }
        for (const [l, o] of s(this, Z)) {
          if (s(this, Z).delete(l), l === t)
            break;
          const u = s(this, V).get(o);
          u && (Y(u.effect), s(this, V).delete(o));
        }
        for (const [l, o] of s(this, le)) {
          if (l === n || s(this, Re).has(l)) continue;
          const u = () => {
            if (Array.from(s(this, Z).values()).includes(l)) {
              var a = document.createDocumentFragment();
              Xn(o, a), a.append(Ot()), s(this, V).set(l, { effect: o, fragment: a });
            } else
              Y(o);
            s(this, Re).delete(l), s(this, le).delete(l);
          };
          s(this, dt) || !r ? (s(this, Re).add(l), ft(o, u, !1)) : u();
        }
      }
    });
    /**
     * @param {Batch} batch
     */
    m(this, Pt, (t) => {
      s(this, Z).delete(t);
      const n = Array.from(s(this, Z).values());
      for (const [r, i] of s(this, V))
        n.includes(r) || (Y(i.effect), s(this, V).delete(r));
    });
    this.anchor = t, g(this, dt, n);
  }
  /**
   *
   * @param {any} key
   * @param {null | ((target: TemplateNode) => void)} fn
   */
  ensure(t, n) {
    var r = (
      /** @type {Batch} */
      w
    ), i = ri();
    if (n && !s(this, le).has(t) && !s(this, V).has(t))
      if (i) {
        var l = document.createDocumentFragment(), o = Ot();
        l.append(o), s(this, V).set(t, {
          effect: ae(() => n(o)),
          fragment: l
        });
      } else
        s(this, le).set(
          t,
          ae(() => n(this.anchor))
        );
    if (s(this, Z).set(r, t), i) {
      for (const [u, f] of s(this, le))
        u === t ? r.unskip_effect(f) : r.skip_effect(f);
      for (const [u, f] of s(this, V))
        u === t ? r.unskip_effect(f.effect) : r.skip_effect(f.effect);
      r.oncommit(s(this, vt)), r.ondiscard(s(this, Pt));
    } else
      s(this, vt).call(this, r);
  }
}
Z = new WeakMap(), le = new WeakMap(), V = new WeakMap(), Re = new WeakMap(), dt = new WeakMap(), vt = new WeakMap(), Pt = new WeakMap();
function Ai(e, t, n = !1) {
  var r = new xi(e), i = n ? Ke : 0;
  function l(o, u) {
    r.ensure(o, u);
  }
  Kn(() => {
    var o = !1;
    t((u, f = 0) => {
      o = !0, l(f, u);
    }), o || l(-1, null);
  }, i);
}
function Ri(e) {
  F === null && kr(), ui(() => {
    const t = lr(e);
    if (typeof t == "function") return (
      /** @type {() => void} */
      t
    );
  });
}
const Ci = "5";
var kn;
typeof window < "u" && ((kn = window.__svelte ?? (window.__svelte = {})).v ?? (kn.v = /* @__PURE__ */ new Set())).add(Ci);
async function Oi(e, t) {
  var i;
  const n = await fetch("/graphql", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query: e, variables: t })
  });
  if (!n.ok) throw new Error(`GraphQL HTTP ${n.status}`);
  const r = await n.json();
  if ((i = r.errors) != null && i.length)
    throw new Error(r.errors.map((l) => l.message).join(", "));
  return r.data;
}
async function Ni() {
  return (await Oi("query { notesPing }")).notesPing;
}
var Di = /* @__PURE__ */ fn('<p class="err svelte-n50uah"> </p>'), Pi = /* @__PURE__ */ fn("<p> </p>"), Fi = /* @__PURE__ */ fn('<div class="wrap svelte-n50uah"><h1>Notes</h1> <p>Bootstrapped Svelte app <code>notes</code>.</p> <!></div>');
function Mi(e, t) {
  An(t, !0);
  let n = /* @__PURE__ */ ie("…"), r = /* @__PURE__ */ ie("");
  Ri(async () => {
    try {
      ce(n, await Ni(), !0);
    } catch (f) {
      ce(r, f instanceof Error ? f.message : String(f), !0);
    }
  });
  var i = Fi(), l = ni(Vt(i), 4);
  {
    var o = (f) => {
      var a = Di(), d = Vt(a);
      vn(() => En(d, fe(r))), Ut(f, a);
    }, u = (f) => {
      var a = Pi(), d = Vt(a);
      vn(() => En(d, `GraphQL: ${fe(n) ?? ""}`)), Ut(f, a);
    };
    Ai(l, (f) => {
      fe(r) ? f(o) : f(u, -1);
    });
  }
  Ut(e, i), Rn();
}
let mt = null;
function ji() {
  if (document.querySelector('link[data-app-css="notes"]')) return;
  const e = document.createElement("link");
  e.rel = "stylesheet", e.href = "/app-assets/notes/spa.css", e.dataset.appCss = "notes", document.head.appendChild(e);
}
const qi = {
  async mount(e) {
    ji(), mt = ki(Mi, { target: e });
  },
  unmount() {
    mt && (Ti(mt), mt = null);
  }
};
export {
  qi as default
};
