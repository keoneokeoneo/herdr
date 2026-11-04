# 이 저장소로 Herdr 커스터마이징하기

이 저장소로 Herdr 창을 바꾸는 방법을 사람과 코딩 에이전트를 위해 정리한다. 파일 지도는 먼저
[저장소 가이드](guide.md)를 읽는다. 이 문서는 Herdr 고유의 내용, 즉 각 부분의 동작 방식, 함정, 변경 확인 방법을 다룬다.

## Herdr가 이 파일들을 사용하는 방식

- **Layout** (`layouts/<name>.json`)은 view를 배치해 하나의 창을 만든다. `layout.switch {name}`이
  하나를 고르고, `shared("layout")`은 표시 중인 이름이다. 기본 제공 레이아웃(`main`, `terminals`, `win98`)은
  시작할 때 복원되므로, 수정하지 말고 복제해서 쓴다.
- **Mod** (`mods/<name>/`)는 view와 command를 구현한다. `main.js`가 View 클래스를 default export하고,
  레이아웃이 `props.view`로 view를 고른다. 모드는 서로 분리된 폴더라 서로 import할 수 없다.
  `shared(...)`/`setShared(...)`, 터미널 label, command(`<mod>.<command>`), 데이터 파일로 공유한다.
- **Theme** (`themes/<name>.json`)은 색을 담는다. **Keys** (`keys.json`)에는 `global`, `terminal`,
  `layout:<name>` 섹션이 있으며, setup은 자신의 바인딩을 `layout:<name>`에 둔다.
- Herdr 자체의 API 참조는 설치된 홈에 `AGENTS.md`, `herdr.d.ts`, `gpui-kit.d.ts`로 생성된다. 함수를 추측하기
  전에 이 파일에서 검색하고, 절대 수정하지 않는다(여기서는 git에서 제외되어 있다).

## 함정

- **핫 리로드:** Herdr는 `mods/<name>/` 안의 숨김이 아닌 파일이 기록될 때마다 모드를 리로드한다. 런타임 데이터는
  상태 폴더(`~/.local/state/...`)나 에셋 폴더에 두고, `mods/` 안에는 절대 두지 않는다. 그곳에 쓰면 QuickJS의
  메모리가 바닥날 때까지 리로드가 반복된다.
- **shipped 모드는 시작할 때마다 복원된다.** 수정한 모드는 `<name>-edited`로 옮겨진다. shipped 모드는 수정하지
  말고 복제하며, shipped 동작은 command로 재사용한다: `openOverlay({mod:"menu"})`, `mod:"new-menu"`,
  `layout.toggle`, `layout.resize`, `terminal.pin`, `app.palette`.
- **`setShared`는 모든 view를 다시 렌더링한다.** 값이 실제로 바뀐 경우에만 호출한다. `cx.notify()`는 그것을
  소유한 view를 다시 만들고 창을 다시 그린다. 숨겨진 view도 처음 표시된 뒤에는 타이머가 계속 돈다.
- **색:** `"transparent"`는 색이 아니다. `"#00000000"`을 쓴다.
- **macOS traffic lights:** `traffic_lights: [12, 10]`은 46px 브랜드 행의 가운데에 맞춘다. x≈80까지 차지하므로
  브랜드 행은 96px 패딩을 두고, 접힌 사이드바의 너비는 92px이다. 120px 미만이면 모든 사이드바 view가 rail 형태
  (`narrow()`)로 전환되어야 하므로, 사이드바 view를 건드릴 때는 두 너비를 모두 테스트한다.
- **Herdr는 Dock에서 시작한다.** Claude 셸에서는 절대 시작하지 않는다. 거기서 시작한 서버는
  `CLAUDE_CODE_CHILD_SESSION=1`을 상속해 그 안의 모든 Claude가 transcript 저장을 멈춘다.
  앱을 Dock에서 다시 실행해도 백그라운드의 `herdr-core` 서버와 터미널을 보관하는 `herdr-core hold` 프로세스는
  그대로 살아 있으므로 표시가 사라지지 않고, 새 탭과 터미널도 계속 물려받는다. 터미널을 모두 닫지 않고 고치려면
  `~/.zshrc`에 `[[ -o interactive && -n $HC_TERMINAL_ID ]] && unset CLAUDE_CODE_CHILD_SESSION`을 두고
  이미 실행 중인 Claude만 다시 시작한다. 상속 여부는 `ps eww -p <hold pid>`로 확인한다.
- **`hc` 서버의 PATH는 최소한이다**(nvm도 zsh alias도 없음). 따라서 `hc run`으로 시작하는 프로그램은 절대 경로나
  `~/.local/bin` shim이 필요하다.
- **사용자의 다른 터미널은 절대 건드리지 않는다.**

## 변경 확인

1. `herdr-app check mods/<name>`이 `check passed`를 출력한다(모드 전용이며, 레이아웃은 전환해서 확인한다).
2. `herdr-app status`가 `"errors": []`와 기대한 `layout`을 보여 준다.
3. `npm test`(모드 로직, 레이아웃, 키, 문서). 설치기는 `npm run test:setup`. 실패하는 테스트를 먼저 작성한다.
4. 눈으로 확인한다. `herdr-app shot out.png`는 traffic lights 없이 마지막 프레임을 저장한다. lights까지
   보려면 창 id를 찾아 `screencapture -x -o -l <id> out.png`를 쓴다. 캡처는 2배 크기이므로 보기 전에 자른다.
5. 사이드바 rail: `herdr-app do layout.resize '{"id":"sidebar","px":92}'`로 접고, `262`로 복원한다. 원래대로 돌려 놓는다.
6. 영향받는 각 레이아웃으로 `herdr-app do layout.switch '{"name":"<name>"}'`를 실행해 전환해 보고, 사용자의 레이아웃으로 되돌려 놓는다.

## setup 추가 또는 변경

[저장소 가이드](guide.md)의 "setup 추가"를 따른 뒤, 해당 setup의 페이지를 본다:
[Base](modes/base.md), [Obsidian](modes/obsidian.md), [Poke](modes/poke.md). setup 고유 규칙
(Poke의 GIF, Codex 미러, 스프라이트)은 여기가 아니라 그 페이지에 있다.
