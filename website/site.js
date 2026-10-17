const copy = {
  en: {
    skip: 'Skip to content',
    eyebrow: 'A small macOS app for Agent Skills',
    headlineFirst: 'Your skills.',
    headlineSecond: 'In one place.',
    intro:
      'Find what you have. Read it comfortably. Keep your Agent Skills close, wherever they live.',
    download: 'Download for macOS ↓',
    source: 'Explore the source ↗',
    requirements: 'Free & open source · Apple Silicon · Version 0.1.1',
    caption: 'A real SkillView interface with example content.',
    preview:
      'SkillView with a file tree on the left and a readable Skill document on the right.',
    findTitle: 'Your Agents. One library.',
    findBody:
      'Browse Skills from Codex, Claude Code, Cursor, Gemini and more. Shared copies appear once, with their Agent logos together.',
    readTitle: 'Room to read.',
    readBody:
      'A familiar file tree, readable Markdown and an adjustable reading size. Switch to Plugins to browse installed Claude Code and Codex plugins, including their names, versions and files.',
    translateTitle: 'Read in your language.',
    translateBody:
      'Translate with Google, Youdao, your own LibreTranslate server or AI. Saved translations stay available across restarts.',
    localEyebrow: 'Your files stay yours',
    localTitle: 'Local by default.',
    localBody:
      'Scan when you choose. Original files stay in their folders. Choose which folders can be edited. Translation sends only the document you select to the service you choose.',
    startTitle: 'A quieter way to explore your Skills.',
    startBody: 'Download the app, move it to Applications, then click Scan.',
    installNote:
      'macOS 13+ declared; minimum OS verification is ongoing. The current download is ad-hoc signed and has not been notarized by Apple.',
    installHelp: 'Apple’s guide to opening downloaded apps ↗',
    footer: 'Made for reading. MIT licensed.',
    releases: 'Release notes',
    feedback: 'Feedback',
  },
  'zh-CN': {
    skip: '跳转到正文',
    eyebrow: '一个轻巧的 macOS Agent Skills 阅读工具',
    headlineFirst: '你的技能，',
    headlineSecond: '一处看全。',
    intro:
      '找到已安装的技能，舒服地阅读。无论它们属于哪个 Agent，都能在这里查看。',
    download: '下载 macOS 版 ↓',
    source: '查看源代码 ↗',
    requirements: '免费开源 · Apple Silicon · 版本 0.1.1',
    caption: 'SkillView 实际界面，使用示例内容。',
    preview: 'SkillView 左侧为文件树，右侧为技能文档阅读区。',
    findTitle: '多个 Agent，一个技能库。',
    findBody:
      '查看 Codex、Claude Code、Cursor、Gemini 等工具的技能。相同技能合并展示，用平台图标标记所属 Agent。',
    readTitle: '留出空间，好好阅读。',
    readBody:
      '熟悉的文件树、清晰的 Markdown、可调整的阅读字号。切换到“插件”，查看本机 Claude Code 和 Codex 插件的名称、版本与文件。',
    translateTitle: '用熟悉的语言阅读。',
    translateBody:
      '支持 Google、有道、自建 LibreTranslate 和 AI 翻译。已保存的译文在重启后仍可复用。',
    localEyebrow: '文件由你管理',
    localTitle: '从本机开始。',
    localBody:
      '点击才扫描，原始文件留在原有目录。由你选择哪些目录允许编辑。翻译时，仅将选中的文档发送到你选择的服务。',
    startTitle: '换一种轻松的方式，阅读你的技能。',
    startBody: '下载后移入“应用程序”，打开并点击“扫描”。',
    installNote:
      '声明支持 macOS 13+，最低系统版本仍待实机验证。当前安装包为 ad-hoc 签名，尚未通过 Apple 公证。',
    installHelp: 'Apple 关于打开下载应用的说明 ↗',
    footer: '为阅读而做。MIT 开源。',
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
