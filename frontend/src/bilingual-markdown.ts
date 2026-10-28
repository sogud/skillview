import type { Nodes, Root, Paragraph, Heading, TableCell } from 'mdast'
import type { Plugin } from 'unified'

type Options = {
  text: string
  language: string
  originalLabel: string
  translationLabel: string
}

function isProse(node: Nodes): node is Paragraph | Heading | TableCell {
  return ['paragraph', 'heading', 'tableCell'].includes(node.type)
}

// Providers replace inline text, preserving the surrounding Markdown. Compare
// block structure before pairing so an old/unexpected result cannot misalign it.
function sameBlocks(original: Nodes, translated: Nodes): boolean {
  if (original.type !== translated.type) return false
  if (isProse(original)) return true
  if ('children' in original && 'children' in translated)
    return (
      original.children.length === translated.children.length &&
      original.children.every((child, index) =>
        sameBlocks(child, translated.children[index]),
      )
    )
  return true
}

export const remarkBilingual: Plugin<[Options], Root> = function (options) {
  return (tree, file) => {
    // Reuse ReactMarkdown's parser and GFM configuration, including definitions.
    const translated = this.parse(options.text) as Root
    if (!sameBlocks(tree, translated)) {
      const label = (value: string): Paragraph => ({
        type: 'paragraph',
        children: [{ type: 'strong', children: [{ type: 'text', value }] }],
      })
      // Keep both complete documents readable if their block shapes differ.
      tree.children = [
        label(options.originalLabel),
        ...tree.children,
        { type: 'thematicBreak' },
        label(options.translationLabel),
        ...translated.children,
      ]
      return
    }
    const source = String(file)
    function pair(original: Nodes, target: Nodes) {
      if (isProse(original) && isProse(target)) {
        const originalText = source.slice(
          original.position?.start.offset,
          original.position?.end.offset,
        )
        const translatedText = options.text.slice(
          target.position?.start.offset,
          target.position?.end.offset,
        )
        if (originalText === translatedText) return
        original.children.push({
          type: 'emphasis',
          data: {
            hName: 'span',
            hProperties: {
              className: ['bilingual-translation'],
              lang: options.language,
            },
          },
          children: target.children,
        })
      } else if ('children' in original && 'children' in target) {
        original.children.forEach((child, index) =>
          pair(child, target.children[index]),
        )
      }
    }
    pair(tree, translated)
  }
}
