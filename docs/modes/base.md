# Base — 변경하는 사람(사람 또는 LLM)을 위한 가이드

저장소 규칙은 먼저 `docs/guide.md`를 읽는다. Base는 중립 계층으로, 사이드바, 탭 바, pane 헤더, 하단 사용량 바,
오른쪽 패널을 담당한다. Poke 코드는 필요하지 않다. 설계 기록은
`docs/specs/2026-11-04-layer-separation-design.md`이다.

## 설치

```sh
python3 scripts/setup.py install base            # or: npm run setup:install -- base
python3 scripts/setup.py install base --select base
```

`mods/base/`, `layouts/base.json`, `themes/base.json`을 복사하고(저장소가 이미 `~/.config/herdr-app`이면 복사하지
않음) `layout:base` 키 바인딩을 추가한다. `~/.zshrc`, `programs.json`, `settings.json`은 수정하지 않는다.

## 위치별 역할

| 바꾸려는 것 | 볼 곳 |
|---|---|
| 창 구조, 크기, 어떤 view가 어디에 가는지 | `layouts/base.json` |
| 색 | `themes/base.json` |
| 이 레이아웃의 키 | `keys.json`, 섹션 `layout:base` |
| 블록이 보여 주는 view | `mods/base/main.js` (`props.view` → 모듈) |
| 사이드바 상단 (`props.logo`로 지정하는 선택적 로고, 검색, 접기 버튼) | `mods/base/views/brand.js` |
| 사이드바 space와 agent | `mods/base/views/spaces.js`, `mods/base/views/agents.js` |
| 탭 바와 패널 토글 | `mods/base/views/tabs.js` |
| pane 헤더 | `mods/base/views/pane.js` |
| 오른쪽 패널 헤더 탭 (레이아웃별로 `props.tabs`에 선언) | `mods/base/views/paneltabs.js`, `mods/base/lib/panel.js` |
| Usage 탭 (한도 카드와 현재 속도 예측, 14일 차트) | `mods/base/views/panel.js`, `mods/base/lib/usage.js` |
| 하단 바 (한도, 오늘의 토큰, jump pill)와 키 command. 패널이 Usage를 보여 주는 동안 한도는 생략 | `mods/base/views/status.js`, `mods/base/behaviours.js` |
| 사용량 파일 읽기 | `mods/base/views/usagefiles.js`, `mods/base/lib/contract.js` |
| agent 행 (Herdr 데이터 → view가 그리는 것) | `mods/base/views/data.js` |
| 아바타와 pill | `mods/base/views/ui.js`, `mods/base/lib/avatar.js` |
| 순수 로직 (긴급도, 그룹, chip, 부제, 포맷) | `mods/base/lib/`, 테스트는 `tests/base/mod-lib/` |
| 번들된 UI 폰트 | `mods/base/fonts/` |

레이아웃 id: `sidebar`, `brand`, `spaces`, `agents`, `footer`, `work`, `tabs`, `work-body`,
`main`, `panel` (`panel-tabs`, `panel-body`에 `usage`, `changes`, `files`, `obsidian`), `status`.
Command (표시 중인 복사본에 닿으며, 하단 바 복사본이 이를 실행한다): `base.jump` (나를 기다리는 다음 agent를
오래된 순으로 포커스)와 `base.notes` (패널을 Obsidian 탭으로 표시).

## 사용량 데이터 계약

UI는 이 파일들을 읽기만 한다. 각 파일의 writer는 정확히 하나다.

| `~/.local/state/herdr-app/usage/`의 파일 | 내용 | Writer |
|---|---|---|
| `claude.json` | 5h(300)와 주간(10080) 윈도우의 `{plan, windows:[{minutes, used, resets_at}], updated}` | 전역 Claude statusline (`~/.claude/statusline`, 이 저장소 밖) |
| `codex.json` | 같은 형태 | `herdr-poke codex` 래퍼 (`helpers/poke/internal/codex`) |
| `daily.json` | `{days:[{date, claude, codex}], updated}`, 로컬 날짜 14일, 오래된 순 | `herdr-poke usage` (launchd, 10분마다) |

모든 reader와 writer가 지키는 규칙: `updated`와 `resets_at`은 epoch 초이고, `resets_at`이 지난 윈도우는
퍼센트로 표시하지 않으며, 오래된 스냅샷은 최신 스냅샷을 대체하지 않고, 파일이 없거나 비었거나 손상된 경우는
오류가 아니라 "데이터 없음"으로 취급한다. 폴더는 `HERDR_USAGE_STATE`로 덮어쓸 수 있다(테스트용).
`HERDR_POKE_STATE`는 헬퍼의 비공개 상태만 옮기며 계약 폴더는 옮기지 않는다(launchd 작업은 이 값을 상시 export한다).

### Claude statusline 변경

Claude의 rate limit은 statusline의 stdin으로만 들어오므로, `~/.claude/statusline/main.go`가 한 줄을 출력한 뒤
이를 기록한다. 최대 300 ms, 오류는 무시하며, 출력 텍스트는 바뀌지 않는다. 파일에 더 새로운 스냅샷이 있거나
바뀐 것이 없으면 쓰기를 건너뛰고, 같은 reset 시각에는 더 큰 `used`를 유지한다. 이전 소스와 바이너리는
`main.go.bak-layer-separation`과 `statusline.bak-layer-separation`으로 보관되어 있다. 그 파일 헤더의 명령으로
다시 빌드하고, 같은 위치에서 `go test main.go main_test.go`로 테스트한다.

### Poke 없이 쓸 때의 의존성

- Claude 수치에는 위의 statusline 변경만 있으면 된다.
- Codex 수치, 일별 사용량, Codex 모델/effort/context label에는 `herdr-poke` 헬퍼(`~/.local/bin/herdr-poke`)가
  필요하다. 없으면 해당 부분만 비고 나머지는 모두 동작한다.
- Claude agent 행에는 더 이상 모델, effort, context %가 표시되지 않는다. Codex 행에는 여전히 표시된다.

## 페르소나 접점

Base는 agent 아바타와 탭 타입 pill을 그리며, 무언가 제공하면 아래 키를 읽는다.
아무것도 제공하지 않으면(Poke가 없으면) 단순한 글자 아바타를 보여 주고 타입 pill은 표시하지 않는다.

- 공유 상태: `poke.sprites` (파일 이름), `poke.evening`, `poke.typeMode` (기본값 off),
  `poke.typeSeed`.
- 터미널 label: `poke.partner`, `poke.review_of`, `poke.findings`, 그리고 telemetry label
  `poke.owner|provider|model|effort|ctx` (Codex 전용).

`mods/base`, `layouts/base.json`, `themes/base.json`이 이 목록 밖의 Poke 이름을 언급하면
`tests/base/mod-lib/layouts.test.mjs`가 실패한다.

## 화면 규칙

- 보조 텍스트 색 `muted`·`dim`은 배경과 chrome 위에서 4.5:1 이상을 유지한다(`tests/base/mod-lib/theme.test.mjs`).
- 막대는 항상 provider 색을 유지한다. 한도가 60% 이상이면 퍼센트 숫자를 accent로, 80% 이상이면 blocked 색과 `⚠`로 표시한다(패널과 하단 바 공통).
- 창이 80%를 넘거나 리셋 전에 100%에 닿을 것으로 예측되면 `notify`로 한 번 알린다. 키에 `resets_at`이 들어 있어 새 창마다 다시 알릴 수 있다(`limitAlerts`, 하단 바가 30초마다 확인).
- 끝난 agent가 있으면 하단 바에 "Review new" 버튼이 생기고, 가장 오래된 것을 연다.
- 단축키가 있는 컨트롤(패널 토글 ⌘K, jump ⌘J, Obsidian 탭 ⌘⇧O)은 툴팁으로 단축키를 보여 준다. 14일 막대는 툴팁으로 날짜와 수치를 보여 준다.
- 접힌 사이드바에서도 idle 얼굴은 `+N` 하나로 접힌다.
- 현재 속도 예측은 창의 10%가 지난 뒤에만 보이며, 리셋 전에 100%에 닿을 때만 "limit in …"으로 경고한다.
- 14일 평균은 사용량이 있는 날만으로 계산한다.
- 기본 번호 이름("1", "2")의 탭과 agent 행은 세션 제목, 없으면 스페이스 이름과 다른 폴더 이름으로 표시한다.
- Idle agent 그룹은 기본으로 접혀 있다.
- provider 색은 아이콘을 따른다(Claude `#d97757`, Codex `#7a9dff`, `mods/base/views/ui.js`의 `PROV`).
- 하단 바는 agent 요약(대기·작업 중·새 결과)을 보여 준다. 스페이스 행에는 idle이 아닌 agent 얼굴만 보이고,
  보조 줄은 브랜치, 없으면 agent 수다.
- 사이드바에서 agent 목록은 스페이스 목록의 두 배 높이를 받는다(레이아웃 `weight` 1:2).

## 변경을 검증하는 방법

1. `herdr-app check mods/base`가 `check passed`를 출력하고, `herdr-app status`가 `"errors": []`를 보여 준다.
2. `npm test` (레이아웃 동등성, 키 검사, 접점 가드, 문서 검사 포함).
3. 실제 확인: `herdr-app do layout.switch '{"name":"base"}'`를 실행한다. 하단 바와 Usage 탭에 두 provider가
   모두 표시되고, Claude 프롬프트 후 `claude.json`의 `updated`가 증가한다. Poke 관련 항목을 모두 중지한 상태에서도
   사이드바, 탭, 사용량, 노트가 여전히 동작하는지 확인한다.
4. 사용자의 다른 터미널은 절대 건드리지 않는다.
