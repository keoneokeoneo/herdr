# 새 Mac으로 herdr-poke 이관

범위: Herdr 앱, Poke 설정, Claude/Codex 연동을 새 macOS 사용자 환경에 재구성한다.
실행 중인 터미널 프로세스는 이관하지 않는다. 아래 복원 작업은 새 머신에서 수행한다.

## 1. 기존 머신에서 준비

- 저장소의 변경과 미추적 파일을 검토해 커밋·푸시한다. `git status --short`로
  남은 항목을 확인한다. 인증정보나 런타임 파일을 커밋하지 않는다.
- 아래 표의 사용자 데이터를 별도로 백업한다. 앱이 쓰는 상태는 앱을 종료한 뒤
  일관된 시점의 복사본을 만든다. 기존 머신의 설치기는 다시 실행할 필요가 없다.
- 현재 작업 디렉터리와 새 머신의 경로 대응을 기록한다. 계정명이 같아도
  저장소 위치, Node 설치 방식, 플러그인 경로는 다를 수 있다.

| 대상 | 가져갈 것 / 처리 |
|---|---|
| Poke | Git 저장소 전체. `programs.json`, `keys.json`, `settings.json`, `layouts/poke.json`, `themes/poke.json`, `mods/poke/`, helper와 scripts 포함 |
| Claude 개인 설정 | `~/.claude/settings.json`, `~/.claude/CLAUDE.md`, `~/.claude/skills/`에서 필요한 설정·스킬. 새 설정과 비교해 병합 |
| Codex 개인 설정 | `~/.codex/config.toml`, `~/.codex/skills/`, 별도 사용자 지침이 있으면 함께 검토·복원 |
| 공유 스킬 | `~/.agents/skills/`. Claude skills의 상대 심볼릭 링크 대상도 함께 복원 |
| 로컬 marketplace | 참조하는 저장소 자체를 새 경로에 준비하고 설정의 경로 수정. 외부 플러그인은 새 머신에서 재설치 |
| 프로젝트 | 소스 저장소, 각 프로젝트의 지침·로컬 설정. 홈 경로 변경 시 hooks, MCP 실행 명령, 프로젝트 신뢰 경로도 검토 |
| 포켓몬 성장·도감 | `~/Library/Application Support/PokeTokenBar/companion-state.json`, `sprites/`, `pokemon-details-v1/`. `last-snapshot.json`은 선택 사항이며 이전 시점의 표시값임 |
| 포켓몬 설정 | `io.github.chattymin.poketokenbar` preferences. 특히 성장 난이도 `growthDifficulty`를 기록하고 새 앱 설정에서 복원 |
| Poke 이미지/울음소리 | `~/.config/herdr-app/poke-assets/`는 선택 백업. 없으면 필요한 에셋을 다시 수집 |
| 사용량 이력 | 필요할 때 `~/.claude/projects/`, `~/.codex/sessions/`. Poke가 로그에서 최근 14일을 다시 집계. 이는 Claude/Codex 세션 재개 전체를 보장하는 절차는 아님 |
| 폰트 | Jetendard를 새 머신에 설치. IBM Plex UI 폰트는 저장소에 포함 |

pokeTokenBar의 `usage-cache.json`은 열거나 복호화하지 않는다. 성장·도감 이관은
위에 명시한 상태와 에셋을 대상으로 하며, 앱 전체 데이터의 공식 복원 기능을
대체하지 않는다. 복원 후 앱에서 성장·도감을 직접 확인한다.

다음은 새 머신에 그대로 덮어쓰지 않는다:

- `~/.local/state/herdr-poke/`의 `install*.json`, 원본 백업, `labels-*`, lock 파일.
  설치 이력은 새 머신에서 생성한다. 사용량 로그를 복원했다면 usage ledger와
  offsets도 복사하지 않고 재집계한다. `limits.json`은 다음 활동으로 갱신한다.
- 기존 helper 바이너리, nvm 실행 파일·shim, launchd plist. 새 머신에서 재생성한다.
- Herdr의 sockets, `data/`, `view.json`, `shared.json`, 창 상태 등 런타임 파일.
  새 머신에서는 작업 공간과 프로젝트 경로를 다시 연결한다. 필요하면 원본을
  별도 보관하되 테마 설치 과정에서 자동 복원하지 않는다.
- Codex/Claude의 플러그인 캐시·임시 runtime 경로. 새 설치가 생성한 경로를 사용한다.
  인증은 새 머신에서 로그인해 확인한다. 테마 설치기는 자격증명을 이전하지 않는다.

## 2. 새 머신의 도구 준비

Herdr 앱과 `herdr-app`/`hc` CLI, Git, Python 3, Go, Claude Code, Codex CLI를 준비한다.
Go 요구 버전은 `helper/go.mod`가 기준이다. Codex는 `--help`에 `--no-daemon`을
지원하는 배포본이어야 한다. 미지원 버전에서 옵션만 제거하면 라벨 수집이 깨진다.

Node 기반 hooks/MCP를 사용한다면 Node/npm/npx도 준비한다. nvm 사용 시 Node를
설치하고 default 버전을 지정한다. nvm이 없어도 설치할 수 있지만, 설치기는
비-nvm Node용 shim을 생성하지 않으므로 새 Herdr 터미널에서도 실행되는지 확인한다.

Jetendard와 pokeTokenBar는 별도로 설치한다. pokeTokenBar를 초기화한 후 종료하고
백업한 상태를 복원한다. 새 머신에 이미 상태가 있다면 먼저 별도로 백업한다.

## 3. 저장소 배치와 설치

1. Herdr를 한 번 실행해 초기 설정을 생성한다. 사용 중인 새 머신이라면 진행 중인
   작업을 정리한 후 앱을 종료하고 배치한다. 아래 예시는 기존 설정 폴더를 통째로
   보관하고 새 checkout을 사용하는 방식이다. 기존 작업 공간을 자동 복원하지 않는다.
2. SSH/Git 접근을 준비하고 **새 머신에서만** 아래를 실행한다. clone이 성공한 뒤
   기존 폴더를 보관하므로 다운로드 실패 시 기존 설정은 그대로 남는다.

   ```sh
   (
     set -eu
     mkdir -p "$HOME/.config"
     migration_stage=$(mktemp -d "$HOME/.config/herdr-poke-migration.XXXXXX")
     git clone git@github.com:keoneokeoneo/herdr-poke.git "$migration_stage/repo"
     if [ -e "$HOME/.config/herdr-app" ] || [ -L "$HOME/.config/herdr-app" ]; then
       mv "$HOME/.config/herdr-app" "$migration_stage/previous-herdr-app"
     fi
     mv "$migration_stage/repo" "$HOME/.config/herdr-app"
     printf 'Previous configuration, if any: %s/previous-herdr-app\n' "$migration_stage"
   )
   ```

3. 이전 설정과 새 checkout을 비교한다. 개인 설정이 필요하면 `settings.json`,
   `keys.json`, `programs.json` 등을 검토해 병합하고 선택한 `poke-assets/`를 복원한다.
   `programs.codex.run`의 wrapper와 `--no-daemon`은 유지한다. 메뉴는 zsh alias를
   확장하지 않으므로 이 파일이 필요하다.
4. Claude/Codex 개인 설정과 스킬을 병합한다. 이전 `/Users/<name>/` 경로를 검색해
   파일별로 검토한다. 경로 전체를 자동 치환하지 않는다. statusline의 helper 경로는
   설치기가 새 HOME 기준으로 갱신한다. 외부 marketplace, hooks, MCP, 플러그인의
   경로와 끊어진 스킬 링크는 별도로 수정한다.
5. 읽기 전용 준비 검사를 실행한다:

   ```sh
   cd "$HOME/.config/herdr-app"
   python3 scripts/preflight.py
   ```

   필수 도구 누락, 실행 실패, Go 버전 부족, Codex 옵션 미지원은 실패한다.
   Node/nvm, Jetendard, pokeTokenBar 상태는 선택 기능 경고다. 검사에서는
   버전·도움말 명령을 실행하지만 로그인·서버 시작·에이전트 세션은 만들지 않는다.
   CLI 자체가 수행하는 초기화까지 차단하는 sandbox는 아니다.
6. Claude 세션을 닫고 일반 터미널에서 `bash scripts/install.sh`를 실행한다.
   준비 검사는 빌드·설치 journal·백업·설정 변경 전에 다시 실행된다. 통과 후에는
   helper, 설정, nvm shim(해당 시), launchd 작업을 생성한다.
   `HERDR_POKE_DRY=1`도 파일을 변경하므로 준비 검사 대용으로 사용하지 않는다.
7. 새 zsh를 열고 Dock에서 Herdr를 시작한다. 앱이 설치 중 꺼져 있어 layout 전환이
   안 됐다면 `cmd-shift-l`에서 `poke`를 선택한다.

기존 권한 정책을 유지하려면 Claude `permissions.defaultMode="bypassPermissions"`,
Codex `approval_policy="never"`, `sandbox_mode="danger-full-access"`를 직접 확인한다.
설치기는 이 설정을 변경하지 않는다. Codex 메뉴 명령에는 기존 `--yolo`가 포함된다.

## 4. 새 머신에서 완료 확인

- `herdr-app check mods/poke`: `check passed`.
- `herdr-app status`: `"errors": []`, 선택 layout은 `poke`.
- 새 Claude/Codex 메뉴 탭에서 한 번 대화하고 model/effort/ctx 표시 확인.
  첫 로그·토큰 이벤트가 생기기 전에는 일부 값이 없을 수 있다.
- `hc get <터미널 주소> --json`의 실행 명령과 `poke.model`, `poke.effort`, `poke.ctx`
  labels 확인. Codex 메뉴 실행은 wrapper와 `--no-daemon`을 포함해야 한다.
- 새 터미널에서 `node --version`, `npm --version`, `npx --version` 확인.
  실제 사용하는 hooks/MCP도 확인한다. 버전 검사는 로그인이나 연결 성공을 보장하지 않는다.
- companion, 성장 난이도·레벨, Caught 도감, GIF·울음소리와 Jetendard 표시 확인.
- `~/.local/bin/herdr-poke usage` 실행 후 사용량 패널 확인. 과거 로그를 옮기지
  않았다면 새 머신의 기록부터 표시되는 것이 정상이다.
- `launchctl print "gui/$(id -u)/dev.keno.herdr-poke.usage"`로 작업 등록 확인.
- Claude/Codex 로그인, 외부 플러그인 연결, 프로젝트 접근을 직접 확인.

자동 테스트는 임시 HOME과 가짜 CLI로 설치·재설치·복구 및 실패 시 무변경을
검증한다. 실제 신규 머신의 인증·launchd·폰트·화면 검증을 대체하지 않는다.
