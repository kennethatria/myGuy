/**
 * Whether a page of notes is being shown with distances: the server tags a
 * note only when the viewer sent a location and the poster shared one, so a
 * page with no tags at all means distances aren't in play (no viewer
 * location, or the proximity service fell back), and no shrugs should show.
 */
export function hasDistances(notes: { distance?: string }[]): boolean {
  return notes.some((note) => !!note.distance)
}
