const copy = {
  en: {
    skip: 'Skip to content',
    eyebrow: 'A small macOS app for Agent Skills',
    headlineFirst: 'Your skills.',
    headlineSecond: 'In one place.',
    intro: 'Browse, read and translate your local Skills.',
    download: 'Download for macOS ↓',
    source: 'Explore the source ↗',
    requirements: 'Free & open source · Apple Silicon · Version 0.1.8',
    caption: 'Actual macOS app with example content.',
    preview: 'SkillView native macOS library window with example Skills.',
    findTitle: 'Your Agents. One library.',
    findBody: 'Browse installed Agents. Shared Skills appear once.',
    readTitle: 'Files, easy to read.',
    readBody: 'Read Skills and installed Claude Code/Codex plugins in a file tree.',
    translateTitle: 'Read in your language.',
    translateBody: 'Google, Youdao, self-hosted LibreTranslate or AI. Translations are cached.',
    localEyebrow: 'On your Mac',
    localTitle: 'Local by default.',
    localBody: 'Scan manually. Archive and restore Skills. Choose which folders allow editing.',
    startTitle: 'Get started.',
    startBody: 'Download the app, move it to Applications, then click Scan.',
    installNote: 'macOS 13+ (minimum version unverified). Ad-hoc signed; not Apple notarized.',
    installHelp: 'Apple’s guide to opening downloaded apps ↗',
    footer: 'MIT licensed.',
    releases: 'Release notes',
    feedback: 'Feedback',
  },
  'zh-CN': {
    skip: '跳转到正文',
    eyebrow: '一个轻巧的 macOS Agent Skills 阅读工具',
    headlineFirst: '你的技能，',
    headlineSecond: '一处看全。',
    intro: '集中查看、阅读和翻译本机技能。',
    download: '下载 macOS 版 ↓',
    source: '查看源代码 ↗',
    requirements: '免费开源 · Apple Silicon · 版本 0.1.8',
    caption: '真实 macOS 应用界面，使用示例内容。',
    preview: 'SkillView 原生 macOS 技能列表窗口，使用示例内容。',
    findTitle: '多个 Agent，一个技能库。',
    findBody: '查看已安装 Agent 的技能，相同技能合并展示。',
    readTitle: '像文件一样阅读。',
    readBody: '通过文件树阅读技能及已安装的 Claude Code、Codex 插件。',
    translateTitle: '用熟悉的语言阅读。',
    translateBody: '支持 Google、有道、自建 LibreTranslate 和 AI，译文自动缓存。',
    localEyebrow: '保存在本机',
    localTitle: '从本机开始。',
    localBody: '手动扫描，支持归档和恢复，自行选择可编辑目录。',
    startTitle: '开始使用。',
    startBody: '下载后移入“应用程序”，打开并点击“扫描”。',
    installNote: '声明支持 macOS 13+，最低版本未实测。ad-hoc 签名，未经 Apple 公证。',
    installHelp: 'Apple 关于打开下载应用的说明 ↗',
    footer: 'MIT 开源。',
    releases: '更新记录',
    feedback: '反馈问题',
  },
}

const languageButton = document.querySelector('#language')
let language = navigator.language.startsWith('zh') ? 'zh-CN' : 'en'
try {
  const saved = localStorage.getItem('skillview-site-language')
  if (saved === 'en' || saved === 'zh-CN') language = saved
} catch {}

function showLanguage() {
  document.documentElement.lang = language
  for (const element of document.querySelectorAll('[data-copy]')) {
    element.textContent = copy[language][element.dataset.copy]
  }
  for (const element of document.querySelectorAll('[data-alt]')) {
    element.alt = copy[language][element.dataset.alt]
  }
  languageButton.textContent = language === 'en' ? '中文' : 'English'
  languageButton.setAttribute(
    'aria-label',
    language === 'en' ? '切换为中文' : 'Switch to English',
  )
  document.title =
    language === 'en'
      ? 'SkillView — A home for your Agent Skills'
      : 'SkillView — 你的 Agent Skills 阅读工具'
}

languageButton.addEventListener('click', () => {
  language = language === 'en' ? 'zh-CN' : 'en'
  showLanguage()
  try {
    localStorage.setItem('skillview-site-language', language)
  } catch {}
})
showLanguage()
