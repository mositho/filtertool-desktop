export type ReleaseNoteBlock = { kind: "text" | "CAUTION" | "WARNING"; text: string }

/** Only GitHub warning blocks are interpreted; all content remains plain text. */
export function releaseNoteBlocks(notes: string): ReleaseNoteBlock[] {
  const blocks: ReleaseNoteBlock[] = []
  const lines = notes.replace(/\r\n/g, "\n").split("\n")
  for (let i = 0; i < lines.length; i++) {
    const alert = /^ {0,3}>\s*\[!(CAUTION|WARNING)\]\s*$/.exec(lines[i])
    if (alert) {
      const text: string[] = []
      while (i + 1 < lines.length && /^ {0,3}>/.test(lines[i + 1])) {
        text.push(lines[++i].replace(/^ {0,3}> ?/, ""))
      }
      blocks.push({ kind: alert[1] as "CAUTION" | "WARNING", text: text.join("\n") })
    } else {
      const previous = blocks.at(-1)
      if (previous?.kind === "text") previous.text += "\n" + lines[i]
      else blocks.push({ kind: "text", text: lines[i] })
    }
  }
  return blocks.map((block) => ({ ...block, text: block.text.trim() })).filter((block) => block.text || block.kind !== "text")
}
