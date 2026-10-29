## Automatic errors native acceptance — 2026-10-29 08:34 UTC · Codex

The user explicitly approved a temporary synthetic-error button after the Inspector limitation. Only the existing disposable candidate's App.tsx gained the button and a fixed result label observing the original Wails promise. The button schedules a synthetic TypeError through setTimeout; it does not call the reporter directly. The production capture module, pinned SDK, generated bindings, native allowlist/consent/transport, service and database were unchanged. The temporary App used normal `production` tags and the existing seven synthetic Skills, not the user's documents.

**Confirmed native delivery:** actual mouse activation while consent was off produced no native send result and no new row (error 6 / feedback 4). After mouse opt-in, the same button produced native success ID `52a0c4e52e39090fe9dfc4cafe100b8b`. Read-only SQLite lookup confirms exactly one new project-1 error with that ID, environment `synthetic`, type `TypeError`, version 0.1.10, fixed message `Automatic error report; message omitted.`, and `frontend.js:43:112645`. The synthetic private-message marker, original function names and URLs are absent. Counts became error 7 / feedback 4. This uses the actual WebKit global error listener → isolated Sentry serialization → Wails `SendAutomaticError` → HTTPS service → SQLite chain; it supersedes the earlier pending-native limitation.

**Confirmed disabled behavior:** the switch was turned off by mouse and saved. A fresh process using the same public endpoint cleared all in-memory deduplication/rate state. The identical button was then clicked; no native send callback occurred, reading continued and SQLite counts remained 7 / 4. This result cannot be attributed to the previous fingerprint or one-minute throttle.

**Confirmed connection-failure isolation:** only the isolated process DSN was changed to loopback port 1, separately verified to refuse connections (errno 61). With consent enabled, mouse activation reached the real native sender and returned failure for ID `b8f8501e87acd534ecd7e38061e87d4b`. The app immediately remained able to open another synthetic Markdown resource, open Settings, disable reporting and return to that resource; no main-UI error dialog or content loss occurred. The failed ID has zero rows and the real service counts remain 7 / 4. The real service, Tunnel and network settings were not stopped or modified. This pass submitted exactly one public synthetic event; the earlier direct-SDK synthetic event remains separately documented.

**Cleanup:** PID 20735 exited, removing its process-only failure endpoint. The isolated consent preference was left false. Exact temporary button/status edits were removed, all 15 feature source paths again matched the repository, and the generated Synthetic/Inspector App bundles were removed. TypeScript/Vite, normal ARM64 production rebuild and strict ad-hoc signature verification passed. Searches of the clean source, generated JS and executable found none of the temporary button, status or private-message markers. Clean executable SHA-256 `ed60315a926add0a7f2542d259d489df071313ad7e6db4809c74aef5a99b7979` is byte-identical to the original pre-trigger candidate. No temporary trigger enters the formal source/build. No commit, push, release, deployment, administrator-secret access, paid API call or installed-App replacement occurred.

The already completed native on/off/restart settings acceptance and earlier Go/race, 18×498 locale, SDK/privacy checks remain valid and were not repeated. This focused native pass uses a TypeError and connection refusal; Promise rejection native pointer triggering, DNS/TLS/timeout/429/5xx native UI combinations, ten distinct errors across ten minutes, other machines, native crashes and symbolication were not newly exercised. Relevant shared-transport failure tests cover those HTTP/network classifications. Translation/network providers were not called. Pure UI logic bugs with no exception remain outside automatic capture.

## Automatic errors native follow-up — 2026-10-29 08:11 UTC · Codex

The user unlocked the Mac. ExternalSSD and the native desktop were available. The existing production candidate `tmp/automatic-errors-lvfmaa5v/bin/SkillView.app` was reused with the existing seven synthetic Skills; no previously passed check or production build was rerun. Native mouse acceptance confirmed the explained checkbox defaults off, enabling writes `automaticErrors: true`, a real process restart restores checked state, disabling writes false, and another restart restores unchecked state. Reading the synthetic code-review Skill remained normal. This verifies the actual Settings → Wails → preferences path, not just file edits.

A separate generated `SkillView-Inspect.app` used the same frontend and backend feature code to try to trigger a synthetic exception in WebKit. It was built with Wails's `production,devtools,private_mac_apis` tags. Its temporary main-file change only added a developer-console menu; it was restored after compilation. The documented OpenInspectorOnStartup option has no macOS implementation in the installed Wails 3.0.0 source. Both the standard developer role and a menu targeting the known window showed no operable Inspector after activation. Bounded attempts stopped without changing OS permissions, Safari settings, SDK sources or production app configuration. This is an Inspector/UI-access blocker, not a lockscreen or pending approval. The Inspector build is a disposable debugging artifact and must never be published.

No controlled native exception was triggered in this pass. Therefore complete WebKit event → native `SendAutomaticError` → public service delivery, disabling preventing that native send, and a failed automatic request while reading remain **unverified**. Existing shared transport tests and the previous direct production-listener/SDK/server acceptance remain valid but are not substitutes for these native checks. Read-only SQLite counts remain error 6 / feedback 4: zero new submissions this pass.

One UI observation immediately after Command-Q caused the computer-use tool to relaunch the candidate without fixture environment, displaying the regular catalog. No interaction was performed on that catalog; the exact candidate process was stopped and subsequent restarts used explicit synthetic paths. The regular preferences' automaticErrors value was confirmed false and no record was transmitted. Later restart/cleanup checks used exact process paths and did not ask the UI tool to reobserve a quit App.

Cleanup confirmed the last Inspector PID 17344 exited, the isolated consent setting is false, all 15 candidate feature paths match working-tree sources, and temporary main.go matches HEAD again. The original production executable SHA-256 remains `ed60315a926add0a7f2542d259d489df071313ad7e6db4809c74aef5a99b7979`. Only existing TODO/feedback/verification records changed this pass. No commit, push, release, service deployment or installed-App replacement occurred.

## Automatic errors (local) — 2026-10-29 · Codex

Implemented against published 0.1.10 (`e52f2501f3dcec03ddb7c2d656f49924f480b8bc`), without committing, publishing or replacing the installed App. Persistent consent defaults to false for new/old settings, with an explained checkbox and two new labels translated across all 18 locales. Window-lifetime listeners survive a React unmount and are removed on disable/page hide. Scope is uncaught frontend errors and unhandled Promise rejections only; no automatic logs, activity collection or original messages. Native consent, destination, exact envelope allowlist, shared single-flight transport, one-minute/ten-error process limits and three-second cancellation are implemented.

Existing `go test -race ./internal/app ./internal/library ./internal/locale` passed app/library; locale initially rejected the intentional key-count change. Updating its existing exact coverage count from 496 to 498 and rerunning `go test ./internal/locale` passed all locale coverage/placeholder checks. Four existing feedback-SDK/localization frontend tests passed. Generated bindings, TypeScript/Vite and an isolated ARM64 native build with strict ad-hoc signature passed. The final isolated build includes the listener-lifetime correction; no full-file formatting, new test framework or test files were added. Existing shared native-transport tests cover DNS/refusal/timeout, 429/5xx, redirects, malformed receipts, cancellation and manual-feedback privacy; they do not directly exercise the new automatic method's consent/rate-limit branches.

Direct execution of the production sanitizer/serializer with synthetic Error stacks omitted the sample private message, function names, local path and queried URL, retaining only allowlisted `frontend.js` line/column positions. A string rejection became `NonErrorRejection` without its value; reporter-chunk errors were excluded. Direct window EventTarget execution of the bundled production listener produced one send attempt for a burst of 12 errors; a rejected sender did not retry, and stopping the listener produced no further attempt. This was a bounded in-memory operation, not a browser/native UI check. The full ten-distinct-error/time-window limit was reviewed in source, not exercised across ten minutes.

One real synthetic `unhandledrejection` passed through the production listener and pinned Sentry serializer, then a direct HTTPS sender to the existing project. Receipt: HTTP 200, accepted 1, duplicates 0, skipped empty. Read-only SQLite lookup confirms event `cd3f2cc494c05d2b25b89b507f962e59`, kind `error`, environment `synthetic`, type `TypeError`, fixed message `Automatic error report; message omitted.`, and only `frontend.js:12:34`. Counts changed from error 5 / feedback 4 to error 6 / feedback 4. No other event was submitted. This confirms SDK/server compatibility and actual storage; the direct HTTPS sender did not exercise Wails `SendAutomaticError`. No administrator credentials, real documents, account content or paid AI were used.

Native candidate: `tmp/automatic-errors-lvfmaa5v/bin/SkillView.app`, using existing synthetic data and an unreachable loopback DSN, never the installed App or production data. Computer-use inspection explicitly reported the Mac locked. Native checkbox pointer/keyboard behavior, persisted choice after restart, and complete WebKit → Wails → native sender → server acceptance remain **unverified**. No unlock bypass was attempted. The exact isolated candidate PID 9503 was stopped and its exit confirmed; no other process was stopped. All 17 pre-existing non-record paths retain their original content (README only gains the scoped privacy-sentence correction); prior TODO/verification records remain intact, the Git index is empty and HEAD is unchanged. Production auto-reporting remains off in the installed 0.1.10 App. Pure UI bugs without an exception, handled errors, startup failures before consent loads, Go/native crashes and symbolication are outside scope. The new translations are not mother-tongue reviewed.

## Published 0.1.10 — 2026-10-29 · Codex

[Release v0.1.10](https://github.com/sogud/skillview/releases/tag/v0.1.10) was published at 06:38:39 UTC from `e52f2501f3dcec03ddb7c2d656f49924f480b8bc`. The annotated tag resolves to that commit. [Build CI 40829552036](https://github.com/sogud/skillview/actions/runs/40829552036) and [website CI 40830451540](https://github.com/sogud/skillview/actions/runs/40830451540) succeeded. Scope is the single menu blur-condition fix, release version, TODO/verification and existing website version text only; 17 unrelated paths remain unchanged.

Focused updater race tests, five existing frontend checks, generated bindings, TypeScript/Vite, ARM64 ZIP/DMG build, strict signature and package integrity passed locally. The release ZIP was extracted and its exact version, architecture and strict code signature checked. Its frontend bundle is the same one used for the native pointer acceptance below; only the release version changed. The initial checksum invocation used the wrong working directory; rerunning in the asset directory verified both packages.

All four unauthenticated public downloads returned HTTP 200 and matched the signed local and downloaded draft bytes. Manifest version/platform/URL/size/SHA-512 and pinned Ed25519ph verification passed. ZIP SHA-256: `1e4959b2e7567359c02b2e476b6a0a3671f2b97ecf1e05918883008156e2f14a` (10,340,293 bytes). Public website HTML/JS match the candidate and show 0.1.10; existing images/layout were not changed. [Full release evidence](../tmp/release-0110-w4fokswz/release-verification.json).

The user's installed App and previously published assets were not replaced. A complete 0.1.9 → 0.1.10 native replacement/relaunch was not rerun because the desktop relocked after the completed menu acceptance; the previous upgrade acceptance does not claim this new path. Other focused native limits remain listed below. Signing remains ad-hoc, without Apple notarization. This post-publication record remains local and does not rewrite the released commit or assets.

## More-menu local fix — 2026-10-29 · Codex

After the user unlocked the Mac, the exact published 0.1.9 executable (SHA-256 `d21a0835ccc7b1a88bff404853e739fc42f085b27d995f3cd68cccbbf01d8dd0`) was run with existing isolated synthetic data. Actual coordinate clicks opened the dropdown, but Settings/Feedback clicks on the Skills list and Settings on the reader closed it without opening a dialog. Keyboard End/Return opened Settings. The plugin page's Settings click succeeded; its initially focused refresh item is a React portal, with different React blur propagation. This is a menu-item activation failure, not hidden or clipped dialogs. Earlier keyboard checks missed the native mouse failure.

The controlled native comparison isolates the faulty branch in App.tsx: WebKit's blur with a null relatedTarget closed the menu before the button click. Only the blur condition changed, closing for a known focus destination outside the toolbar. Existing outside-pointer dismissal, click handlers and keyboard controls remain intact. The handler already exists in 0.1.8; the 0.1.9 Back-button change did not introduce it. No event instrumentation or new test framework was added.

The isolated production build from HEAD plus this single source patch passed generated bindings, TypeScript/Vite, ARM64 native build and strict ad-hoc signature verification. Existing large-chunk/duplicate-libobjc warnings remain. Actual 900×560 macOS pointer checks passed Settings, Feedback and Archives opening, closing and reopening; Skills/reader/plugin-page menus; Skills and plugin refresh entries; returning from the reader and opening Settings again. Trigger toggling and outside clicks close the dropdown, Escape restores the More trigger, Command-F/Shift-Tab moving focus outside closes it, and End/Return still activates Settings. Dialog closure restores focus to More. An unsaved synthetic draft survives Settings/Feedback open/close and Continue Editing after Back; undo restores the original text, then Back and Settings work. Feedback was only opened/closed, with its DSN empty and submission disabled. No translation or real document transmission occurred.

Native proof: [mouse-opened Settings](../tmp/menu-fix-e9k0j2wz/native-settings.jpg), [mouse-opened Feedback](../tmp/menu-fix-e9k0j2wz/native-feedback.jpg). The installed App and all 17 unrelated worktree paths were preserved. Both original and fixed isolated App processes exited, confirmed by process inspection. The final optional Finder-window cleanup call stalled and reported the Mac locked again; it was not retried. No new commit, push, tag, release, deployment or installed-App replacement occurred.

This focused fix is locally verified. The user authorized publishing 0.1.10 at 06:23 UTC; completed publication and asset checks are recorded above. Native plugin-detail content, an update-available menu entry (absent in the fixture), other window sizes/platforms, and prior broader release limitations were not re-tested. Earlier broad menu acceptance statements do not supersede this actual pointer evidence.

## Published 0.1.9 — 2026-10-29 · Codex

[Release v0.1.9](https://github.com/sogud/skillview/releases/tag/v0.1.9) was published at 04:27:50 UTC from `c881d15646df6d1f50e6237791287be8c4ac2e91`. [Build CI 40817209216](https://github.com/sogud/skillview/actions/runs/40817209216) and [website CI 40817886287](https://github.com/sogud/skillview/actions/runs/40817886287) succeeded for that commit. Only the nine scoped frontend/version/documentation/website-copy paths were committed; all 17 unrelated paths remain unchanged.

All four public assets returned HTTP 200 and matched the signed local files. Manifest version, platform, URL, size, SHA-512 and pinned Ed25519ph verification passed. ARM64 ZIP SHA-256: `96f5bac7988d2d571939270ea2aa687764dcde15261cd9dcd426e2b942458037` (10,340,248 bytes). The [website](https://sogud.github.io/skillview/) retains the approved imagery and displays 0.1.9; all five public website files match the candidate. The initial website check located Chinese text in the English HTML and was corrected to check the existing bilingual script; deployed bytes were correct.

The original published 0.1.8 in an isolated App/data path discovered and verified 0.1.9 from public GitHub. An unsaved synthetic draft blocked restart; Continue Editing retained it. Saving and updating then replaced/relaunched the isolated copy, preserving every preference field and the saved draft. Strict codesign passed, and the new executable SHA-256 `d21a0835ccc7b1a88bff404853e739fc42f085b27d995f3cd68cccbbf01d8dd0` matches the release ZIP. The new native window reopened its seven synthetic Skills. The installed App was untouched. Candidate/upgrade Apps and Vite exited, browser tab closed and viewport reset. [Release evidence](../tmp/release-019-o8rau6uj/release-verification.json).

Post-publication TODO/verification updates remain local and do not rewrite the released commit or assets. Limits remain those recorded below: ad-hoc/no notarization, no separate native plugin-detail acceptance or deliberately prolonged save-state observation, and previous unverified platform cases.

## Titlebar return control (0.1.9) — 2026-10-29 · Codex

Skills and plugin details now put their existing Back control in the titlebar search slot, beside the traffic-light reserve. The extra reader navigation row is removed; query state and existing draft protection are retained. App/Plugins formatting was restored from their current text only, using the original quote/semicolon style with canonical-format equality and concurrent-content checks. No HEAD-based file replacement occurred.

Actual synthetic browser checks passed German at 760px and 420px, Arabic dark RTL at 420px, Skills/Plugins switching with retained detail selection and exactly one corresponding Back button, Enter/Escape return and restored list search. The reader starts at y=54px with no old navigation row; long labels truncate with full titles and no header/page overlap. Actual native 900×560 candidate inspection and a coordinate click verify the moved control is visible and clickable in the titlebar. Unsaved synthetic edits show the existing confirmation; Continue Editing retains the draft, Command-S saves it, and Command-F returns to focused search. Native plugin detail layout is supported by the shared titlebar implementation and browser checks; this pass did not open a native plugin detail. Saving-disabled wiring and request(saving) guard were reviewed; the short native save was not artificially delayed to observe that transient state.

Focused existing updater race tests and five frontend file-tree/group tests passed. Generated bindings, TypeScript/Vite, ARM64 ZIP/DMG packaging and strict ad-hoc signature passed; generation left tracked candidate sources unchanged. Existing large-chunk/duplicate-libobjc warnings remain. No new test framework or fixtures were written; existing synthetic data tools/runtime were reused. The installed App and unrelated CLI/workflow changes were preserved. Public asset, exact-commit CI and upgrade verification are recorded after publication. [Native proof](../tmp/release-019-o8rau6uj/native-back.jpg).

## Published 0.1.8 and public website — 2026-10-28

[Release v0.1.8](https://github.com/sogud/skillview/releases/tag/v0.1.8) was published at 15:53:06 UTC. Remote main and the annotated tag resolve to `3094ad1e24932b8af7e58c6a018871dffee06f82` (tree `b28213ec0de960cc7f9f549a83fe164239cc8c56`). [Build CI 40717704630](https://github.com/sogud/skillview/actions/runs/40717704630) and [website CI 40720459703](https://github.com/sogud/skillview/actions/runs/40720459703) succeeded for that exact commit. The earlier paused/superseded candidates below are historical; only the final approved candidate was published.

Four public assets—ARM64 ZIP, SHA-256 file, BUILD_INFO and signed manifest—returned HTTP 200 without authentication and matched the uploaded bytes. Independent pinned-key Ed25519ph, manifest version/platform/URL/size and SHA-512 checks passed. ZIP SHA-256: `109960c587be1c4afda963150af7a5d29a72107e9d8493a53b60912e27b6b86e` (10,340,352 bytes). The ZIP contains the canonical App and notices, with no private signing key. The DMG was locally validated but is not a release asset.

An unmodified published 0.1.7 App, extracted into the disposable candidate directory with isolated data and synthetic source only, checked the public feed and downloaded/verified 0.1.8. Restart was blocked by its unsaved synthetic draft. Continue Editing retained the draft; saving and retrying restarted successfully as 0.1.8. All preference fields and the saved draft were retained. The installed test executable SHA-256 is `5fc8998ec432115f5907147cc886d2fb250c1c708b9ee69ff223a9b10f002748`, byte-equal to the final release, and strict codesign verification passed. The final native toolbar rendered after relaunch. Only the disposable copy was replaced; the user's installed App was untouched. The discard branch and other update environments are not covered.

The [live website](https://sogud.github.io/skillview/) loads the approved [native screenshot](https://sogud.github.io/skillview/assets/reader.jpg?v=0.1.8) at 899×559 and the generated book icon at 256×256. Public HTML, JS, CSS and both images match the candidate bytes. Screenshot SHA-256: `0d5c7813ee83b25062bb33b89b02992bbf0c5326a2de4870c57284f7122dfc35`. Actual browser inspection confirmed Chinese/English 0.1.8 labels, no duplicate decorative titlebar, loaded images and no 390px horizontal overflow. Only synthetic Skills appear. [Live-page screenshot](../tmp/release-018-final-jbz6d1rh/skillview-018-live-website-full.jpg).

Temporary native App, local website server and browser preview were cleaned up, and the viewport override reset. All 17 unrelated worktree paths remain unchanged. Detailed ignored evidence: [release record](../tmp/release-018-final-jbz6d1rh/release-verification.json), [public file comparisons](../tmp/release-018-final-jbz6d1rh/public-verification.json). This post-publication record is a local documentation update; it does not alter the released commit or immutable assets.

Remaining limits: ad-hoc signing without Apple notarization; initial translations without native-speaker review; no full ICU plural rules. Native dragging, traffic-light actions, Dock appearance, other scaling/macOS configurations and the Settings accelerator remain unverified. Previously accepted native Command-F and translation mode save/restart/reopen remain valid. More-menu pointer acceptance for the published 0.1.8/0.1.9 builds is withdrawn; the local fix is verified above.

## Approved 0.1.8 final candidate and website — 2026-10-28

The user approved the final native toolbar preview and reauthorized release and the existing website screenshot update. A fresh isolated source export excludes unrelated CLI/workflow changes; only main.go's native-window hunk belongs to this release. The approved native capture replaces website/assets/reader.jpg, the existing brand icon uses the generated book icon, and the decorative website titlebar is removed so real traffic lights appear once. Image dimensions/alt/caption and displayed version are synchronized; other website copy remains unchanged. Publication checks and exact-commit CI are recorded separately; earlier paused candidates below remain historical and must not be reused.

Final candidate validation passed all ten Go/race packages, focused vet, nine frontend tests, 18 × 496 locale key/placeholder coverage, generated Wails bindings, TypeScript/Vite, fresh ARM64 ZIP/DMG packaging, strict ad-hoc signing, checksums and hdiutil integrity. Runtime govulncheck and production npm audit found no known vulnerabilities. Initial system-temp exhaustion was resolved with TMPDIR/GOTMPDIR in the existing external workspace tmp; the library test's absolute /projects/ assertion also required that neutral temporary parent. Tests and assertions were not modified. Local website checks confirmed the native screenshot (899×559), matching icon, no duplicate decorative titlebar, English/Chinese version 0.1.8, and no overflow at desktop or 390px width. Exact-commit CI and public assets are subsequent publication gates.

## Native toolbar correction; publication paused — 2026-10-28

The user rejected the actual native toolbar before any commit/push/tag/release. Screenshot `libfile_fcf1d4dbe4248191b3f087d47464e5ef` was materialized through the current Library helper into existing tmp and its pixels viewed. Wails source confirms that InvisibleTitleBarHeight controls the drag area, not traffic-light placement; HiddenInset creates an NSToolbar with automatic style. Browser horizontal-center measurements alone did not accept this native geometry, and supplied screenshot pixels were not treated as CSS pixels.

Final local changes: explicit MacTitleBarHiddenInsetUnified; matching 54px web header and drag band; necessary traffic-light reserve only on the left; right edge 12px; search capped at 260px. Equal grid tracks keep tabs centered against the whole window without repeating the left inset on the right. Only New Skill and More stay visible in the Skill list; Plugins retain their read-only scope with More. Secondary refresh/archive/update/feedback/settings actions share one keyboard menu. The tray-arrow action is Archived Skills, not import; resource importing stays in the reader. Action icons are 16px with 1.75 stroke inside 28px controls. No palette, reading typography or artwork changes.

Actual native 900×560 window capture (899×559 image after window-edge cropping) after the final explicit-Unified rebuild shows traffic lights and web controls on the same visual centerline. Native startup/empty state, scanning seven synthetic Skills, normal list, menu keyboard navigation to Settings, close-to-trigger focus and real Command-F passed. Translation-only was saved, the isolated App quit and restarted, and reopened native settings still showed translation-only; its isolated JSON agreed. No provider request or real document was sent. Native resize through the UI tool had no observable effect and is not passed. Native dragging, traffic-light actions, actual Dock appearance, other scale/macOS configurations and the Settings accelerator remain unverified.

Browser synthetic checks passed Chinese 1000px, German 760/420px and Arabic dark RTL 760/420/1440px. Header CSS center y=27px, tabs center x=width/2, rightmost control at width−12px, no horizontal overflow or header overlap. The 420px stress case is below native minimum and compresses search. Long menus fit; rescan, archives, feedback, New Skill, Settings and update entry points work; plugin refresh retains query. ArrowDown/ArrowUp wrapping, Home/End, Enter, Escape, outside click and dialog/trigger focus restoration passed.

Nine frontend tests, TypeScript/production frontend build, generated bindings, isolated ARM64 native build and strict ad-hoc signature passed. Final native style-only rebuild and signature passed. Existing large-chunk/duplicate-libobjc warnings remain. The 17 unrelated paths are preserved, with main.go's pre-existing CLI hunk unchanged beside this task's window hunk. Real Git index is empty, HEAD remains v0.1.7 cbd7fe739b7223e6326b2784a1403e82b42a61be. Vite and isolated App processes, browser tab and viewport override are cleaned up. No installed App replacement or unlock bypass.

Final actual native preview: [toolbar-native-corrected.png](../tmp/toolbar-native-corrected.png), Library libfile_6567d32abbd08191b56536f211065096, version 2, file file_0000000073e4823085bf6ed344d067fd, 144,959 bytes (PNG encoding of the unchanged native capture). Publication remains paused for user visual approval; no tool approval is pending. Old 0.1.8 ZIP/DMG files are explicitly marked superseded in the ignored acceptance directory and must not be published.

## Superseded 0.1.8 release candidate (not published) — 2026-10-28

The release candidate combines the compact plugin list, persisted immersive/translation-only display preference, 18 × 496 locale keys, literary book icon, restrained near-white surface and one truly window-centered toolbar. The rejected sage/ivory redesign is excluded. A separate index/export includes only this scope and leaves 17 unrelated CLI/workflow/rules paths byte-for-byte unchanged.

Fresh candidate checks passed: all ten Go packages with race detection, focused vet, all nine frontend tests, generated Wails bindings, TypeScript/Vite, ARM64 native build and ZIP/DMG packaging. All tracked exported files still matched the reviewed source index after generation. Strict ad-hoc codesign, version/localization plist, package SHA-256 and hdiutil verification passed. Runtime govulncheck and production npm audit found no known vulnerabilities at this check. The existing frontend chunk-size warning remains.

An isolated copy with a separate bundle identifier and synthetic source/settings launched successfully. Native accessibility returned the Skills window, search, Skills/Plugins tabs, all five actions and empty catalog. That call took approximately 23 minutes, so it does not count as pointer, drag, traffic-light, shortcut, setting-restart or actual Dock acceptance. These remain documented limitations, supported separately by earlier browser geometry/interactions and backend persistence tests. No unlock bypass or installed-App replacement occurred. Only this test App's confirmed PID was stopped.

Release signing, exact-commit CI and remote assets are verified as separate publication gates; the detailed historical checks below retain their original dates and scope.

## Unpublished toolbar symmetry refinement — 2026-10-28

Only `frontend/src/styles.css` changed for this refinement. The original asymmetric 88px/12px padding and unequal column minima shifted the tabs 38px right. Both outer reserves are now 88px and both side tracks are `minmax(0, 1fr)`, so the center is relative to the whole window. The action container fills its equal-width track; only its controls are non-draggable, leaving unused background draggable. Physical-right alignment remains correct in RTL. The disclosure breakpoint is now 800px to keep longer translations and the optional update action clear of the center. Colors, icons, typography, translations and application behavior are unchanged.

Actual browser geometry and interaction evidence (retained from this run, not inferred from the build):

| Viewport / locale | Tabs center | Center error | Left / right region widths | Result |
| --- | ---: | ---: | --- | --- |
| 1000px / Chinese Skills and Plugins | 500px | 0px | 335 / 335px | Action count change does not move center |
| 1440px / Russian + update | 720px | 0px | 542.117 / 542.117px | Six actions fit |
| 801px / Russian + update | 400.5px | 0px | 222.617 / 222.617px | Full-action side of breakpoint; no collision |
| 760px / German + update | 379.996px | -0.003906px | 225.438 / 225.445px | Narrow disclosure, search remains wide |
| 420px / German + update | 209.996px | -0.003906px | 55.438 / 55.445px | Stress case, search compresses without overlap |
| 420px / Arabic dark | 209.996px | -0.003906px | 49.750 / 49.758px | RTL, action button ends at x=332 |

All measured views retained the 38px header, symmetric 88px outer reserves, nonnegative 8px/12px side gaps and no page horizontal overflow. Sub-0.01px differences are layout-engine fractional-pixel rounding. German search filtered correctly; keyboard refresh invoked the synthetic Scan once and returned focus to the overflow trigger. The labeled menu and Settings dialog remained bounded, with focus restored after closing. Arabic Skills/Plugins switching, Escape and the fixture's unchanged native-Find event bridge focused plugin search. No actual native accelerator acceptance is claimed. The 420px viewport is below the native 760px minimum; its search field is correspondingly compact.

Production frontend build (`tsc --noEmit` + Vite) and `git diff --check` passed; existing chunk-size warning remains. No JS/Go logic changed, so the prior successful nine frontend tests and 18 × 496 locale checks were not redundantly rerun. Actual final screenshot replaced the existing Library preview, preserving its identity/history: `libfile_fb22d934affc81919ff8ffa23ca30612`, version 1, file `file_00000000af0c81fda7afe5bf314bf6c3`, 38,790 bytes, [toolbar-ui-preview.jpg](../tmp/toolbar-ui-preview.jpg).

The preview tab was closed, viewport reset, and task Vite PID 38777 stopped and confirmed absent. Native traffic-light clicks and dragging remain unverified. No unlock attempt, commit, push, release or installed-App replacement occurred. Earlier plugin layout, translation display modes, icon and unrelated edits were preserved.

## Unpublished single-row library toolbar — 2026-10-28

The supplied screenshot `libfile_522e9512a31c819196dda52d981c03af` was materialized through the current Library helper to `tmp/image(20261028-131455).png` and its actual pixels inspected. It showed view tabs and global actions above the search/action row. The last library button is Archived Skills, not import; its behavior is preserved.

Search, view tabs and contextual/global actions now share the existing 38px titlebar. Skills and Plugins use the same search/action positions; plugin controls are rendered into the titlebar while their existing query/catalog/reader state remains owned by Plugins. The titlebar preserves the physical-left 88px traffic-light reservation and draggable background gaps. Input, tabs and actions remain `no-drag`. The native window configuration (38px hidden inset titlebar, 760px minimum width), theme palette and reading typography are unchanged in this round.

At widths ≤700px the actions move into a labeled disclosure group. Search remains visible, the keyboard hint alone is hidden, and long tab labels can truncate with full title tooltips. All actions remain available; pending updates retain a dot and accessible label on the trigger. The group supports Tab/Enter, Escape, outside-click close and focus restoration. Opening a dialog returns focus to the visible overflow trigger when the dialog closes. No new runtime dependency was added. “More actions” is translated in all 18 dictionaries (496 keys each).

Actual synthetic browser acceptance:

- Chinese 1000×700 normal and empty catalogs: one 38px toolbar; search starts at x=88, middle tabs and right actions share the same vertical center, no second search row. Search filters results. Skill rescan invokes the mock service once; New Skill, Archived Skills, Feedback and Settings open their existing dialogs.
- Russian 760×480 with an update available: all six actions remain visible, no overlaps or horizontal overflow. Search, tabs and actions are contained between x=88 and x=748.
- German 420×600: compressed search and one overflow trigger, long action labels fit. Every action entry, including update, was clicked and the corresponding dialog verified. No actual files were created, feedback sent, update downloaded or real service called. Keyboard Tab/Enter invokes rescan, then returns focus to the trigger; Escape and outside click close the group. Settings close also restores trigger focus.
- Plugin refresh invokes the mock service; search filters to Short and survives switching views. The shared toolbar disappears only in the existing reader presentation. Find returns from both skill and plugin readers and focuses the appropriate search. This uses a fixture bridge for the unchanged native menu events; it is not native macOS accelerator acceptance.
- Arabic dark RTL at 420×600: the header, filters and labeled actions fit; the menu bounds were x=249.5…408 at y=39…211, body width=viewport width=420. The physical-left traffic-light space stays reserved while text and controls follow RTL.

Final TypeScript/production build, all nine existing frontend tests, `go test ./internal/locale` (18 × 496 keys/placeholders and language matching), and `git diff --check` passed. The existing bundle-size warning remains. No Go service/API or generated binding change was needed for this toolbar. Earlier plugin/translation/icon work and unrelated user changes were preserved.

Actual final screenshot: [toolbar-ui-preview.jpg](../tmp/toolbar-ui-preview.jpg), Library `libfile_fb22d934affc81919ff8ffa23ca30612`, file `file_00000000290481fd8a6fae6c9a79a966`, 38,894 bytes. It contains only synthetic acceptance data. This single-row layout supersedes the previous two-row refined preview; its near-white/neutral palette remains unchanged.

The browser preview tab was closed and viewport override reset; the exact task Vite process (PID 35373) was stopped and confirmed absent. Native traffic-light clicks, window dragging, real menu accelerators, settings restart and Dock appearance remain unverified. No native UI or unlock attempt, commit, push, release or installed-App replacement occurred.

## Unpublished restrained visual correction — 2026-10-28

The prior warm/sage redesign was rejected in user review and has been precisely rolled back. The original centered navigation, control shapes, neutral text and blue accents, full-width unfilled search and original dialog backdrop are restored. The added wordmark, browser-header wrapper, separator and 520px search cap were removed. Both the explicit dark theme and system-dark CSS blocks are byte-for-byte equal to the pre-redesign Git baseline. The only light palette difference is the surface changing from `#fff` to near-white `#fdfcf9`. Library top padding is 6px instead of 10px, toolbar bottom gap 4px instead of 6px, and filter bottom padding 6px instead of 10px; plugin search has a matching 16px horizontal margin. No global font shrinking or reader spacing change was introduced.

Long-title ellipsis/tooltips and RTL filter arrow direction are retained as focused accessibility fixes. The previously completed compact plugin badge, both translation display modes and icon changes remain intact. Existing unrelated edits were preserved; no reset, checkout, stash or clean was used.

Actual synthetic browser checks covered the Chinese normal catalog at 1000×700; German long-title truncation and no-search-results state at 420×600; Arabic dark plugins at 420×600, where All has three source badges and Codex has zero redundant badges with 56px rows; and the Arabic translation settings and both options in the bounded dropdown. Document width equalled the 420px viewport, header actions remained inside it, and the long Skill title retained its full tooltip with ellipsis. Actual dark CSS resolved to `--surface: #292929` and `--accent: #8bacf8`. These checks were performed against the production frontend with an isolated synthetic service fixture, not real Skills or a translation API.

TypeScript/production frontend build, all nine existing frontend tests, exact dark-theme baseline assertions and `git diff --check` passed. The existing bundle-size warning remains. This round changes presentation only; prior Go/persistence/translation acceptance is recorded below and was not rerun or relabelled as fresh acceptance.

The rejected [warm preview](../tmp/warm-ui-preview.jpg) and corrected [refined preview](../tmp/refined-ui-preview.jpg) were both actually viewed for comparison. Current screenshot: Library `libfile_01423d4d94a8819183a6127f30fe22af`, file `file_00000000c83081fd802fc5aa3639c356`, `/refined-ui-preview.jpg` (38,712 bytes). It contains only synthetic data. The old Library image `libfile_c61595ae3c90819195bb7245dc44ab29` is superseded, not overwritten. The earlier icon proof remains `libfile_1648faeaf1088191aa07b015cf187ad3`.

The temporary browser tab was closed, its viewport override reset, and the exact task Vite process (PID 31689) stopped and confirmed absent. No native UI or unlock attempt occurred. Native titlebar/dragging, setting restart and Dock appearance are still unverified; an unlocked Mac and a fresh isolated build are needed for that acceptance. No commit, push, release or installed-App replacement occurred.

## Unpublished plugin layout, translation display and icon — 2026-10-28

Plugin rows now keep the source in a compact title badge, hidden when the list is already filtered to that source. The exact supplied screenshot was materialized and viewed. Actual synthetic browser checks covered All/Codex, search, keyboard Enter, long titles/descriptions, English and Arabic RTL at 420×600 and 900×650; rows were 56px with no horizontal overflow.

Translation settings persist `displayMode` as `immersive` or `translation`. Mode-only saves preserve in-memory/disk cache identity and active requests. Actual browser round trips reused one Translate call; reload retained the selection. A delayed request survived a mode change, completed in the latest mode, and a separate explicit cancellation left the original readable. Thirty repeated Markdown fixtures (390 aligned prose blocks), lists/tables/code, plain text and intentionally mismatched block shapes were checked. Full mode showed only translated content; immersive mismatch used the existing complete-document fallback. German and Arabic option menus and settings fit the narrow viewport. No real document or model endpoint was used.

Go/race passed for `internal/translation`, `internal/locale`, `internal/app`; focused vet and all nine existing frontend tests passed. Added focused cases verify cache reuse without a provider request, persisted JSON reload, invalid mode rejection and legacy/unknown stored-value fallback. All 18 dictionaries contain 495 matching keys with matching placeholders. Generated Wails bindings, TypeScript, production frontend and isolated ARM64 App build passed. Existing bundle-size and duplicate-libobjc warnings remain; they did not fail the build.

The icon source from Library `libfile_8406cc13e2488191b052159d7522ee54` is preserved at `build/icon-source.png` (SHA-256 `2d0ceda3ec9048be388f81d2ebaa4028b03d315f755136cd14a6edb663d06756`). Existing AppKit/iconutil packaging adds rounded transparent padding and ten standard sizes without repainting the artwork. The actual light/dark [contact sheet](../tmp/icon-proof.png) was viewed at 16/32/64/128/256px; dimensions, zero-alpha outer corners/padding and opaque centers passed assertions. `tmp/AppIcon.icns` SHA-256 is `a7586721afadda20f81ba7b9b5556b490b0292929e794fa31a485fa4930aa072`. The isolated App contains identical icon bytes, its plist references AppIcon, and its ad-hoc signature verifies.

Native UI acceptance is **blocked, not passed**: the isolated App launched, but the computer-use call eventually returned that the Mac was locked and could not be unlocked automatically. Native save/restart and actual Dock appearance were not observed. `tmp/display-acceptance-path.txt` identifies the disposable fixture and its build log for a later unlocked check. Its process and the localhost Vite process were stopped; browser tabs and viewport overrides were cleaned up. No release, push, commit, service change or installed-App replacement occurred.

## 0.1.7 manual feedback and update dialog

The actual pinned Sentry JS 8.55.0 SDK generated one synthetic AI error and linked feedback; the native transport received commit receipts and subsequent read-only SQLite checks confirmed IDs, release and association. Identical retries did not add records. An isolated native App submitted a locally redacted synthetic report; its visible receipt matched the stored row. A separate slow loopback endpoint caused a bounded error; explicit retry reused the ID, and closing during retry returned to normal document reading. No real service/network configuration or installed App was changed.

Focused tests cover fixed diagnostic/redaction boundaries, rejection of ambient private fields, partial delivery/deduplication, DNS failure, connection refusal, timeout/cancellation, 429 cooldown, 5xx, unauthorized/redirect responses and malformed/oversized receipts. Ten Go/race packages, focused vet, nine frontend tests, 18 × 490 locale key/placeholder checks, bindings and production builds passed. Runtime govulncheck and production npm audit reported no known vulnerabilities at this check. These do not prove absence of defects.

Update-dialog-only layout now has 18px horizontal/bottom footer padding, a fixed header/footer and one scrolling body. Actual synthetic browser rendering was checked at 420×480 for German long text and Arabic RTL, with 18px body/20px button text stress, and at 900×650. Buttons stayed inside the dialog and long notes scrolled without moving the footer. This is a browser layout stress check, not macOS system text-size acceptance. The parent's screenshot description supplied the original symptom; the worker inspected local rendered UI, not that original image.

No automatic behavior/error collection, screenshots, native crash symbolication or private content was sent. The backend has one configured project, no guaranteed uptime and no autostart. Pattern redaction cannot recognize arbitrary private prose. Native-speaker review, all-screen/all-size coverage, OS permission prompts, minimum macOS and Apple notarization remain unverified. Update-dialog changes apply after installing this version.

## 0.1.6 update and Wails migration checks

Wails Go, frontend runtime and CLI are pinned to 3.0.0. The unchanged 18-language dictionary and regenerated bindings passed coverage/placeholder, matching, persistence, TypeScript and frontend tests. Updated runtime Go/race checks cover app, locale, library, translation, network and updates; updater tests cover transient/persistent missing manifests, recovery, unsigned or mismatched metadata, tampered downloads, wrong keys and range resumption.

Public 0.1.5 checks through the old beta.27 provider, Updater and Manager retained the signed manifest. Internal signature loss was not reproduced. The reproducible classification flaw was that a missing manifest became a manual-only release. A missing or stale manifest now retries once with cache revalidation, then fails closed as a retryable check error. A failed recheck clears a stale install offer. Wails 3.0 stages beside the target, removing the old requirement that the App and system temporary directory share a volume.

Wails 3.0 includes the macOS child-frame control-channel fix for [GHSA-vfcr-9mmf-hc6m](https://github.com/wailsapp/wails/security/advisories/GHSA-vfcr-9mmf-hc6m), which affects beta.27. SkillView already skips raw HTML in Markdown and embeds no iframe; this reduces the exposed content path but does not justify retaining the old framework. The package is a desktop build, not Wails server mode. Go 1.27.3 and Vite 8.3.4 remain pinned. See the [official release](https://github.com/wailsapp/wails/releases/tag/v3.0.0) for security fixes and remaining limitations.

Native package and public release evidence is recorded for the exact release separately. Ad-hoc signing is not Apple notarization; minimum-OS, all-screen/all-size, OS permission prompts and native-speaker review remain outside this acceptance. Feedback submission is still unconfigured and disabled.

# Verification

## Interface languages and manual report preparation (0.1.5)

The shared Go/TypeScript dictionary covers 18 interface languages with 490 keys each. Key and placeholder checks, 30 language-tag fixtures, native system-language reading and all-language preference persistence passed. Browser acceptance used synthetic data for Chinese settings, German long text, Arabic RTL with LTR code and paths, file-tree resizing, system-language variants, reload and English fallback. See [localization](localization.md) for exact coverage and editorial limits.

Manual feedback can be reviewed locally with a fixed AI response-format diagnostic, app version and redacted user description. The default release has no feedback endpoint or project key, so submission is disabled. No documents, model responses, screenshots or logs are attached. The service is not deployed; server receipt, search, status and end-to-end delivery are unverified. See [feedback](feedback.md).

New translations have not been reviewed by native speakers. Full ICU plural rules, every screen/size, native permission prompts and complete installed-app upgrade acceptance are not claimed. Release-specific packaging, signature, CI and native smoke evidence is recorded with the release.

## Reader fixes (0.1.4)

Browser interaction with the production React components and synthetic service
responses verifies `./references/guide.md`, returning through `../SKILL.md`,
and refusal of an outside-root link. A 100-file tree has its own scroll area;
keyboard navigation reaches its last entry while the heading and controls stay
visible. The earlier completely unscrollable native symptom was not reproduced.

The same fixture verifies original/translated paragraphs, headings, nested list
items, quotations and table cells, with one unchanged code block. Parser checks
also cover reference links and complete-document fallback for incompatible block
structures. Thirteen path checks cover encoded names, traversal, absolute paths
and malformed input. Existing frontend checks and race-enabled library and
translation checks pass. Credentials and their storage are unchanged; no real
Skill or key was used for this acceptance.

The release binary scan identified GO-2026-6860 and GO-2026-6866 in
`golang.org/x/net` v0.60.0. The release updates it to the fixed v0.62.0, including
its required `x/sys` v0.49.0 and `x/text` v0.43.0 dependencies.

Native window inspection was interrupted by an unavailable computer-use
connection and timed out again after reconnection. The native binary built and
its isolated bundle passed strict signature verification, but native pointer
interaction and a complete in-app upgrade are not accepted for this release.

Local acceptance was performed on 2026-10-08–09 on an Apple Silicon Mac running macOS 27. Synthetic documents and isolated settings were used for mutation and provider checks. These checks support an experimental preview, not stable-release certification.

## Automated behavior checks

The focused Go tests cover safe file access, collisions, raw YAML/CRLF and file attributes, external-edit conflicts, directory replacement, linked/hard-linked files, discovery, shared Agent membership, independent copies, resources, Trash rollback and restoration after restart. Translation checks cover export without overwriting sources or aliases, cancellation, uncached fixed-sample connection tests, malicious output and bounded rate-limit retries. Cache and watcher behavior have their own package tests.

The three frontend grouping tests cover one description per Skill name, distinct versions and selection of each Agent's own copy without inheriting another copy's write access. TypeScript and the Vite/native production builds passed locally. Builds report large frontend/editor chunks and a duplicate Objective-C library warning.

ZIP/DMG checks cover checksums, archive integrity, the canonical `SkillView.app` path, executable equality, licenses and strict ad-hoc signature verification. The [Actions workflow](../.github/workflows/preview.yml) repeats the package tests, builds and runtime vulnerability checks on a separate macOS runner and records its source commit and tool versions. A successful Actions run applies only to that commit; runtime UI acceptance remains separate.

The production dependency Goldmark was updated from v1.7.16 to v1.7.17 after govulncheck identified [GO-2026-5320](https://pkg.go.dev/vuln/GO-2026-5320). The corrected binary passed the known-vulnerability scan locally. A fresh cloud scan then flagged twelve Go 1.27.1 standard-library advisories; the build was upgraded to the upstream security release, Go 1.27.2. Runtime checks exclude development dependencies, native libraries and unknown vulnerabilities. Each preview's CI run must refresh them before its packages are published.

## Native observations

- Search, selection, Markdown/resource reading, shortcuts, creation, saving, draft-preserving conflicts and quit confirmation were observed. Saved fixture bytes were inspected on disk.
- Automatic sources remained read-only; selecting a fixture source for editing enabled only that source. Favorites, theme and interface language persisted across restart. English/Chinese UI and native menus were observed. Pointer clicks removed the unwanted search focus box; keyboard focus remained visible.
- Same-name copies across Agents appeared in one row with overlapping logos/counts. Differing contents and each copy's access permissions remained available in the reader. Tab navigation, large-list scrolling and virtual rendering were observed.
- Resource creation/import/rename, external-change refresh, Trash and restoration were checked with real fixture files. Trashing one Skill copy preserved other Agent copies. Removing and restoring final Skill/resource symlinks preserved their targets, including restoration after restarting the app.
- Two native connection tests each sent only `Hello.` to a local OpenAI-compatible service. Translation retained inline code and link destinations. Copy included its target-language label. Export created a separate 0600 file; existing filenames and managed originals were preserved. Retrying after a collision succeeded with a new filename.
- Cancelling slow fixture translations returned to the original view. Late responses were neither displayed nor cached. Backend tests separately check HTTP context cancellation.

## Update checks (alpha.2 local build)

Focused updater package tests passed with race detection. They cover preview version ordering, platform-specific ZIP selection, missing manifests as manual-only downloads, malformed/unsigned manifests, mismatched release versions, corrupted downloads, wrong signing keys, cancellation, preview isolation and preservation of an already staged update.

The actual local alpha.2 ZIP passed Ed25519ph verification with the pinned release public key, Wails download/extraction, and checks of the staged app identifier, exact release version and strict macOS code signature. Its archive contains one canonical `SkillView.app`, with no private signing key or `__MACOSX` top-level directory. A separate disposable native helper check replaced an old stub app, cleaned its backup/staging directory and produced a relaunch receipt. The user's installed app was not replaced.

Native observations used an isolated local release API: the purple sidebar badge appeared for alpha.3, Settings and the native menu opened the release notes, checking the current version cleared the badge, network failure exposed retry/GitHub controls, and Chinese UI labels were observed. Preview fixtures disabled automatic checking and installation. This is not acceptance of the full save/discard/restart flow against a published future version.

The newly used `golang.org/x/mod` dependency was upgraded to v0.40.0 after the binary scan identified [GO-2026-6180](https://pkg.go.dev/vuln/GO-2026-6180) and [GO-2026-6179](https://pkg.go.dev/vuln/GO-2026-6179). The corrected alpha.2 binary passed the known-vulnerability scan. See [update publishing](updates.md) for the signing and installation requirements.

## Feedback fixes (alpha.3)

Startup tests confirm that known Skills are readable before the wider search starts. Discovery/reading progress tests cover real counters, overlapping and linked roots and cancellation without a false completion. Native observations used 80 synthetic Skills and 3,000 empty fixture directories; installed-Agent tabs wrapped into rows, reading occupied the main area, Back to Skills restored the list and navigation backgrounds were neutral.

A local OpenAI-compatible fixture showed completed-section progress, elapsed time, the translated document with a persistent completion label, cancellation returning to the original and localized HTTP 503 feedback with retry/configuration controls. AI batch tests reject incomplete and partially failed documents without caching them. No real Skill was sent to a translation provider. The public Google endpoint translated the fixed `Hello.` connection sample successfully on this Mac; availability on another computer remains unverified.

A public GitHub API check reproduced rate limiting. The fallback through the public Atom release feed then returned the correct up-to-date state. Fixture tests verify that the fallback still rejects unsigned manifests and tampered ZIPs, preserves stable/preview channel filtering and never offers an older version. Static macOS HTTP/HTTPS proxy support uses [go-ieproxy](https://github.com/mattn/go-ieproxy); loopback models are tested to bypass proxies. PAC and SOCKS-only configurations are not evaluated.

The new dependency's initial `golang.org/x/net` v0.58.0 was flagged by the binary scan for five known HTTP/2 advisories, including [GO-2026-6610](https://pkg.go.dev/vuln/GO-2026-6610). It was upgraded to the listed fixed version v0.60.0 before packaging. Package checks and CI evidence apply to the exact reviewed source and ZIP.

## Manual scanning and file tree (alpha.4)

Focused cache tests cover reopening without traversal, reading current disk bytes, replaced folders, removed sources, current automatic-link targets and metadata-only cache migration. Redirect tests cover Google GET 302, AI POST 307 with unchanged body, bounded redirect loops and refusal to forward requests to another origin. Five frontend tests cover file-tree nesting/collapse and shared Skill grouping. TypeScript, production builds and focused vet checks passed.

Native acceptance used an isolated synthetic Skill with two copies and nested references, scripts and assets. Fresh startup waited for Scan; restart opened the cached catalog without scanning. An externally added Skill stayed absent until Scan was clicked. Tree expansion, file selection, resource reading, copy selection with keyboard controls and unsaved-draft protection were observed; discarded fixture edits left original bytes unchanged. Settings language/provider menus opened inside the modal and retained the selected checkmark.

A local AI fixture redirected with HTTP 307, received the same request body and showed translation progress followed by translated text and the completion label. The public Google fixed `Hello.` sample returned HTTP 429 on this Mac during this check. Same-origin 302 handling passed locally, but Google availability on the user's other computer remains unverified. No real Skill was sent to a provider.

## Known Agent scans, appearance and diagnostics (alpha.5)

Focused race-enabled checks cover retiring old automatic sources outside the registry, keeping explicitly added folders, discovering Codex's legacy bundled-plugin root, folder-reading progress and theme-color persistence/validation. The updater tests reproduce a connection closed halfway and a download that outlives the short release-check timeout. They verify resumed bytes through Wails's original full-archive signature and extraction path, reject ignored/wrong ranges and tampered resumed archives, and stop retries on cancellation. Diagnostics tests confirm persistence, rotation, private permissions and exclusion of credential-bearing URLs and local paths.

An isolated native preview showed only library and Settings navigation, manual scanning, the theme-color palette and readable update diagnostics. The selected color survived a real app restart, which reopened the cached catalog without scanning. An unreachable loopback update service produced localized retry controls and a persisted, classified connection failure in the diagnostics view. No real Skill contents were transmitted.

## Free translation services and navigation (alpha.6)

Focused translation/app tests passed with race detection. They cover batched Google and Youdao requests, request pacing and per-provider cooldown, anonymous Youdao cookies/signatures/encrypted responses, malformed or incomplete responses, cancellation, caching, LibreTranslate batches and independent credential rollback. Thirty short Google spans use two requests instead of thirty. Incomplete translations are neither shown as complete nor cached.

The production translation service successfully translated a fixed synthetic two-section Markdown sample through both public Google and anonymous Youdao endpoints on this Mac. No real Skill was transmitted. LibreTranslate used a local synthetic API fixture; no public hosted instance or paid AI account was tested.

A browser fixture rendered the actual React components with isolated API responses. It showed no left navigation rail, Settings and update controls at the top right, and exactly four translation choices: Google, Youdao, self-hosted LibreTranslate and AI. Selecting Youdao showed no key field; saving and connection-test completion labels were observed with mock responses. The final TypeScript/frontend and native production builds passed. Native window clicks were not checked for this version.

## Folder settings and long paragraphs (alpha.7)

Focused race-enabled translation/app checks cover sentence boundaries, Unicode/emoji limits, large documents above the previous 128 KiB cap, the 1 MiB maximum, per-chunk progress and failure/cancellation without partial results. A 2,320-character paragraph produced three bounded requests and progress from 0/3 through 3/3 while preserving adjacent inline code. Fixed synthetic samples of 900, 1,500, 3,000 and 6,000 characters translated successfully through the public Youdao service; the user's exact red error was not reproduced.

A browser fixture showed the additional-folder explanation, a 26 px Add button and no empty list occupying space. Adding read-only/editable folders passed the selected permission to the mock picker; removing a mock folder restored the empty state. These browser checks do not verify the native macOS folder picker. No real folders or Skills were modified or sent to a service.

## Navigation and first regular release (0.1.0)

Back to Skills is the only action in a separate navigation row. File-tree controls and current-document actions belong to their own panes. An isolated native macOS build with disposable documents verified collapsing folders, hiding the tree without leaving the reader, Enter to restore it with focus on the selected file, reference reading and returning to the list. Original fixture bytes remained unchanged. This does not exercise Apple Gatekeeper installation or live providers.

Focused update tests verify stable/alpha ordering, stable feed filtering and signed download/extraction; file-tree unit checks pass. TypeScript and the native production build pass, including strict ad-hoc signature verification. A production React browser fixture verifies separated control positions, keyboard focus, and no horizontal page overflow at 640 px. The tested reader and the introduction website have zero automated accessibility violations. Website checks cover English/Chinese switching, loaded demo imagery and a 390 px viewport without horizontal overflow. The screenshot uses only synthetic content. No real Skills or private account data are included in the website.

## Installed plugin browsing (2026-10-14)

Focused race-enabled library tests verify Claude Code installation records, disabled Codex installations, desktop remote installation markers, semantic and pinned cache versions, exclusion of unregistered caches and staging folders, malformed metadata without private parse-error details, missing pinned versions, replaced roots, symlink refusal and rejection of plugin identifiers by Skill write APIs. Existing resource and manual-scan behavior is checked separately.

Production React browser fixtures verify first-open metadata loading, list search, Agent filters, empty/error states, version/disabled labels, relative Markdown file links, read-only controls and retained Skill selection across mode switches. The tested Chinese/light and English/dark plugin lists and readers have zero automated accessibility violations after transitions settle; the 640 px layout has no horizontal page overflow.

An isolated native Apple Silicon build with two synthetic installations verifies the Skills/Plugins switch, Claude Code and Codex names/versions, folder expansion, README reading, relative document navigation, JSON manifest preview and Back to Plugins. The production build and strict ad-hoc signature verification pass. Fixtures and generated screenshots remain ignored; no installed plugins are edited or executed. Other machines, project-specific enablement and the Agent's exact active runtime version are not verified.

## Skill archiving

Focused race-enabled tests cover complete-folder archive/restore, recovery after replacing application settings, preservation of independent copies and resource symlinks, exclusion from regular and explicitly selected archive sources, stale previews, read-only sources, failed moves and staged interruption recovery. Restoration refuses occupied destinations, replaced payloads/parents and linked archive roots. A disposable fixture crossing the external SSD and system volume verifies original bytes, file permissions and relative links after archive and restore.

The production frontend and native build pass, with strict ad-hoc signature verification. Browser fixtures exercise archive confirmation, removal from the active list, the archive list, a collision with both copies retained, successful restoration, the empty archive and an empty active list with a restore entry. Chinese/light and English/dark dialogs have zero settled automated accessibility violations; the tested 640 px viewport has no horizontal page or dialog overflow. These fixtures do not modify real Skills. Native pointer acceptance subsequently passed with the shared CLI fixture below.

## Shared CLI and Skill

Focused race-enabled tests cover shared settings and write locks, revoked editing access, stale previews, sync boundaries, recoverable cleanup, resource backups, security-rule matching, current-Agent filtering and command installation conflicts. TypeScript, the production native build, focused vet checks and strict ad-hoc signature verification pass.

An isolated native fixture verifies manual scanning, reading, archive/restore and command installation. The installed CLI reads the same catalog, cold-starts the app and activates one running window. CLI archive/restore updates the app's list on activation without scanning; restored bytes match the original. Doctor, topology, graph, backup and security commands run against synthetic files. No real Skills are changed or sent to translation services.

The local skill is now `skillview`, backed by this repository's CLI. Downloading or installing Skills is not supported. Distribution signing and another-machine installation remain unverified.

## Remaining acceptance

Alpha.9 checks compare all 79 named Agents and their macOS project/global paths in the pinned Vercel Skills CLI registry with SkillView's 83 definitions. Focused discovery tests verify fx membership in the shared root and preservation of legacy roots. A browser fixture verifies that hiding, restoring and collapsing the tree keep the reader open, Back returns to the list, and the tree visibility controls are outside the top navigation. English/Chinese labels and the tested light/open and dark/hidden reader states pass with zero automated accessibility violations. SQLite persistence and translation cache/failure tests pass with race detection; no live provider requests are made. Frontend and native production builds pass with strict ad-hoc signature verification. Native window clicks are not exercised in this pass.

Alpha.8 reading-layout checks use isolated preferences to verify old settings receive 14 px/244 px defaults, changes survive reload, invalid ranges are rejected and a failed save preserves previous values. A browser fixture exercises the production UI with synthetic documents and a mock service: drag and keyboard resizing, restored width/font size, the 20 px limit, 640 px window layout, wide-table containment, keyboard scrolling and translation progress/completion/failure states. The reader has zero automated accessibility violations in the tested final light/dark states. Translation providers and native macOS interactions are not exercised by this fixture.

The ordinary app loaded a large real catalog with installed-Agent tabs; originals were not modified. This does not prove unrestricted access to every folder or execution of a Skill by an Agent. Permission failures and scan limits are reported.

The optional Google public endpoint succeeded once on synthetic text and failed in a later native check. AI and Google Cloud response/error behavior was checked with local fixtures, not paid accounts. Credential rollback used an in-memory key store; real Keychain acceptance remains outstanding.

Developer ID signing, Apple notarization, Gatekeeper installation on another Mac, the macOS 13 minimum and Intel hardware are not accepted. See [release readiness](../RELEASE_READINESS.md).
