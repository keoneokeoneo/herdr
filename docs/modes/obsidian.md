# Obsidian 리더

로컬 Markdown vault용 읽기 전용 네이티브 Herdr 모드. macOS, Python 3, `TextView.markdown`을 지원하는 Herdr가
필요하다. Obsidian이 실행 중일 필요는 없다.

## Base 통합 (권장)

```sh
python3 scripts/setup.py install base obsidian
herdr-app do layout.switch '{"name":"base"}'
```

오른쪽은 동등한 탭을 가진 하나의 패널이다: **Usage / Changes / Files / Obsidian** (Poke의 레이아웃은 **Poke**
탭을 추가한다). 한 번에 한 탭의 내용만 보인다. 오른쪽 위 패널 버튼 또는 **Cmd-K**로 패널 전체를 숨기거나 표시하고,
**Cmd-Shift-O**로 Obsidian 탭을 연다. 리더의 **×**는 패널 전체를 닫는다. 같은 레이아웃 안에서는 탭을 전환하거나
닫아도 현재 노트, 검색, 펼친 폴더가 유지된다.

리더는 vault 이름, 접을 수 있는 폴더 트리, 전체 경로 없는 노트 제목을 보여 준다. 폴더를 펼쳐 탐색하고, 검색하면
일치하는 경로가 펼쳐진다. 검색어를 비우면 처음 200개 노트가 아니라 색인된 트리 전체가 복원된다.
**Open in Obsidian**은 [공식 URI](https://obsidian.md/help/uri)를 사용해 현재 노트를 데스크톱 앱에서 연다.
클릭하기 전에는 앱을 실행하지 않는다.

키는 `keys.json`의 `layout:base`와 `layout:poke` 섹션에 있다. 설치기는 빠진 바인딩만 추가하고, 이미 있는 다른
바인딩은 유지하며 보고한다. 패널과 레이아웃 id는 [Base](base.md)를 참고한다. Obsidian은 Poke가 필요하지 않다.

## 단독 레이아웃

```sh
python3 scripts/setup.py install obsidian   # or: bash scripts/obsidian/install.sh
herdr-app do layout.switch '{"name":"obsidian"}'
```

이 레이아웃은 활성 터미널 탭을 리더 옆에 배치하며, 구분선을 끌어 크기를 조절한다. **Vault**로 Obsidian의 macOS
레지스트리에서 찾은 vault를 고르거나, 절대 폴더 경로(`~/...`도 가능)를 입력하고 **Use folder**를 선택한다.
이렇게 하면 소스를 바꾸지 않고도 나중에 개인용 vault를 지원할 수 있다. 수동으로 추가한 vault와 마지막으로 선택한
vault는 Herdr 공유 상태의 `obsidian.vaults`와 `obsidian.root`에 유지된다. 레지스트리와 vault에는 절대 쓰지 않는다.

제목, 상대 경로, 본문 검색어를 입력하고 Enter 또는 **Search**를 누른다. 검색어를 비우면 폴더 트리를 탐색한다.
노트를 클릭하면 Markdown을 읽고, 위키 링크나 상대 Markdown 링크를 따라갈 수 있으며, **←**로 돌아가고,
**Refresh**로 변경된 노트를 다시 불러온다. **Search**는 파일 목록도 새로 고친다. HTTP(S) 링크는 기본 브라우저에서
열린다. **Back**은 `base` 레이아웃으로 전환한다(`base`가 설치되어 있지 않으면 다음 레이아웃으로). Cmd-Shift-L은
레이아웃을 순환한다.
이 모드는 키보드나 CLI 연동을 위해 `herdr-app do`로 `obsidian.search`, `obsidian.vault:<folder>`,
`obsidian.note:<relative path>` command를 받는다. `pane.go`는 검색 입력에 포커스를 준다.
이 모드는 다른 레이아웃에서도 `{"mod":"obsidian"}`으로 사용할 수 있다.

## 한계

이것은 Markdown 리더이지 Obsidian 애플리케이션이 아니다. 플러그인, Dataview, Canvas, 그래프 뷰, 첨부 임베드,
편집은 구현되어 있지 않다. 위키 임베드는 탐색 링크로 표시되고, 제목/블록 링크는 앵커로 스크롤하지 않고 노트만
연다. 중복되는 짧은 링크 이름은 더 구체적인 경로가 필요하다. 숨김 디렉터리와 심볼릭 링크는 색인 중에 제외된다.
읽기는 선택한 vault 안의 Markdown 파일로 제한된다. 1 MiB를 넘는 파일은 거부되고, 트리는 최대 20,000개 노트를
색인하며, 비어 있지 않은 검색은 최대 200개 결과를 표시한다(파일 이름 일치가 먼저). 검색은 키 입력마다가 아니라
제출할 때 실행된다.

설치기(`scripts/setup.py install obsidian`)는 이 모드, 그 레이아웃, 공용 `themes/base.json`만 복사하며, 변경된
기존 파일은 옆에 백업한다. 키, 셸 설정, launchd, Obsidian 설정, 현재 레이아웃은 바꾸지 않으며 Base도 Poke도
필요하지 않다. 런타임 상태는 Herdr의 공유 저장소에 있다. 제거하려면 다른 레이아웃으로 전환한 뒤 앱 홈에서
`mods/obsidian`과 `layouts/obsidian.json`을 삭제하고, 공용 base 테마는 남겨 둔다.

## 검증

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/obsidian/scripts
npm test
herdr-app check mods/obsidian
herdr-app status
```

실행 중인 UI도 확인한다: vault 선택, 검색, 노트 렌더링, 내부 링크, 뒤로 가기 탐색, 실패하는 폴더 경로.
모드 검사는 초기 상태만 렌더링하며 비동기 읽기는 검증하지 않는다.
