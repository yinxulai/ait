function isEditableTarget(target: EventTarget | null): target is HTMLElement {
  if (!(target instanceof HTMLElement)) {
    return false
  }

  if (target.isContentEditable) {
    return true
  }

  const tag = target.tagName
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT"
}

function shouldIgnoreShortcut(event: KeyboardEvent): boolean {
  if (event.ctrlKey || event.metaKey || event.altKey) {
    return false
  }

  if (event.isComposing) {
    return false
  }

  return isEditableTarget(event.target) && event.key.length === 1
}

function installGuard(): void {
  window.addEventListener(
    "keydown",
    (event) => {
      if (!shouldIgnoreShortcut(event)) {
        return
      }

      event.stopImmediatePropagation()
    },
    true
  )
}

if (typeof window !== "undefined") {
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", installGuard, { once: true })
  } else {
    installGuard()
  }
}
