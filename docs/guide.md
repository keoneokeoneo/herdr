# Personal Herdr — 저장소 가이드

이 저장소는 여러 레이아웃, 모드, 테마로 이루어진 개인 Herdr 설정을 관리하며, 설정은 독립된 **setup** 단위로 나뉜다.
[Base](modes/base.md)(중립적인 사이드바, 탭 바, 사용량 바, 오른쪽 패널), [Obsidian](modes/obsidian.md)(vault 리더),
Poke([개발](modes/poke.md), [설치](modes/poke-setup.md); Base 위에 얹는 Pokémon 파트너)가 있다.
setup의 구현을 바꾸기 전에 해당 setup의 가이드를 읽고, Herdr가 이 파일들을 로드하는 방식과 함정, 변경 확인 방법은
[Customizing Herdr](herdr-dev.md)를 읽는다.

## 위치별 역할

| 역할 | 위치 |
|---|---|
| Herdr 네이티브 리소스 | `layouts/<name>.json`, `mods/<name>/`, `themes/<name>.json` |
| setup이 필요로 하는 리소스와 선행 조건 | `scripts/setups.json` |
| 선택적 설치기 | `scripts/setup.py` (`npm run setup:list`, `setup:install`, `setup:status`) |
| 선택적 setup 헬퍼 | `helpers/<name>/` (Poke의 헬퍼는 독립된 Go 모듈) |
| setup 설치 도구 | `scripts/<name>/` |
| JavaScript 및 설치기 테스트 | `tests/<name>/mod-lib/`, `tests/<name>/scripts/`, `tests/setup/` |
| Herdr 사용법과 함정 | `docs/herdr-dev.md` |
| setup 가이드 | `docs/modes/<name>.md` |
| 설계 기록과 계획 | `docs/specs/`, `docs/plans/` (날짜별 이력이며, 현행 문서는 가이드) |
| 공용 키보드, 런처, 앱 설정 | `keys.json`, `programs.json`, `settings.json` |

루트는 Herdr의 설정 홈(`~/.config/herdr-app`)과 일치한다. 네이티브 리소스는 위 디렉터리에 두고,
별도의 `modes/` 폴더 아래에 중첩하지 않는다.
여기서 "setup"이나 "mode"는 조직 차원의 개념일 뿐 Herdr API가 추가된 것이 아니다.
레이아웃은 view를 배치하고, 모드는 view와 command를 구현하며, 테마는 색을 제공한다.
모드는 서로 분리된 폴더라 서로 import할 수 없으므로, 공용 동작은 공유 상태, 터미널 label, command,
데이터 파일 계약을 통해 전달한다. 공용 구현은 여러 setup이 실제로 필요로 할 때만 분리한다.

## 계층

- **Data**: `~/.local/state/herdr-app/usage/`의 사용량 데이터 계약(`claude.json`, `codex.json`,
  `daily.json`). 파일마다 writer는 하나이며 `docs/modes/base.md`에 설명되어 있다.
- **Base** (`mods/base`): 계약을 읽어 셸을 그린다. 다른 것이 실행 중이지 않아도 동작한다.
- **Poke** (`mods/poke`): 페르소나. 문서화된 접점(`docs/modes/base.md`에 나열된 공유 키와 터미널 label)으로만
  base에 접근하며, 테스트가 그 목록을 지킨다.
- **Obsidian** (`mods/obsidian`): 독립적이다. base가 설치되어 있으면 base의 패널에 임베드된다.

## setup 추가

1. 리소스와 command/state 네임스페이스에 쓸 고유한 이름을 정한다.
2. 필요한 레이아웃, 모드, 테마를 네이티브 디렉터리에 추가한다. 쓸 만한 곳에서는 기본 제공(shipped) 모드를
   재사용한다. Herdr는 시작할 때 shipped 파일을 복원하므로 수정하지 않는다.
3. setup을 `scripts/setups.json`에 추가한다(모드, 레이아웃, 테마, 필요한 setup, `layout:<name>` 키 섹션,
   셸이나 앱 설정을 수정할 때만 `installer`).
4. 헬퍼나 설치기는 setup에 필요할 때만 해당 이름의 디렉터리 아래에 추가한다.
   런타임 에셋과 상태는 `mods/` 바깥에 둔다. 그곳에 쓰면 리로드가 발생한다.
5. `tests/<name>/` 아래에 테스트를 추가하고, 헬퍼 테스트는 소스 옆에 둔다. `npm test`는
   `tests/*/mod-lib/*.test.mjs`를 찾아 실행하므로, 그 외 테스트 스위트와 setup 전용 설치에는
   package 스크립트를 명시적으로 추가한다.
6. 의존성, 리소스 이름, 설치 효과, 런타임 저장소, 검증 방법을 `docs/modes/<name>.md`에 문서화한 뒤
   README의 setup 표에 추가한다
   (표, 가이드, 문서 내 경로가 낡았으면 `tests/base/mod-lib/docs-current.test.mjs`가 실패한다).
7. 변경한 모드는 `herdr-app check mods/<name>`으로 확인한다. 설치된 setup을 테스트할 때는 접을 수 있는
   영역을 포함해 `herdr-app status`와 UI를 확인한다.

## 공용 설정과 키 소유권

루트 설정은 모든 레이아웃에 영향을 주므로, 각 setup은 자신의 바인딩을 `keys.json`의 `layout:<name>`
섹션에 둔다(해당 레이아웃이 표시되는 동안 Herdr가 그 섹션을 적용한다). `global`에는 일반 바인딩만 둔다.
선택적 설치기는 setup에 빠진 바인딩만 추가하고, 다른 바인딩을 덮어쓰지 않는다.

`programs.json`은 현재 Codex를 `herdr-poke`를 거쳐 실행하고, 앱 사운드는 Poke의 울음소리를 위해 꺼져 있다.
이 두 파일과 `~/.zshrc`는 `poke` setup을 설치할 때만 수정된다. 다른 레이아웃을 선택해도 이 변경은 되돌아가지
않는다. Poke는 전역 Claude `statusLine`을 수정하지 않는다. Claude의 한도는 전역 statusline이 기록한다
(`docs/modes/base.md` 참고).

새 setup마다 이 기본값을 재사용할지 바꿀지 명시적으로 결정한다. Poke의 설치기를 일반적인 선행 조건인 것처럼
복사하지 않는다. 설치 중 공용 설정을 수정하기 전에 백업하고, 설치/제거 소유권 journal을 보존한다.

## 개발 규칙

- `AGENTS.override.md`와 `CLAUDE.md`가 에이전트를 여기로 안내한다. 기존 `herdr-poke-dev` 스킬은 Poke setup과
  그 헬퍼를 다룬다.
- Herdr는 설치된 홈에 `AGENTS.md`, `herdr.d.ts`, `gpui-kit.d.ts`를 생성한다. API 세부 사항은 그 파일에서
  검색하고, 생성된 참조 파일은 수정하지 않는다. 소스만 있는 체크아웃에는 없을 수 있다.
- `.gitignore`는 생성 파일, shipped 리소스, 런타임 데이터를 제외한다. 새 setup이 런타임 에셋을 도입하면
  확장하되, 직접 만든 소스 파일은 제외하지 않는다.
- 다른 도구가 다른 위치를 제안하더라도 문서는 `docs/modes/`, `docs/specs/`, `docs/plans/`에 둔다.
  문서에 커밋 해시를 넣지 않는다.
- 커밋 제목은 저장소 이력을 따라 `type(scope): 한국어 요약` 형식으로 쓰고, scope는 해당 setup 또는
  저장소로 한다. (`~/Documents/work/commit-convention.md`는 `~/Documents/work` 아래 작업에만 적용된다.)
  커밋과 PR에 AI 서명을 넣지 않는다.
- 관련 없는 사용자 터미널은 절대 건드리지 않는다. Poke 고유의 시각·통합 규칙은 해당 가이드에 있으며,
  앞으로 만들 모든 setup의 요구 사항은 아니다.

## 경로 이전

이전의 `helper/`, `scripts/*`, `tests/{mod-lib,scripts}/`는 이제 `helpers/poke/`, `scripts/poke/*`,
`tests/poke/{mod-lib,scripts}/`에 있다. 셸과 사용량 view는 `mods/poke`에서 `mods/base`로 옮겨졌고,
레이아웃 id에서 `poke-` 접두사가 빠졌으며, 사용량 파일은 `~/.local/state/herdr-app/usage/`로 옮겨졌다.
이름이 정해진 npm 명령이나 새 경로를 사용한다. Poke의 바이너리 이름, Go 모듈, label, 에셋 경로, 헬퍼 상태
경로, launchd 식별자는 그대로이므로 헬퍼를 다시 설치할 필요가 없다. 로컬 체크아웃과 GitHub 저장소의 이름은
`herdr`이며, `origin`은 `git@github.com:keoneokeoneo/herdr.git`을 가리킨다.
