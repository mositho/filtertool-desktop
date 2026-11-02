/** Rule groups share a visible title, so find them by stable subsection key. */
export function ruleButton(root: ParentNode, name: string): HTMLButtonElement {
  const section = root.querySelector(`[data-editor-subsection="${name}"]`)
  if (section) return section.querySelector<HTMLButtonElement>(":scope > div > button[aria-expanded]")!
  return [...root.querySelectorAll<HTMLButtonElement>("button")].find((entry) => entry.textContent?.trim().startsWith(name))!
}
