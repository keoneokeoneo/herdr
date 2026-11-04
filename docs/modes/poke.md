# Poke — 변경하는 사람(사람 또는 LLM)을 위한 가이드

저장소 규칙은 먼저 `docs/guide.md`를 읽고, 그다음 변경이 건드리는 파일만 연다. Poke는 **base**
(`docs/modes/base.md`) 위에 얹는 페르소나 계층으로, pokeTokenBar 동료(companion), Dex 탭, 파트너 스프라이트,
울음소리를 추가한다. 사이드바, 탭 바, 사용량 바, 오른쪽 패널은 base가 담당하며 Poke 없이도 동작한다. Poke는
base를 필요로 한다. 제품 의도와 최초 결정은 `docs/specs/2026-11-03-herdr-poke-design.md`에 있고(그 문서의
"Current state" 섹션에 v1 이후 달라진 점이 나열되어 있음), 계층 분리는
`docs/specs/2026-11-04-layer-separation-design.md`에 있다. Herdr 자체의 API 참조는 저장소 루트에 생성되는
`AGENTS.md`이며 `herdr.d.ts`, `gpui-kit.d.ts`가 함께 있다. 그 파일은 약 1000줄이므로 필요한 API를 검색해서
찾는다. 처음부터 끝까지 읽지 말고, 절대 수정하지 않는다. Herdr가 시작할 때마다 다시 쓴다.

## 위치별 역할

설치되면 저장소 루트가 Herdr의 홈인 `~/.config/herdr-app`이 된다. `.gitignore`에는 Herdr가 쓰는 모든 것이
나열되어 있어, `git status`에는 우리 파일만 보인다.

| 바꾸려는 것 | 볼 곳 |
|---|---|
| 창 구조 (블록, 크기, 어떤 view가 어디에 가는지, traffic lights) | `layouts/poke.json` (공유 블록 id를 `layouts/base.json`과 동일하게 유지; 테스트가 확인함) |
| 색 | `themes/poke.json`, `mods/poke/views/ui.js` `C` (타입/chip 강조색) |
| Codex 메뉴 실행 명령 | `programs.json` (`programs.codex.run`) |
| 키 | `keys.json`, 섹션 `layout:poke` (`cmd-j`, `cmd-k`, `cmd-shift-o`는 base command) |
| 블록이 보여 주는 view | `mods/poke/main.js` (`props.view`: `companion`, `dex`, `host`, `camp`) |
| 사이드바: pokeTokenBar 동료 카드 / 알 | `mods/poke/views/companion.js` |
| 오른쪽 패널 Poke 탭 (Dex 그리드, 탭 타입 컨트롤) | `mods/poke/views/dex.js` |
| 일어나는 일: 파트너 배정, 울음소리, Dex 동기화, 레벨/진화 토스트, 저녁 어둡게 | `mods/poke/behaviours.js`, 보이지 않는 `poke-host` 블록(`mods/poke/views/host.js`)이 시작 |
| 페르소나가 읽는 agent 행 (상태와 파트너 label) | `mods/poke/views/rows.js` |
| Garden 미니맵과 확장 창 (하나의 정원과 세션 파트너) | `mods/poke/views/camp.js`, `mods/poke/lib/camp.js` |
| 스프라이트, pill, GIF 다시 그리기 | `mods/poke/views/ui.js` (`sprite`, `alive`, `pill`) |
| 순수 로직 (파트너 선택, 성장 공식, 울음소리, 기본값) | `mods/poke/lib/`, 테스트는 `tests/poke/mod-lib/` |
| 사이드바, 탭, pane 헤더, 하단 바, Usage 탭, 패널 탭, 아바타 | **base**: `docs/modes/base.md` |
| 세션 소유권과 완전한 telemetry 스냅샷 | `helpers/poke/internal/labels/session.go`, `mods/base/lib/telemetry.js` |
| label 캐싱, 실패 백오프, `hc label` 타임아웃 | `helpers/poke/internal/labels/labels.go` |
| CLI 디스패치와 헬퍼 빌드 | `helpers/poke/main.go`, `scripts/poke/build.sh` |
| 모드 등록 | `mods/poke/mod.json` (번들된 UI 폰트는 `mods/base/fonts/`) |
| hc용 Node 진입점 | `scripts/poke/nvm-shim` |
| Codex → label과 한도 (래퍼와 `--attach`) | `helpers/poke/internal/codex` (`wrap.go`, `attach.go`, `find.go`, `rollout.go`) |
| 스프라이트, 울음소리, 이름, GIF 정규화 | `helpers/poke/internal/assets` (`assets.go`, `normalize.go`) |
| 14일 사용량 집계 (launchd 10분마다) | `helpers/poke/internal/usage` |
| 상태 파일과 lock | `helpers/poke/internal/state` (`~/.local/state/herdr-poke/`; 사용량 계약은 `~/.local/state/herdr-app/usage/`) |
| 설치 선행 조건 | `scripts/poke/preflight.py` (읽기 전용 설치 점검), `tests/poke/scripts/test_preflight.py` |
| 새 Mac으로 이전 | `docs/plans/2026-11-04-machine-migration.md` |
| 설치와 되돌리기 | `scripts/poke/install.sh`, `scripts/poke/uninstall.sh`, `scripts/poke/settings_edit.py` (설정 journal), `scripts/poke/install_files.py` (설치된 파일의 소유권/백업), 테스트는 `tests/poke/scripts/` |
| setup의 선택적 설치 | `scripts/setups.json`, `scripts/setup.py`, `tests/setup/test_setup.py` |

Seen은 여러 space에 걸쳐 열려 있는 agent 세션의 파트너를 중복 제거한 스냅샷이다.
기존의 2초 behaviour tick은 구성원이 바뀔 때만 이를 교체한다.

## 데이터 흐름 한 단락 요약

Herdr는 모드에 `agents()`, `spaces()`, `terminals()`(상태, label, 포커스)를 제공한다. Go 헬퍼
`~/.local/bin/herdr-poke`는 Codex에 대해 Herdr에 없는 것을 보탠다.
- `codex -- …` 래퍼(zsh alias)가 Codex 세션을 미러링한다. 다른 방식으로 시작한 Codex는 오래된 모델 label이
  남아 있더라도 `mods/base/behaviours.js`가 주기적으로 `codex --attach <t>` 프로브를 보낸다. 로컬의 bare
  Codex 포그라운드 프로세스는 지원되며, 헬퍼는 원격 머신의 PID나 파일을 검사할 수 없다.

헬퍼는 완전한 터미널 스냅샷(`poke.model`, `poke.effort`, `poke.ctx`)을 `poke.owner`(포그라운드 PID),
`poke.provider`, `poke.session`과 함께 게시하고, 한도 파일 `~/.local/state/herdr-app/usage/codex.json`도
게시한다. `usage`는 같은 폴더에 `daily.json`을 쓴다. Claude의 한도는 전역 Claude statusline에서 오며 Poke는
더 이상 이를 건드리지 않는다. 따라서 Claude 행에는 모델, effort, context가 없다. 스프라이트, 울음소리, 이름은
`~/.config/herdr-app/poke-assets/`에 저장되며 모드는 이를 `poke-assets/<file>`로 읽고, Poke는 파일 목록을
공유 `poke.sprites`로 게시해 base가 아바타를 그릴 수 있게 한다. pokeTokenBar의 폴더
(`~/Library/Application Support/PokeTokenBar`)는 읽기 전용이다.

## 함정 (하나하나가 실제로 시간을 잡아먹었다)

일반적인 Herdr 함정(핫 리로드, shipped 모드, `setShared`, 색, traffic lights, Dock과 `hc` PATH)은
[Herdr 커스터마이징](../herdr-dev.md)에 있다. 아래는 Poke 고유의 함정이다.

- **GIF는 창이 다시 그려질 때만 진행된다.** `mods/poke/views/ui.js`의 `alive()`(와 `mods/base/views/ui.js`의
  복사본)는 스프라이트를 서브픽셀 크기의 네이티브 `.loop()`로 감싸 계속 움직이게 한다. 정적 PNG는 이 루프를
  거치지 않는다. 예전에 로컬에서 관찰한 값은 CPU 약 3%였으며, 머신이나 스프라이트 수에 걸쳐 측정한 예산은 아니다.
- **스프라이트 GIF는 프레임이 고르지 않게 온다**(프레임마다 더 작은 사각형, 정사각형이 아닌 캔버스). Herdr는 모든
  프레임을 박스에 맞게 늘린다. 헬퍼가 이를 정사각형으로 맞춘다(복구는 `assets --normalize`, 다운로드, companion
  복사, `assets --dex`가 새 스프라이트를 정규화). 원본 GIF를 그대로 보여 주지 않는다.
- **Codex 미러 소유권:** 래퍼와 attach 모두 모니터링 전에 터미널의 flock을 획득해야 한다. attach는 5초마다
  터미널 포그라운드를 확인하고, 바뀌거나 조회할 수 없으면 종료한다. 1분 주기 프로브가 멈춘 미러를 복구하며, 남아
  있는 모델 label은 미러가 살아 있다는 증거가 아니다. lock 파일은 의도적으로 디스크에 남긴다. flock은 프로세스가
  종료되면 해제되므로, 살아 있는 lock을 unlink하지 않는다. label 백오프는 게시를 다시 시도할 수 있을 때까지
  오류를 반환하며, 최신 Codex 한도의 비교와 교체는 같은 lock 아래에서 일어난다.
- **Telemetry 소유권:** UI는 포그라운드 PID와 provider가 일치하지 않는 스냅샷을 숨긴다. Codex 래퍼는 게시 전에
  제한 시간 안에서 `hc get`과 프로세스 조상을 확인하므로, 지연된 이전 프로세스가 교체된 세션을 가로챌 수 없다.
  Codex는 프로세스 내 세션 전환을 감지하기 위해 열려 있는 rollout을 5초마다 다시 확인하며, 전환 시 누락된
  필드를 비운다. PID 재사용은 프로세스 시작 시각으로 식별하는 검사가 아니며, 지원되지 않거나 레거시인 스냅샷은
  사용 불가로 표시된다. 헬퍼를 다시 빌드해도 이미 실행 중인 Codex 래퍼/attach 프로세스는 업그레이드되지 않는다.
  기존 래퍼는 사용자가 세션을 마친 뒤 정상적으로 재시작해야 하며, 업그레이드를 위해 작업을 중단시키지 않는다.
- **Codex에는 `--no-daemon`이 필요하다.** 없으면 `lsof`가 프로세스를 rollout과 연결하지 못하고
  `HC_TERMINAL_ID`를 잃는다.

## 변경을 검증하는 방법

1. `herdr-app check mods/poke`와 `herdr-app check mods/base`가 `check passed`를 출력해야 한다.
   `herdr-app status`가 `"errors": []`를 보여 줘야 한다.
2. 로직: `npm test`(모드 `lib/`, 레이아웃, 문서)와 `cd helpers/poke && go test ./...`. 설치기는
   `npm run test:poke:scripts`, 선택적 설치기는 `npm run test:setup`.
   설치기 통합 테스트는 임시 HOME과 stub 빌더를 사용하며, launchd와 실제 앱은 제외된다. 실패하는 테스트를
   먼저 작성한다.
3. 시각: `herdr-app shot out.png`는 macOS traffic lights 없이 마지막 프레임을 저장한다. lights까지 보려면 창 id
   (`CGWindowListCopyWindowInfo`, owner "Herdr")를 찾아 `screencapture -x -o -l <id> out.png`를 실행한다.
   캡처는 2배 크기이므로 보기 전에 자른다. `sips -c`는 중앙을 기준으로 자르므로, 작은 Swift나 CG 스크립트를
   사용한다.
4. 사이드바 rail: `herdr-app do layout.resize '{"id":"sidebar","px":92}'`로 접고, `262`로 복원한다.
   끝나면 너비를 원래대로 돌려 놓는다.
5. 헬퍼 변경: `cd helpers/poke && go build -o ~/.local/bin/herdr-poke .`. 숨겨진 테스트 터미널
   (`herdr.space` label이 없어 앱에 표시되지 않음):
   `hc run --bg --no-notify --name x-test -- codex --no-daemon "Reply only OK."`, 그다음 `hc kill`, 그다음 `hc rm`.
6. 사용자의 다른 터미널은 절대 건드리지 않는다.

## 규칙

- 커밋은 저장소 이력을 따라 `type(scope): 한국어 명사형` 제목(`~/Documents/work/commit-convention.md`는
  `~/Documents/work` 아래에서만 적용)과 ~함/~됨으로 끝나는 본문 줄로 쓴다. 커밋과 PR에 AI 서명을 넣지 않는다.
  `keoneokeoneo/herdr`(private)로의 push는 괜찮다.
- 문서는 `docs/specs/`와 `docs/plans/`에 둔다(도구가 다른 위치를 제안하더라도). 문서에 커밋 해시를 넣지 않는다.
- pokeTokenBar 파일은 읽기 전용이다. `usage-cache.json`은 열거나 복호화하지 않는다.
- `~/.claude/settings.json`, `~/.zshrc`, `~/.codex/config.toml`, Herdr의 `settings.json`을 수정하기 전에
  백업한다.
- UI 텍스트는 영어다. 픽셀 아트는 Pokémon뿐이다. 테마는 어둡고 차분하게 유지한다.

## 저장소 밖의 것 (새 머신에 필요한 것)

백업, 안전한 체크아웃 위치, 외부 설정, 이전 후 검증은 `docs/plans/2026-11-04-machine-migration.md`를 사용한다.
`python3 scripts/poke/preflight.py`는 설치 전에 필수 CLI, 로컬 Go 버전, Codex `--no-daemon` 지원을 점검한다.
Node/nvm, Jetendard, pokeTokenBar 상태는 경고를 낸다. 로그인이나 실제 UI는 검증하지 않는다. `install.sh`는
설치 쓰기 전에 이를 실행하며, `HERDR_POKE_DRY=1`은 여전히 파일을 쓰므로 격리된 테스트용이지 읽기 전용 점검이
아니다.

`scripts/poke/install.sh`가 이 대부분을 설정하므로, 이전 후에 확인한다. 먼저 `scripts/setup.py`로 `base`와
`poke`를 설치한다. 전역 Claude `statusLine`은 절대 수정하지 않는다. 없는 설정 파일은 생성한다. 없는 Codex
alias는 추가하고, 정확히 일치하는 이전 alias는 교체하며, 사용자 정의 또는 중복 alias는 덮어쓰지 않고 중단한다.
설치기는 `.zshrc`에 관리되는 `~/.local/bin` PATH 줄을 추가한다.

설정 원본은 첫 쓰기 전에 journal에 기록된다. 반복 설치해도 그 원본은 보존된다. 설치된 바이너리, shim, 링크,
launchd plist에는 독립된 소유권 journal이 있다. 제거는 설치된 파일이 여전히 일치할 때만 원본을 복원한다.
사용자가 교체한 파일과 journal에 없는 레거시 파일은 그대로 둔다. 파일을 쓴 뒤 fingerprint를 기록하기 전에
중단되면, 저장된 원본에서 수동으로 복구하도록 남겨 둔다. 설치기를 동시에 실행하지 않는다.

제거는 선택된 레이아웃, 저장소, 런타임 상태, 에셋, 백업을 그대로 둔다. 의도적으로 수동으로 둔 권한 설정이나
외부 스킬은 되돌리지 않는다.

| 항목 | 위치 | 설치 주체 |
|---|---|---|
| 터미널 폰트 Jetendard | macOS 폰트 | 수동; 번들되지 않음 |
| Node 버전과 기본 선택 | `${NVM_DIR:-~/.nvm}` | nvm/Node를 먼저 설치; shim이 이를 필요로 함 |
| 헬퍼 바이너리 | `~/.local/bin/herdr-poke` | install.sh |
| Claude 한도 파일 (전역 statusline이 `claude.json`을 씀) | `~/.claude/statusline/` | 수동; 이 저장소에 없음, `docs/modes/base.md` 참고 |
| `alias codex='herdr-poke codex -- codex --yolo --no-daemon'` | `~/.zshrc` | install.sh (manifest) |
| Herdr 사운드 끄기 | Herdr `settings.json` | install.sh (manifest) |
| launchd 사용량 작업 `dev.keno.herdr-poke.usage` | `~/Library/LaunchAgents/` | install.sh |
| hc용 node/npm/npx shim | `~/.local/libexec/nvm-shim`, `~/.local/bin/{node,npm,npx}` | install.sh |
| 어디서도 권한 프롬프트 없음: Codex `approval_policy="never"`, `sandbox_mode="danger-full-access"`; Claude `permissions.defaultMode="bypassPermissions"` | `~/.codex/config.toml`, `~/.claude/settings.json` | 수동, 의도적으로 (보안 설정) |
| 리뷰 스킬이 alias 플래그와 Codex 래퍼로 실행됨 | `~/.claude/skills/dual-review-hc`, `dual-review` | 이 저장소에 없음 |

## 백그라운드 작업과 UI 범위

| 작업 | 주기 / 트리거 | 비용과 범위 |
|---|---|---|
| agent 전환, 파트너 배정 | 2초 | Herdr 행을 읽음; 무조건적인 notify 없음 |
| attach 상태 프로브 (base 상태 바 복사본) | Codex당 60초 | 헬퍼 하나; 잡고 있는 flock은 hc를 조회하기 전에 종료됨 |
| attach 포그라운드 검증 | 살아 있는 attach당 5초 | 제한 시간이 있는 `hc get` 한 번; 실패하면 종료하고 나중에 재시도 |
| rollout 탐색 / tail | 약 1초 | 찾을 때까지 `lsof`; 이후 증분 읽기 + 5초마다 열린 rollout 재확인 |
| Companion + Dex 복사, 진화/레벨 토스트 | PTB 폴더 감시(`companion-state.json`, `sprites/`) + 300초 | 실행당 헬퍼 세 번 실행, 병합됨; 변경 없는 파일은 다시 쓰지 않음; 새 변형 GIF만 정규화 |
| 한도/사용량 로드 (base)와 Dex 로드 (Poke) | 필터링된 폴더 이벤트 + 15초 | 변경 없는 notify 억제; 동시 로드는 병합 |
| 상태/패널 시간 label | 60초 | 카운트다운/만료 갱신 |
| agent/pane 경과 시간 label | 초기화된 view당 20초 | 숨겨진 view에서도 타이머 계속 |
| Companion 카드 / 이벤트 | 30초 + 카드 폴더 감시 | 필터링/병합된 읽기; 기본값 조회는 30초마다; 보이는 변경에만 notify |
| 사용량 집계 | launchd 600초마다 | 로그 증분 읽기지만, 디렉터리 탐색은 로그 루트를 순회 |

에셋 작업은 pokeTokenBar가 상태를 쓸 때마다(몇 분마다)와 5분 폴백 시점에 헬퍼를 세 개씩 시작하며, 파트너 가져오기,
재시도, attach 프로브도 포함된다. CPU 영향은 다시 측정하지 않았다.

브랜드 버튼은 rail로 접히고, `cmd-b`는 사이드바를 숨긴다. 더블클릭 이름 변경과 순서 변경은 펼쳐진 space와 탭에
구현되어 있으며, agent 행이나 rail의 space에는 없다. pane 푸터/dirty/ahead, 리뷰 완료 배지, 파트너 기반 터미널
이름 변경, 선택 가능한 저장소별 타입 모드는 보류되어 있다. 현재 범위는 설계 문서의 Current state 섹션에도 기록되어
있다.

## 통합 오른쪽 패널

`panel-body`(`layouts/poke.json`에 있음)는 Usage / Changes / Files / Poke / Obsidian 중 하나를 보여 주며, 탭은
`panel-tabs` 블록에 선언되어 있다. Usage, Changes, Files, Obsidian은 base의 것이고, Poke 탭(`poke-dex`)은
Dex다. 패널 버튼 또는 Cmd-K로 패널 전체를 숨기거나 표시하고, Cmd-Shift-O(`base.notes`, 하단 바가 처리)로
Obsidian을 연다. 별도의 Notes 패널은 없다. Obsidian 리더는 접을 수 있는 폴더 트리를 가지며 vault/노트 이름을
전체 경로 없이 보여 준다. [리더 가이드](obsidian.md)를 참고한다.

## Poké Garden

Poke 하단에 실시간 미니맵이 표시된다. 미니맵이나 `Expand`를 누르면 큰 창이
열리며, 닫으면 미니맵은 그대로 남는다. 모든 space의 agent를
세션당 한 마리씩 하나의 잔디 정원에 표시한다. 상태별 구역은 없으며 Working은 밝기가 바뀌는
점 말풍선, Needs you는 !, Done은 체크, Idle은 z로 표현한다. 상태 변경 시 위치는 유지하며,
세션 추가/종료 시 주소 순서에 따라 재배치될 수 있다. hover는 세션 상세, 클릭은 세션 이동이다.
기존 Active는 종별 중복 제거지만 정원은 같은 종도 세션별로 유지한다. 미배정/에셋 누락은 ?,
알 수 없는 상태는 Unknown으로 표시한다. Herdr live 데이터를 사용하며 타이머를 추가하지 않는다.
`herdr-app do poke.camp`로도 연다. `Close ×`로 닫고 헤더를 드래그해서 옮긴다.
같은 버튼을 다시 눌러도 중복되지 않는다. 많은 세션과 작은 창에서는 정원을 스크롤한다.
창 크기는 최대 1,000×700 논리 px이며 작은 앱에서는 너비 88% / 높이 80% 이내로 줄어든다.
앱의 확대 배율은 다른 UI와 동일하게 적용된다.

큰 창은 96px 스프라이트, 14px 세션 이름, 13px 상태명을 사용한다. 숫자 기본 탭 이름은
세션 제목으로 대체한다. Done은 보지 않은 완료, Idle은 다음 작업 대기, Blocked는 입력 대기로
설명하며 실제 작업 내용을 추측하지 않는다. 포켓몬 위치의 미세 애니메이션은 사용하지 않는다.

Active 옆 Garden: On/Off는 미니맵 표시를 제어한다. 기본값은 On이며 선택은
`poke.gardenEnabled` shared 값으로 앱 재시작 후에도 유지한다. 이미 연 큰 창은 유지한다.
