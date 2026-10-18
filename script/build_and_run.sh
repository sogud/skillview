#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/toolchain.sh"
mkdir -p "$SKILLVIEW_TMP_DIR"
MODE="${1:-run}"
case "$MODE" in run|--verify|--logs|--debug|--build-only) ;; *) echo "Usage: $0 [--verify|--logs|--debug|--build-only]" >&2;exit 2;; esac
# Stop only a previous process recorded by this project's own launcher.
if [[ -f "$SKILLVIEW_TMP_DIR/app.pid" ]]; then
 previous_pid="$(cat "$SKILLVIEW_TMP_DIR/app.pid")"
 previous_command="$(ps -p "$previous_pid" -o command= || true)"
 if [[ "$previous_command" == "$PROJECT_DIR/bin/SkillView.app/Contents/MacOS/SkillView"* ]]; then
  kill "$previous_pid" || true
 fi
fi
bash "$PROJECT_DIR/script/build.sh"
APP_BUNDLE="$PROJECT_DIR/bin/SkillView.app"
if [[ "$MODE" == "--build-only" ]]; then exit 0; fi
if [[ "$MODE" == "--debug" ]]; then exec lldb -- "$APP_BUNDLE/Contents/MacOS/SkillView"; fi
launch_options=(-n "$APP_BUNDLE")
for variable in SKILLVIEW_DATA_DIR SKILLVIEW_SOURCE SKILLVIEW_FIXTURE_EDIT; do
 if [[ -n "${!variable:-}" ]]; then launch_options+=(--env "$variable=${!variable}"); fi
done
/usr/bin/open "${launch_options[@]}"
sleep 1
ps -axo pid=,command= | awk -v path="$APP_BUNDLE/Contents/MacOS/SkillView" '$2 == path { print $1 }' > "$SKILLVIEW_TMP_DIR/app.pid"
if [[ "$MODE" == "--verify" ]]; then test -s "$SKILLVIEW_TMP_DIR/app.pid"; fi
if [[ "$MODE" == "--logs" ]]; then exec /usr/bin/log stream --info --style compact --predicate 'process == "SkillView"'; fi
