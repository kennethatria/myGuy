// Browser tab title: "<page> · MyGuy", or just "MyGuy" without a page name.
export function setPageTitle(page?: string) {
  document.title = page ? `${page} · MyGuy` : 'MyGuy'
}
