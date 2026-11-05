# 0070. `@mmerp/ui` props: one name per concept (`tone`, `open`, named sizes)

Status: accepted · 2026-11-05 · builds on [ADR-0037](./0037-own-ui-kit-on-base-ui.md)

## Context

The kit's primitives were written over several weeks, some ported from Mantine's API and some shaped after Base UI's, so the same idea has several names:

- Colour is `color` on `Button`, `ActionIcon`, `Badge`, `ThemeIcon`, `Indicator`, `Timeline`, `Alert`, `Toast`, `Avatar` and `Menu.Item`, and `tone` on `Text` and `Anchor`. Its values come from five unions: `ButtonColor`, `ColorName`, the inline union of `Alert`, `ToastColor`, and `Tone`, which also mixes text-only values (`default`, `dimmed`) with meaning.
- Open state is `opened` on `Dialog`, `Drawer` and `Burger`, `defaultOpened` on `NavLink`, and `open` on `Popover` and `Menu`. `Dialog` and `Drawer` close through `onClose`; Base UI parts report `onOpenChange`.
- A change is `onValueChange` on `Tabs` and `onChange` everywhere else. Most composite inputs (`Select`, `Combobox`, `MultiSelect`, `NumberInput`, `FileInput`, `DateInput`, `MonthPickerInput`, `SegmentedControl`) are controlled only.
- Size is a name on `Button`, `ActionIcon`, `Loader` and `Dialog`, but raw pixels on `Avatar` and `ThemeIcon`, and `Dialog` and `Drawer` accept a number too. Spacing is `gap` on `Stack`, `Group` and `Grid`, but `spacing` on `SimpleGrid` and `List`.

Every new screen has to look up which spelling a component uses, agents guess wrong, and wrappers in `shared/ui` exist only to translate one spelling to another (`StatusBadge` renames tones to colours).

## Decision

- **Colour is `tone`, everywhere, and only for meaning.** One union, `Tone`: `neutral`, `primary`, `success`, `warning`, `danger`, `info`, `discovery`. `gray` becomes `neutral`. A component that offers fewer tones types its prop as a subset of `Tone` (a toast has no `neutral`), never as a union of its own. `Text` and `Anchor` take `Tone | 'dimmed'`; omitting `tone` gives the main text colour, so `default` goes. `Menu.Item` takes `tone="danger"`. `Avatar`'s colour carries no meaning, so it is not a tone: it is `accent`, from the decorative palette.
- **Open state follows Base UI and React: `open`, `defaultOpen`, `onOpenChange(open)`.** This covers `Dialog`, `Drawer`, `Burger`, `NavLink`, `Popover` and `Menu`. `Dialog` and `Drawer` drop `opened` and `onClose`.
- **Every input takes `value`, `defaultValue` and `onChange`**, so each works controlled or uncontrolled. Inputs that are a native element (`TextInput`, `Textarea`, `PasswordInput`) keep the native `onChange(event)`; every other input calls `onChange(value)` (a checkbox's value is its checked state). `Tabs` moves from `onValueChange` to `onChange`.
- **Size is a name, never pixels.** Each component offers named steps (`xs`, `sm`, `md`, `lg`, `xl`, only those it needs); the pixel value of each step lives in the tokens. `Avatar`, `ThemeIcon`, `Dialog` and `Drawer` stop taking numbers. Pixel sizes in use today map to the nearest step; near-duplicates (avatars at 24 and 26px) merge.
- **Spacing between children is `gap`**: `SimpleGrid` and `List` drop `spacing`.
- **`shared/ui` components that wrap one of these concepts use the same names** (`FormModal` takes `open` and `onOpenChange`; `StatusBadge` takes `tone` and passes it through).
- **Renames land without aliases.** The kit is private and every caller is in this repository, so each family (colour, open state, inputs, size) is renamed in the kit and in every caller in the same PR; there is no deprecation period.

## Consequences

- A prop name says what it is for: `tone` always means meaning, a size is always a step on a scale, and open state reads the same on every overlay. The lookup per component goes away, and `/dev/ui`, Storybook and the `mmerp-ui` skill describe one spelling.
- `StatusBadge`'s tone-to-colour mapping and similar renaming code in `shared/ui` disappear.
- Uncontrolled inputs become possible, which simple filters and settings forms can use without local state.
- The renames touch most screens once. Each family is its own PR, kept mechanical, so a reviewer checks the kit change and skims the callers; type errors find every caller, so none is missed.
- Some sizes change by a few pixels where near-duplicate steps merge; the PR that renames sizes lists them.
- `docs/ui.md` and the `mmerp-ui` skill change with each PR, since they quote these props.
