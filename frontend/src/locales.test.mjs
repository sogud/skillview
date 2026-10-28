import test from 'node:test'
import assert from 'node:assert/strict'
import cases from '../../internal/locale/matching-cases.json' with { type: 'json' }
import { languages, matchLocale, preferredLocale, resolveLocale, localeDirection, translate } from './locale-core.ts'

test('locale matching agrees with the shared Go fixtures and honors preference order', () => {
  assert.equal(languages.length,18)
  for (const fixture of cases) assert.equal(matchLocale(fixture.input), fixture.expected || undefined, fixture.input)
  assert.equal(preferredLocale(['sv-SE','fr-CA','en']), 'fr')
  assert.equal(preferredLocale([]),'en')
  assert.equal(resolveLocale('system','zh-Hant-HK'),'zh-TW')
  assert.equal(resolveLocale('de','ar-EG'),'de')
  assert.equal(resolveLocale('unknown','ar'),'en')
})

test('English fallback, Chinese interpolation and RTL isolates preserve identifiers', () => {
  assert.equal(translate('unknown','Unlisted {value}',{value:'fixture'}),'Unlisted fixture')
  assert.equal(translate('zh-CN','Report received: {id}',{id:'ab123'}),'已收到报告：ab123')
  assert.equal(localeDirection('ar'),'rtl')
  for (const language of languages.filter((item) => item.id !== 'ar')) assert.equal(localeDirection(language.id),'ltr')
  const result = translate('ar','Report received: {id}',{id:'/fixture/example.md'})
  assert.ok(result.includes('\u2068/fixture/example.md\u2069'))
  assert.equal(translate('en','{a} {b}',{a:'{b}',b:'safe'}),'{b} safe')
  assert.equal(translate('en','Missing {argument}'),'Missing {argument}')
})
