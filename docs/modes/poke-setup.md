# Poke — 설치

Herdr용 차분한 다크 작업 setup. [Base](base.md) 위에 얹는 레이아웃과 페르소나 모드로, 모든 Claude Code와 Codex
세션을 한눈에 보여 준다: 누가 나를 필요로 하는지, 누가 작업 중인지, 5h와 주간 한도가 얼마나 남았는지. 각 세션에는
무작위 Pokémon 파트너가 붙고, 사이드바에는 pokeTokenBar companion이 표시된다. 픽셀 아트는 Pokémon뿐이다. Poke는
Base를 필요로 하며, Base(와 Obsidian)만 Poke 없이 설치할 수도 있다. README를 참고한다.

Poke 탭에는 세션별 포켓몬이 함께 있는 정원 미니맵도 포함된다. Active 옆 `Garden: On / Off`로
표시를 선택하고, 미니맵이나 `Expand`로 큰 창을 연다. 포켓몬 클릭은 해당 세션으로 이동하며
`Close ×`는 큰 창만 닫는다. 별도 설치는 필요 없다. [사용 방법](poke.md#poké-garden)을 참고한다.

## 변경하기

`docs/guide.md`와 `docs/modes/poke.md`부터 시작한다. 창의 각 부분에서 해당 파일로의 지도, 함정, 검증 방법,
저장소 밖에 있는 것이 담겨 있다. Claude Code는 `CLAUDE.md`를 통해 이를 로드한다. Codex는
`AGENTS.override.md`를 통해 로드하며, Herdr가 생성하는 `AGENTS.md`는 API 참조로 남는다. 두 도구 모두
`.agents/skills/`의 `herdr-poke-dev` 스킬을 사용한다.

## 선행 조건

- Herdr: `brew install --cask herdr-app`
- pokeTokenBar: `brew install --cask chattymin/tap/poke-token-bar` (읽기 전용; 절대 수정하지 않음)
- `helpers/poke/go.mod`에 맞는 Go 또는 그 이상: `brew install go`
- Python 3 (설치기), Jetendard 터미널 폰트 (별도 설치)
- 선택적인 hc node/npm/npx shim을 위해 Node 버전이 설치된 nvm
- Claude Code와 `--no-daemon`을 지원하는 Codex CLI

## 설치

1. [새 Mac 이전 체크리스트](../plans/2026-11-04-machine-migration.md)에 따라 기존 Herdr 설정을 백업하고 저장소를
   `~/.config/herdr-app`에 둔다. 먼저 스테이징 폴더에 clone하고, 기존 설정 위에 강제로 checkout하지 않는다.
   `programs.json`은 유지한다. 메뉴에서 실행한 Codex가 래퍼와 `--no-daemon`을 쓰도록 한다.
2. `python3 scripts/poke/preflight.py`를 실행한다. 필수 도구, `helpers/poke/go.mod`의 Go 버전, Codex
   `--no-daemon` 지원은 통과해야 한다. Node/nvm, Jetendard, pokeTokenBar 상태는 선택 기능에 대한 경고를 낸다.
   로그인과 실제 UI 검증은 수동으로 남는다.
3. `bash scripts/poke/install.sh`를 실행한다(`python3 scripts/setup.py install poke`와 동일). 설치 쓰기 전에
   preflight를 다시 실행한 뒤 다음을 수행한다.
   - `base`와 `poke` 모드, 레이아웃, 테마, `layout:` 키 바인딩을 설치한다
     (`scripts/setup.py`; 저장소가 이미 `~/.config/herdr-app`이면 아무것도 복사하지 않음).
   - `~/.local/bin/herdr-poke`를 빌드한다.
   - `~/.zshrc`(`codex`가 `herdr-poke codex -- codex --yolo --no-daemon`을 거쳐 실행됨)를 백업하고 수정한다.
     Herdr의 `settings.json`은 수정하지 않는다. 예전 설치가 울음소리 때문에 꺼 둔 앱 사운드가 그대로 꺼져
     있으면 원래 값으로 되돌린다.
     `~/.claude/settings.json`은 절대 수정하지 않는다. Claude의 한도는 전역 statusline에서 오며,
     [Base](base.md)를 참고한다.
   - companion 스프라이트를 복사하고, 14일치 사용량을 수집하고, launchd 작업
     (`dev.keno.herdr-poke.usage`, 10분마다)을 추가하고, Herdr를 `poke` 레이아웃으로 전환한다.
4. 새 zsh를 연다. Herdr는 Dock에서 시작하고, Claude 세션 안에서 시작하지 않는다(Claude에서 실행한 Herdr는
   `CLAUDE_CODE_CHILD_SESSION`을 터미널에 넘겨 그 transcript 저장이 멈춘다).

키(레이아웃 `poke`, `base`와 동일): `⌘J` 나를 기다리는 다음 세션(오래된 순), `⌘K` 오른쪽 패널 표시/숨김,
`⌘⇧O` Obsidian 탭, `⌃1–9` space.

## 되돌리기

`scripts/poke/uninstall.sh`는 가장 최근의 전체 파일 백업이 아니라 설정 journal과 설치 파일 journal을 사용한다.
원래 설정과 소유한 파일을 복원하며, 이후 사용자가 교체한 파일과 journal에 없는 레거시 파일은 보존한다.
현재 레이아웃, 저장소, 에셋, 런타임 상태, 백업은 그대로 남는다. 수동으로 둔 보안 기본값과 외부 리뷰 스킬은
제거되지 않는다. 복구와 이전의 세부 사항은 `docs/guide.md`를 참고한다.

## 문제 해결

- `herdr-app status` — 레이아웃과 모드의 오류.
- 레이아웃이나 모드 이름 옆의 `*.error` 파일에 로드에 실패한 내용이 적혀 있다.
- 창을 쓸 수 없으면 `ctrl-alt-shift-escape`로 Herdr를 기본 레이아웃으로 초기화한다.
- `herdr-app check mods/poke`와 `herdr-app check mods/base` — 리로드 없이 모드를 컴파일한다.
- 헬퍼 상태는 `~/.local/state/herdr-poke/`에 있고, UI가 읽는 사용량 데이터는
  `~/.local/state/herdr-app/usage/`(`claude.json`, `codex.json`, `daily.json`)에 있다.
