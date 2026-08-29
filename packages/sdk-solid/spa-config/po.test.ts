import assert from 'node:assert/strict'
import { test } from 'node:test'
import { parsePO } from './po.ts'

test('parsePO parses msgid/msgstr and skips fuzzy and empty', () => {
  const msgs = parsePO(`
msgid "inventory.products.title"
msgstr "Products"

#, fuzzy
msgid "skip.fuzzy"
msgstr "nope"

msgid "skip.empty"
msgstr ""

msgid "multi"
msgstr ""
"line "
"two"
`)
  assert.equal(msgs['inventory.products.title'], 'Products')
  assert.equal(msgs['skip.fuzzy'], undefined)
  assert.equal(msgs['skip.empty'], undefined)
  assert.equal(msgs.multi, 'line two')
})
