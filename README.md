# Herdr

제 Herdr 작업 환경입니다. 커스텀 레이아웃, mod, 테마와 이를 지원하는 도구가 들어 있습니다.
따로 설치할 수 있는 **setup** 단위로 나뉘어 있습니다.

- **base**: 중립적인 기본 셸(사이드바, 탭 바, 사용량 바, 오른쪽 패널)
- **obsidian**: Obsidian 볼트 리더
- **poke**: base 위에 포켓몬 파트너와 세션 상태를 보여 주는 정원 미니맵을 얹는 페르소나

## 구조

```text
layouts/<name>.json       Herdr가 불러오는 윈도우 레이아웃
mods/<name>/              Herdr가 불러오는 UI mod
themes/<name>.json        Herdr가 불러오는 테마
helpers/<name>/           선택 설치하는 헬퍼 프로그램
scripts/setups.json       매니페스트: setup마다 필요한 리소스
scripts/setup.py          선택 설치 도구 (list / install / status)
scripts/<name>/           setup 전용 빌드·설치·제거 도구
tests/<name>/             setup 전용 테스트
docs/modes/<name>.md      setup 전용 가이드
keys.json                 공용 키 바인딩 (setup의 키는 `layout:<name>` 섹션에 둠)
programs.json             공용 터미널 실행 명령
settings.json             공용 앱 설정
```

레이아웃, mod, 테마는 서로 별개인 Herdr 리소스입니다. setup은 여러 mod를 조합하거나 기존 테마를
재사용할 수 있고, 리소스를 종류별로 하나씩 가질 필요는 없습니다.

## Setup 목록

| Setup | Layout | Mod | Theme | Requires | 가이드 |
|---|---|---|---|---|---|
| **base** | `layouts/base.json` | `mods/base/` | `themes/base.json` | 없음 | [Base](docs/modes/base.md) |
| **obsidian** | `layouts/obsidian.json` | `mods/obsidian/` | `themes/base.json` | 없음 (패널 탭은 base 필요) | [Obsidian](docs/modes/obsidian.md) |
| **poke** | `layouts/poke.json` | `mods/poke/` | `themes/poke.json` | base | [개발](docs/modes/poke.md) · [설치](docs/modes/poke-setup.md) |

설치할 때 건드리는 것:

- **base**: 자기 mod, 레이아웃, 테마 파일과 `layout:base` 키 바인딩
  (`cmd-j` 대기 중인 에이전트로 이동, `cmd-shift-j` 끝난 에이전트 확인, `cmd-k` 오른쪽 패널 열기/닫기,
  `cmd-shift-o` 노트).
  Claude 한도는 전역 Claude statusline이 채우고, Codex 한도와 일별 사용량은 `herdr-poke` 헬퍼가
  설치돼 있을 때 채워집니다. 자세한 내용은 Base 가이드에 있습니다.
- **obsidian**: 자기 mod, 레이아웃과 공용 base 테마. 그 외에는 건드리지 않습니다.
- **poke**: base에 더해 페르소나 mod와 레이아웃, `herdr-poke` 헬퍼, launchd 사용량 작업,
  `codex` 셸 alias, 관리되는 PATH 한 줄, Herdr 사운드 설정. 셸과 앱 설정을 수정하는 유일한
  setup이므로 설치 가이드를 먼저 읽어 주세요.

## Poké Garden

Poke 탭 하단의 정원 미니맵에서 모든 space의 에이전트 세션을 한눈에 볼 수 있습니다.
같은 포켓몬이라도 세션마다 한 마리씩 표시하며, 상태가 바뀌면 말풍선과 상태명이 바뀝니다.

- Active 옆 **Garden: On / Off**로 미니맵을 표시하거나 숨깁니다. 기본값은 On이며 선택은 저장됩니다.
- 미니맵이나 **Expand**를 클릭하면 최대 1,000×700px의 정원 창이 열립니다. 많은 세션은 스크롤해서 봅니다.
- 큰 창에서 포켓몬에 마우스를 올리면 세션 상세가 보이고, 클릭하면 해당 터미널로 이동합니다.
- **Close ×**로 창을 닫아도 미니맵은 남습니다. 미니맵을 꺼도 이미 열린 창은 유지됩니다.

Working은 작업 중, Needs you는 입력 대기, Done은 아직 확인하지 않은 완료, Idle은 다음 작업 대기입니다.
추가 설치 없이 Poke에 포함되며, 동작과 구현 위치는 [Poke 가이드](docs/modes/poke.md#poké-garden)에 있습니다.

## 다른 사람을 위한 빠른 시작

Herdr를 설치하고 한 번 실행해서 `~/.config/herdr-app`이 생긴 상태여야 합니다. 설치 도구와 테스트용으로
Python 3와 Node.js도 필요합니다. 설치 도구는 덮어쓸 파일을 먼저 백업하고, 이미 있는 키 바인딩은 바꾸지 않습니다.

```sh
git clone https://github.com/keoneokeoneo/herdr.git && cd herdr
npm run setup:list                        # 어떤 setup이 있고 무엇이 필요한지
npm run setup:install -- base --select base
```

이렇게 하면 기본 셸(사이드바, 탭 바, 사용량 바, 오른쪽 패널)이 설치됩니다. 필요한 것만 더 추가하세요.

| 원하는 것 | 실행 | 필요한 것 |
|---|---|---|
| 볼트 리더 | `npm run setup:install -- obsidian` | Obsidian 앱과 볼트 |
| 포켓몬 파트너 | `npm run setup:install -- poke` | macOS, Go. `~/.zshrc`, `programs.json`, `settings.json`도 수정하므로 [설치 가이드](docs/modes/poke-setup.md)를 먼저 읽을 것 |

사용량 파일을 써 주는 도구가 없으면 사용량 바는 비어 있습니다. UI는 파일을 읽기만 하므로 어떤 도구로든
채울 수 있습니다. 파일은 `~/.local/state/herdr-app/usage/`에 있는 JSON 문서입니다.

```json
{ "plan": "pro", "windows": [
    { "minutes": 300,   "used": 13, "resets_at": 1793791800 },
    { "minutes": 10080, "used": 27, "resets_at": 1794121200 } ],
  "updated": 1793775748 }
```

파일 이름은 `claude.json` 또는 `codex.json`입니다. `used`는 퍼센트, `resets_at`과 `updated`는 epoch 초이고,
`300`은 5시간 창, `10080`은 주간 창입니다. 리셋 시각이 지난 창은 "–"로 표시됩니다. `daily.json`에는
일별 토큰 합계가 들어갑니다. 정확한 규칙은 [Base 가이드](docs/modes/base.md)에 있습니다.
Poke 헬퍼가 없으면 Codex 한도와 일별 합계는 직접 파일을 써 주지 않는 한 비어 있습니다.

setup을 제거하려면 `~/.config/herdr-app`에서 해당 mod, 레이아웃, 테마 파일을 지우고 레이아웃을
전환하세요(`Cmd-Shift-L`). 설치 도구가 기록한 내용은 `setup:status`로 볼 수 있습니다.

개인 설정 모음이라 그대로 쓰기보다 참고해서 자기 환경에 맞게 고쳐 쓰는 것을 권장합니다.

## 설치할 것 고르기

```sh
npm run setup:list                               # setup 목록, 필요한 것, 설치 여부
npm run setup:install -- base obsidian           # 이것들만 (필요한 것과 함께) 설치
npm run setup:install -- base --select base      # 설치 후 Herdr를 해당 레이아웃으로 전환
npm run setup:update                             # 이미 설치한 setup을 지금 체크아웃으로 다시 복사 (git pull 뒤)
                                                 # 설치 뒤 바깥에서 바뀐 파일은 건너뛰고 알림 (--force로 교체)
npm run setup:status                             # 설치된 setup과 파일 변경 여부
```

`setup:install`은 선택한 리소스만 `~/.config/herdr-app`에 복사합니다(이 체크아웃이 이미 그 폴더이면
복사하지 않습니다). 빠진 키 바인딩만 추가하고(이미 있는 다른 바인딩은 유지하고 알려 줍니다), 교체할
파일은 `~/.local/state/herdr-app/backups/<시각>/`에 백업하며(mod 폴더 안에 쓰면 리로드되므로), 한 일을 기록합니다. `poke`를 설치하면 `base`도 함께 설치되고, poke 전용 설치기는 마지막에
실행됩니다. `bash scripts/poke/install.sh`는 `poke`를 설치하는 것과 같습니다.

## 개발

규칙과 setup 추가 방법은 [레포 가이드](docs/guide.md)에, Herdr가 mod와 레이아웃을 불러오는 방식과
변경 후 확인할 것은 [Herdr 커스터마이징](docs/herdr-dev.md)에 있습니다.
이 레포는 Herdr의 기본 디렉터리 구조를 그대로 유지해서 `~/.config/herdr-app`에 둘 수 있습니다.
다른 위치의 체크아웃은 소스일 뿐 설치된 Herdr 설정이 아니므로, 위 명령으로 원하는 setup을 설치하세요.

테스트:

```sh
npm test                        # 모든 setup의 JavaScript 테스트 (문서 검사 포함)
npm run test:setup              # 선택 설치 도구
npm run test:poke:helper        # Go 헬퍼
npm run test:poke:scripts       # Poke 설치 스크립트
npm run preflight:poke          # Poke 설치 전 읽기 전용 점검
```

`herdr-app check mods/<name>`은 mod를 다시 불러오지 않고 컴파일만 확인합니다.

헬퍼 바이너리, 상태 디렉터리, launchd 작업은 `herdr-poke` 이름을 그대로 씁니다. UI가 읽는 사용량
데이터는 `~/.local/state/herdr-app/usage/`에 있습니다. 로컬 체크아웃과 GitHub 레포 이름은 `herdr`이고,
`origin` 리모트는 `git@github.com:keoneokeoneo/herdr.git`입니다.

## 라이선스

코드는 [MIT](LICENSE)입니다. `mods/base/fonts/`의 IBM Plex 글꼴은
[SIL Open Font License 1.1](mods/base/fonts/OFL.txt)을 따릅니다. Pokémon은 Nintendo, Creature Inc.,
GAME FREAK inc.의 상표이며, 이 프로젝트는 이들과 관련이 없는 비공식 개인 프로젝트입니다. 레포에는
포켓몬 스프라이트나 울음소리 파일이 들어 있지 않습니다.
