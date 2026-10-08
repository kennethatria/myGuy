# MyGuy mobile redesign: implementation spec

Source mockups: the `.dc.html` files (one per screen, 390×844, inline styles, no logic). Treat them as the visual source of truth. Copy exact values from them. This spec covers behaviour and rules that the mockups can't show.

Goal: minimalist, sticky-note feel, friendly. Mobile-first (390px width).

---

## 1. Design tokens

### Colour
| Token | Value | Use |
|---|---|---|
| `bg` | `#FAFAF8` | page background |
| `surface` | `#FFFFFF` | bars, chat screens, inputs |
| `text` | `#111827` | primary text |
| `text-muted` | `#6B7280` | meta, captions |
| `text-body` | `#4B5563` | secondary body |
| `border` | `#E7E7E3` | dividers |
| `accent` | `#5B94F5` | FAB, filled buttons, active tab underline, unread dot (soft blue) |
| `on-accent` | `#14306B` | **text** on accent fills (white text fails contrast). Icons on accent fills (FAB icon, sheet ✕) are white `#FFFFFF`. |
| `accent-text` | `#2F5FC4` | links, text-only buttons ("+ Post"), accent text on light bg |
| `accent-tint` | `#EAF1FF` | chips, avatars, light accent backgrounds |
| `note-gig` | `#DCE8FF` (alt `#E8F0FF`) | Gig notes (blue) |
| `note-sell` | `#FBEFC0` (alt `#FBF1CE`) | For-sale notes (yellow) |
| `note-want` | `#FADCEB` (alt `#FCE8F2`) | Wanted notes (pink) |
| `price` | `#713F12` | price text on yellow notes |
| `reserved` | `#374151` bg, white text | "Reserved" badge |
| `badge-red` | `#EF4444` | FAB notification dot (2px `#FAFAF8` border) |
| Category dots (radar/legend) | Gigs `#2563EB`, For sale `#F59E0B`, Wanted `#8B5CF6` | |
| Rating: strong ≥4.5 | `#3E7F46` | |
| Rating: fair 3–4.4 | `#B07A2C` | |
| Rating: weak <3 | `#B9402F` | |
| Radar/network rings (outer→inner) | `#F5F9FF`, `#EDF3FF`, `#E2ECFF`, `#D6E4FF`, `#C9DBFF` | cool light blue |

### Type
DM Sans (400/500/600/700). Screen titles 17/600. Note titles 16–18/600–700. Meta 12–13. Body 14–15.

**Handwriting accent (Home only, used sparingly):** Caveat 600 for note titles (21px), prices (21px) and the time-left stamp (16px, colour `#6B87C4`, rotated -2deg). Everything else stays DM Sans. Load via Google Fonts (`Caveat:wght@600;700`).

### Sticky-note component (core pattern)
- Radius 4–6px, no border.
- Shadow: `0 2px 5px rgba(17,24,39,.12), 0 1px 1px rgba(17,24,39,.06)`.
- Slight tilt, alternating `±0.4–0.6deg`. Not on forms or inputs.
- **Home rows use a pin, not tape:** an 8×8 grey dot (`radial-gradient(circle at 35% 30%, #E5E7EB, #9CA3AF)`, shadow `0 1px 1px rgba(17,24,39,.25)`), centred, 6px from the top edge, inside the note. Deliberately quiet.
- Other screens use optional "tape": 48×14 `rgba(255,255,255,.6)`, centred on top edge, radius 2px (photo notes use 64×18).
- Folded corner (used on Chat and Network notes; Home rows no longer have it): 14×14 bottom-right, `linear-gradient(135deg, #FAFAF8 50%, rgba(17,24,39,.14) 50%)`.
- Dashed divider inside notes: `1px dashed rgba(17,24,39,.22)`.
- Photo on a note: white frame, 3–5px padding, 3–4px radius, small shadow, optional ±3° tilt on row thumbnails (44px).
- All content notes show time left until expiry.

### Floating chat button (FAB)
**Icon:** two rounded, overlapping chat bubbles (round, friendly, not rectangular). Front bubble solid white, back bubble white at 70% opacity. SVG path data is in any screen file with a FAB (e.g. `design/screens/Main.dc.html`). **Behaviour:** tap opens the Messages sheet (3.4), it does not navigate to a new page.
56px circle, `accent`, shadow `0 4px 12px rgba(91,148,245,0.40)`, icon white (`#FFFFFF`), red dot top-right when unread. Fixed bottom-right, 16px from edges (20px from bottom). On the Detail screen it sits above the action bar (bottom 128px). Opens the Messages list.

### Navigation
Hamburger left in the header, no bottom tab bar. Back arrow replaces hamburger on detail/chat screens. Text-only "+ Post" top-right (`accent-text`, 15/600).

### Emoji and GIF rules
- Subtle emoji only: categories, status, empty states.
- GIF slots: success moments (post created) and empty states.

---

## 2. Global product rules
1. **Everything expires after 24h**: gigs, items for sale and wanted requests. Show remaining time (e.g. `23h`).
2. **Tapping any note opens its detail page first.** Chat never opens straight from a list.
3. **CTA wording:** Gigs → **Apply**. Items for sale → **Buy Now**. Both have an optional message field.
4. **Chat opens only after the owner approves.** Approval happens inside the chat conversation (there is no separate applicants screen).
5. **Declined applicants** get a generic "didn't work out" notification, with no reason.
6. **Applicants see their status** on an Applications screen (Pending / Approved / Not selected).
7. **Photos:** max 3, only for Sell and Request posts. Never for Gigs. The detail screen shows all 3.
8. Post types: Gig, Sell, Request.

---

## 3. Screens

### 3.1 Home (`Main`)
- Header: hamburger, "Near you", "+ Post".
- **Radar** (300px SVG): 5 rings with labels `<1 km`, `~2`, `~5`, `~10`, `10+`, plus `No loc.`. Centre "You" dot. Coloured dots per note by category. It replaces the old filter chips.
- Legend under the radar: plain text buttons with counts (Gigs n, For sale n, Wanted n). They are pill-less. Tapping one should filter the list and radar by that category (an assumption, not drawn).
- **Note list:** stacked horizontal sticky rows (gap 14, padding 16×16×14, 4px radius, tilt ±0.4–0.6°, grey pin, standard note shadow).
  - Titles and prices in Caveat (see Type). Description and meta in DM Sans.
  - Gig row (blue): title, one-line description; right column distance and a handwritten `Nh left` stamp.
  - Sell row (yellow): framed 44px emoji/photo thumbnail tilted -3°, title, `distance`, price (Caveat) above the stamp.
  - Reserved sell rows show "Reserved" in the stamp slot and no time.
  - Wanted row: pink, same layout as gig.
  - **Fade:** rows fade as expiry nears via `opacity` (23h → 1.0, 21h → 0.97, Reserved → 0.93, 3h → 0.82). Keep the minimum around 0.8 so text stays readable.
- FAB. The "Need something?" prompt was intentionally removed.

### 3.2 New post (`PostGig`)
- Title: **New post**.
- Type switch: Gig / Sell / Request.
- Yellow sticky form with dashed-underline fields: Headline, Note, Price (Sell only).
- **Photos (optional) n/3** only for Sell and Request: filled thumbnails with ✕ and dashed "+" slots. Hidden for Gig.
- Rough-area row, hint "expires in 24h".
- Pinned bottom: "Stick it on the board" (primary) + Cancel.

### 3.3 Posted (`Posted`)
- Header "Your post" (not "Your gig").
- GIF slot with 🎉, "Posted!".
- Blue sticky summary: "Your post · 0 applicants · 24h left".
- Share / Edit buttons, "Remove post".
- Toast "Stuck on the board".

### 3.4 Messages sheet (`Messages`)
**Not a full screen.** Tapping the FAB opens Messages as a bottom sheet over the current page (same behaviour as the live app).
- Sheet: inset 16px left/right, top ≈92px, bottom 24px, white, 16px radius, shadow `0 -4px 24px rgba(17,24,39,.18)`. Page behind is dimmed with `rgba(17,24,39,.28)`. The FAB is hidden while the sheet is open.
- Header bar (64px): `accent` background, "Messages" 20/700 in `on-accent`, close (✕, white stroke) button right: 40×40, 10px radius, `rgba(255,255,255,.35)` fill, 1px `rgba(255,255,255,.6)` border. Tapping ✕ or the dimmed area closes the sheet.
- Tabs: **Active n / All** (default All; not "Done"). Active tab has a 2px `accent` underline.
- Row: initial avatar (circle, accent-tint, `accent-text` letter) to the left, with title and chips to the right. No username label, no distinct icons per type.
- Chips: Active, Expired, Completed (green check), relative time, `★ rating`.
- Unread dot (`accent`) on the right. Expired rows use muted title colour.
- Footer strip (`#F8F9FA`, top border): "A chat opens once the owner approves your application".
- Tapping a row opens the conversation inside the same sheet (see 3.5).

### 3.5 Chat: pending request (`ChatPending`)
Conversation view **inside the Messages sheet**, shown before the owner approves.
- Same sheet frame and `accent` header ("Messages" + ✕).
- Sub-bar (`#F8F9FA`): back arrow + "All conversations" (returns to the list).
- Item header: item title 20/700, "with @username", "View item" chip (`accent-tint`, `accent-text`).
- Sent message as an accent bubble (`accent` bg, `on-accent` text, right-aligned, time).
- **Request note** (yellow sticky, taped): 📩 "Buy Now request sent" (use "Application sent" for gigs), item and price, Pending badge, 3-step tracker **Sent → Seller → Chat** (current step highlighted), "Waiting for the seller to answer", date/time, "Expires in 23h if there's no answer".
- Footer strip instead of a composer: "You can chat once the seller approves your request." After approval the footer becomes a normal message composer.
- States still to build (see §5): Approved (composer enabled, system note "Approved"), Declined (generic "didn't work out").

### 3.6 My stuff (`CreatedGigs`)
- Tabs Live n / Ended n.
- Tilted sticky cards with "Review" and "Remove" buttons. FAB.
- "Review" has no target yet (see §5).

### 3.7 Menu / drawer (`Drawer`)
- **Browse:** Home, Gigs, Marketplace, Network.
- **Mine:** My stuff (formerly "My posts"), Assignments, Applications.
- Messages is NOT in the menu (it is the FAB).
- Footer: user name with rating (e.g. "Silver ⭐ 4.7"), Sign out.

### 3.8 Empty state (`Empty`)
Applications empty: GIF slot (labelled) + 🌱, short copy, "Browse gigs" (primary), "Post your own gig" (text). Use the same pattern for other empty lists.

### 3.9 Item detail (`Detail`), for sale
- Header: back + "For sale".
- Large photo (210px, radius 12) with `1 / 3` counter, 3 thumbnails below (selected outlined with `accent`).
- Yellow sticky: title, price, seller (initial avatar, `@name · ★ rating`, distance), "Posted just now · Expires in 23h".
- Optional message textarea.
- Pinned bar: **Buy Now** (full width, `accent`), caption "A chat opens once the owner approves".
- FAB above the bar.
- Gig and Request detail: same layout, no photos for gigs, CTA **Apply**. Not mocked yet.

### 3.10 Applications status (`Applications`)
Cards grouped Pending / Approved (with "Open chat") / Not selected (generic message).

### 3.11 Marketplace (`Marketplace`)
- Header "Marketplace", "+ Post". Tabs **For sale / Wanted / Yours**.
- For-sale notes: taped, tilted yellow sticky with a framed photo (150px), title, price, `distance · @seller · time left`, Reserved badge when applicable.
- Items without photos use a compact note.
- Wanted tab: same pattern (pink), not mocked yet.

### 3.12 Network (`Network`)
- Header: hamburger, "Network", "n people".
- Intro as a small taped sticky: "Everyone you've done a gig or a sale with, and the rating you gave each other. Tap someone to see what others say."
- Graph (340×440): concentric pale-blue rings, "You" at the centre, connections placed outward. Lines are 2px with a 4px white casing and round caps. Line colour = average rating between the two people (both ways) using the rating colours. Small rating pill (36×16, 10px text, `★ 3.5`) on each line's midpoint. Node = 34px circle with initial and a 4px ring in the rating colour. Outer rings show who each connection has worked with.
- Legend: Strong 4.5+ / Fair 3–4.4 / Weak <3. Caption below.
- Tap a person → profile with what others say (not mocked yet).

---

## 4. Implementation notes
- Build the sticky-note as one reusable component with props: `color`, `tilt`, `tape`, `fold`.
- Treat tokens in §1 as CSS variables or theme constants.
- Keep minimal motion. Respect `prefers-reduced-motion` for any GIF.
- Touch targets ≥44px. Text on `accent` uses `on-accent` only.
- Don't use bottom nav, filter chips or the "Need something?" prompt. They were deliberately removed.

## 5. Open items (decide before building)
1. "Review" button on My stuff: where does it go now that Applicants is removed? Likely the chat for that applicant.
2. Gig detail and Request detail screens (Apply flow) are not mocked.
3. Chat states: Approved and Declined.
4. Wanted tab/rows: show budget or distance?
5. Marketplace: search or category filter?
6. Seller name tap → profile?
7. Photo tap → full-screen viewer?
8. Network person tap → profile screen.
9. Legend tap on Home: filter behaviour as assumed in 3.1.
