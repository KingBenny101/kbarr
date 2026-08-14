# Episodes ledger — design doc

Date: 2026-08-15
Status: Approved (user)
Branch: revamp/episodes-table-pc-view

## Subject

kbarr is a self-hosted anime library manager. The Media Detail page shows one
season of an anime and its episodes. The episodes table is the working
instrument of the page: it answers "which episodes are out, which do I have,
which am I tracking" and lets the user flip monitoring per episode.

The page's single job: let the user scan a season's broadcast ledger at a
glance and act on it (monitor / unmonitor, open the source link).

## Direction

The current table is a generic Mantine data grid (striped, bordered,
8 columns, all cells centered). The revamp treats the table as a
**broadcast-style ledger**: the episode index is the hero of each row, and
availability is readable down the gold index spine like a TV schedule.

## Token system

### Color

No new hues — the gold identity stays.

| Token | Hex / value | Use |
|---|---|---|
| Gold (lit) | `--mantine-color-yellow-5` | Available episode index |
| Off-air | dimmed/gray | Unavailable episode index, unused data |
| Pills | existing tones | green = available/monitored, gray = not, violet/green/red = Special/Credit/Parody |
| Surface | existing default surface | Rows sit on the page surface, one rounded container |

### Type

- Body: Inter (existing brand face).
- Data: the theme's mono stack — EP index, air date, quality. Tabular figures,
  machine-print feel.
- Header labels: tiny uppercase dimmed micro-labels (size xs, letter-spaced),
  replacing the bold centered header cells.

### Layout

- Kill stripes, vertical column borders, and the full-grid border.
- One rounded container; hairline divider between rows only.
- Fixed EP-index spine on the left (gold when available, gray when off-air).
- Title is the flexible column, single-line ellipsis.
- Link icon appears on row hover.
- PC: table layout. Mobile: existing cards re-skinned with the same index
  block and air date, keeping the pill language.

### Signature

The **EP index block**: fixed-width cell with a tiny mono "EP" label over a
large gold (available) / gray (unavailable) number. Availability scans down
the spine. The deliberate risk: the episode number — not the title — is the
hero of each row, because in anime tracking "what episode is out" is the
user's mental model.

## PC wireframe

```
┌──────────────────────────────────────────────────────────────────┐
│ NO.      TITLE                          AIR DATE   STATUS   …    │
├──────────────────────────────────────────────────────────────────┤
│ EP01     The Beginning                  2026-04-06  [Available]  │
│ EP02     A Turning Point                2026-04-13  [Unavailable]│
│ EP03     (unavailable: index gray,      —           [Unavailable]│
│          title dimmed)                                          │
│ S1       Special — OVA [Special]        2026-05-01  [Available]  │
│ …                                                                 │
└──────────────────────────────────────────────────────────────────┘
```

Columns (PC): No. (spine) | Title | Air date | Availability | Quality |
Subtitles | Monitor | Link (hover). Type pill appears only for
Special/Credit/Trailer/Parody.

## Mobile sketch

Same tokens, card form:

```
┌────────────────────────────────┐
│ EP01  The Beginning      [Reg.]│   ← index block left, pill right
│ 2026-04-06   [Available]       │
│ 1080p · ja        [Monitored]  │   ← monitor pill + link
└────────────────────────────────┘
```

## Quality floor

- Responsive: mobile cards + PC ledger.
- Keyboard focus: monitor pill and link remain focusable; hover-only reveal
  still works via focus-within.
- prefers-reduced-motion: no added motion beyond what already exists.
- Dark/light: both schemes, since the app follows the system scheme.

## Constraints

- Keep the "Monitored" / "Not monitored" pill labels — the existing
  MediaDetailPage tests select them by text.
- Air date comes from `Episode.air_date` (already returned by the API,
  currently unused). Format as a short date (e.g. `2026-04-06` or
  `Apr 6, 2026`).
- Sorting (ep_no / title), pagination, and the type filter chips stay as is.
