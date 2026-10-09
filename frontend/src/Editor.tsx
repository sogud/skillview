import { useEffect, useRef } from 'react'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { markdown } from '@codemirror/lang-markdown'
import { basicSetup } from 'codemirror'

export default function Editor({
  text,
  newline,
  onChange,
}: {
  text: string
  newline: string
  onChange: (text: string) => void
}) {
  const root = useRef<HTMLDivElement>(null)
  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange
  useEffect(() => {
    const view = new EditorView({
      parent: root.current!,
      state: EditorState.create({
        doc: text,
        extensions: [
          basicSetup,
          markdown(),
          EditorView.lineWrapping,
          EditorState.lineSeparator.of(newline),
          EditorView.contentAttributes.of({ 'aria-label': 'File editor' }),
          EditorView.updateListener.of((update) => {
            if (update.docChanged) onChangeRef.current(update.state.doc.toString())
          }),
        ],
      }),
    })
    view.focus()
    return () => view.destroy()
    // Document/baseline changes remount this component through its key.
    // Draft updates keep the editor instance and cursor in place.
  }, [])
  return <div className="editor" ref={root} />
}
