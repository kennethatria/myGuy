// Marketplace notes. A listing has no price: it goes in the note or is
// agreed in chat (store-service accepts none).

/** "1 offer", "3 offers": listings sellers made for a request. */
export function offersLabel(count: number): string {
  return count === 1 ? '1 offer' : `${count} offers`
}
