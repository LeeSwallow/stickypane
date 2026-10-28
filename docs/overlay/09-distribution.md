# 09. 배포와 설치 화면

> 상태: Draft · 2026-10-29 · 관련 요구: R-30, R-31, R-43

## 9.1 산출물

[사용자 결정] 제품 이름에서 "overlay"를 뺀다. 앱 이름은 **stickypane**이다. 저장소의 `overlay/` 폴더와 Go 모듈 경로는 안쪽 이름이라 그대로 둔다.

| 플랫폼 | 산출물 | 만드는 법 |
|---|---|---|
| macOS (arm64, amd64) | `stickypane-<버전>-<arch>.dmg`, 볼륨 이름 `stickypane`, 안에 `Stickypane.app` | `wails3 package` → `scripts/overlay-dmg.sh`(hdiutil + 배경 + 아이콘 배치) |
| macOS | Homebrew cask | 같은 DMG를 가리킨다([미결정]: 이름과 본체 cask와의 관계) |
| Linux (amd64, arm64) | `stickypane-app-<버전>-linux-<arch>.tar.gz` (앱 + `stickypane` + `.desktop` + 아이콘 + `install.sh` + 라이선스(`LICENSE`, `THIRD_PARTY_NOTICES.md`, `licenses/`)) | Linux에서 `wails3 build` → `scripts/overlay-linux.sh`. WebKitGTK 4.1이 필요하다. 풀고 `sh install.sh`를 하면 `~/.local/lib/stickypane-app/`에 앱과 CLI, `~/.local/bin/stickypane-app` 링크, `~/.local/share/applications/stickypane.desktop`과 아이콘이 깔린다(`--uninstall`로 지운다) |

| 이름 | 값 |
|---|---|
| 번들 | `Stickypane.app`, `CFBundleName`·`CFBundleDisplayName` = `stickypane` |
| 번들 id | `io.github.leeswallow.stickypane` (단일 실행 UniqueID도 같다) |
| 앱 실행 파일 | `Contents/MacOS/stickypane-app`, Linux는 `stickypane-app` (CLI `stickypane`와 겹치지 않게) |
| `.desktop` | `Name=stickypane`, `Exec=stickypane-app %u`, `Icon=stickypane` |
| 버전 | 하나뿐: `overlay/build/config.yml`의 `info.version`. `Info.plist`·`Info.dev.plist`의 `CFBundleShortVersionString`·`CFBundleVersion`이 같은 값을 되풀이하고 `TestVersionsAgree`가 지킨다. `overlay-dmg.sh`는 `.app`의 버전이 다르면 멈춘다 |

[사용자 결정] **앱만 깔면 다 된다.** 앱 안에 `stickypane` 명령을 넣는다. macOS는 `Stickypane.app/Contents/Helpers/stickypane`, Linux는 앱 바이너리 옆 `stickypane`. 같은 체크아웃의 `./cmd/stickypane`을 앱과 같은 버전(`-X main.version=<CFBundleShortVersionString>`), 같은 아키텍처로 `overlay-dmg.sh`·`overlay-linux.sh`가 빌드해 넣는다. 앱은 bridge를 찾을 때 이 번들 사본을 **먼저** 보고(`bridge.BundledCLI()`), 그다음 `STICKYPANE_BIN`, PATH, `/opt/homebrew/bin`·`/usr/local/bin`·`~/go/bin` 순이다. 터미널에서 쓰도록 PATH에 링크를 거는 버튼은 따로 한다.

라이선스 전문도 앱과 함께 간다. macOS는 `Contents/Resources/`, Linux는 압축 폴더 맨 위에 `LICENSE`, `THIRD_PARTY_NOTICES.md`, `licenses/`를 둔다. `licenses/stickypane-app.txt`는 `overlay/scripts/licenses.sh --write`가 앱에 링크된 Go 모듈과 번들된 npm 패키지(글꼴의 OFL-1.1 포함)의 라이선스 파일을 그대로 모은 것이고, `licenses/stickypane.txt`는 `scripts/licenses.sh`가 번들 CLI의 모듈로 만든 것이다. robust-predicates(mermaid → d3-delaunay, Unlicense)도 목록에 그대로 둔다([미결정]: 뺄지).

본체 `stickypane`은 지금처럼 goreleaser로 배포한다. 앱은 같은 태그에 산출물을 더한다(CGO와 WebKit 때문에 macOS·Linux 러너에서 따로 빌드한다. 릴리스 워크플로 연결은 [미결정]). 그 전까지 `overlay/`나 `scripts/overlay-*.sh`를 건드린 PR마다 `.github/workflows/overlay.yml`의 `package` 작업이 macOS 러너에서 DMG를, Linux 러너에서 tar.gz를 만들어 워크플로 산출물(14일)로 올린다. 태그·릴리스·Homebrew에는 손대지 않는다. [설계자 판단] 앱은 Windows용을 만들지 않는다(R-30).

## 9.2 DMG 설치 화면

[사용자 결정] DMG 설치 화면도 doodle 양식이 드러나게 꾸민다(R-31). 앱 아이콘을 오른쪽 Applications 바로가기로 끌어 놓으면 설치된다.

![DMG 창](assets/dmg.png)

```text
┌──────────────────────────── stickypane ──────────────────────────────┐
│   (격자 노트, 테이프 붙은 손글씨 제목)                                  │
│                 stickypane                                            │
│             "끌어다 붙이면 끝!"                                         │
│                                                                       │
│     ┌─────────┐      ~~~~~~~~~~~~~~~~~~~~~~>      ┌─────────┐          │
│     │ [앱 아이콘]│    (손으로 그린 크레용 화살표)    │Applications│        │
│     └─────────┘                                   └─────────┘          │
│     stickypane                                     Applications        │
│                                                                       │
│   ┌ 노란 메모지 (살짝 기울어짐) ──────────────────────┐                  │
│   │ 처음 열 때 경고가 뜨면                             │                  │
│   │ 시스템 설정 → 개인정보 보호 및 보안 → '그래도 열기' │                  │
│   └────────────────────────────────────────────────┘                  │
└───────────────────────────────────────────────────────────────────────┘
```

| 요소 | 규칙 |
|---|---|
| 창 크기 | 내용 660×420 pt(창 틀은 제목 막대만큼 더 높다). 배경은 @1x 660×420과 @2x 1320×840을 `tiffutil -cathidpicheck`로 `background.tiff` 하나로 합친다 |
| 배경 | 밝은 종이(`stickypane-light`의 종이 `#fffdf7`) + 격자 노트. Finder 아이콘 이름 글자가 검은색이라 밝은 배경을 쓴다 |
| 제목 | "stickypane", Gaegu Bold, 테이프 두 조각. 아래에 "끌어다 붙이면 끝!"과 크레용 밑줄 |
| 화살표 | rough.js로 그린 크레용 화살표(아이콘의 빨강), 앱 아이콘에서 Applications로. seed를 고정해 늘 같은 획 |
| 아이콘 위치 | 앱 (170, 210), Applications 바로가기 (490, 210), 아이콘 크기 112, 글자 13 |
| 메모지 | 서명·공증이 없는 동안 Gatekeeper 안내: "처음 열 때 경고가 뜨면 / 시스템 설정 → 개인정보 보호 및 보안 → '그래도 열기'". macOS 15부터 우클릭 → 열기로는 넘어가지 않는다 |
| 볼륨 아이콘 | 앱 아이콘과 같은 `.icns`(`.VolumeIcon.icns` + `SetFile -a C`) |

배경 이미지는 손으로 그리지 않고 **코드로 만든다**: `overlay/build/dmg/background.html`(rough.js + Gaegu, 앱의 `frontend/node_modules`에 있는 자원만)을 `overlay/build/dmg/render.sh`가 헤드리스 Chrome으로 두 배율 PNG로 찍고 TIFF로 합친다. 그래서 테마·문구를 바꿔도 다시 만들 수 있다. 결과 PNG·TIFF는 저장소에 넣어 두어 DMG를 만들 때 Chrome이 필요 없다.

`scripts/overlay-dmg.sh <app> <out.dmg>`의 순서:

1. `.app`을 검사한다(Info.plist와 그 `CFBundleExecutable`, 버전이 `config.yml`과 같은지). 같은 이름의 볼륨이 이미 마운트돼 있으면 멈춘다.
2. 임시 폴더에 `Stickypane.app`으로 복사하고, `stickypane` CLI를 빌드해 `Contents/Helpers/`에 넣고(Apple 규칙상 `Resources`에는 실행 코드를 두지 않는다), 안쪽부터 ad-hoc 서명한다(CLI → 앱). 서명 전에 `Contents/Resources/`에 라이선스 파일(9.1)을 넣는다. `/Applications` 심볼릭 링크와 `.background/background.tiff`를 둔다.
3. `hdiutil create -format UDRW` → `hdiutil attach`(Finder가 볼 수 있게 `-nobrowse` 없이).
4. AppleScript로 그 디스크 이미지의 Finder 창만 정리한다: 아이콘 보기, 도구 막대·상태 막대 숨김, 창 경계, 아이콘 크기·위치, 배경. Finder가 `.DS_Store`를 쓸 때까지 기다린다.
5. `.VolumeIcon.icns`와 사용자 정의 아이콘 속성, `bless --openfolder`. Apple Silicon에서는 `bless --openfolder`가 지원되지 않아 실패할 수 있고, 그러면 경고만 내고 계속한다. 마운트하면 창이 저절로 열리는지는 [미검증]이다.
6. 분리 → `hdiutil convert -format UDZO`. `trap`이 어떤 경우에도 분리하고 임시 파일을 지운다.

CLI는 `-trimpath -buildvcs=false`로 빌드해 같은 체크아웃에서 같은 바이너리가 나온다. Linux 압축은 GNU tar일 때 이름 순서, 소유자·그룹 0, 시각 `SOURCE_DATE_EPOCH`(없으면 마지막 커밋 시각)로 묶어 재현된다. DMG는 그렇지 않다: HFS+ 이미지는 만든 시각과 볼륨 id를 담아 매번 비트가 다르다.

## 9.3 서명과 공증

- 지금 본체 바이너리도 공증하지 않는다(README의 cask가 quarantine을 지운다). 앱도 **ad-hoc 서명**(`codesign --force -s -`)만 한다. 번들 안의 `stickypane`을 먼저, 앱을 나중에 서명하고 `codesign --verify --deep --strict`로 확인한다.
- 내려받은 ad-hoc 앱을 처음 열면 macOS가 "확인할 수 없음" 경고를 띄운다. 사용자는 **시스템 설정 → 개인정보 보호 및 보안**에서 **그래도 열기**를 누른다(메모지 문구).
- Developer ID와 공증은 [미결정]. 하게 되면 DMG의 메모지 문구를 "끌어다 놓으면 끝"으로 바꾸고 cask의 quarantine 단계를 뺀다.

## 9.4 첫 실행 연결

1. 사용자가 DMG에서 앱을 Applications로 끌어 놓고 연다.
2. 앱은 `~/.stickypane/`을 만들고 세션 목록을 보인다. 비어 있으면 빈 상태 한 장([08 §8.2](08-app-home-and-sessions.md#82-첫-실행--설명-없이-바로-목록)).
3. bridge는 앱에 든 `stickypane`으로 연다(9.1). 그것이 없거나 지워졌을 때만 PATH 등에서 찾고, 그래도 없으면 `brew install --cask LeeSwallow/tap/stickypane`을 보인다.
4. 설정(⌘,)의 **터미널과 에이전트**에서 나머지를 한다([ADR-0019](adr/0019-one-install.md)). 셸의 `SetupService`가 `install/service`를 부른다.

**터미널 명령 설치.** 앱 안의 `stickypane`을 PATH에 링크한다. 기본은 `~/.local/bin/stickypane`(폴더가 없으면 만든다, 암호 없음). `/usr/local/bin/stickypane`은 사용자가 그쪽 버튼을 고를 때만, 시스템 암호 창(macOS `osascript … with administrator privileges`, Linux `pkexec`)으로 `ln -sfn`한다. 그 자리에 앱이 둔 것이 아닌 프로그램이 있으면 묻고 나서 바꾼다. 다른 위치의 앱(옛 사본, 옮긴 앱)을 가리키는 링크나 끊어진 링크는 앱의 것으로 보고 묻지 않고 바꾼다. **제거**는 앱의 링크만 지운다.

앱은 사용자의 로그인 셸(`$SHELL -ilc`)에게 PATH를 물어 터미널이 실제로 무엇을 먼저 쓰는지 본다. GUI 앱의 PATH는 짧아서 그대로는 알 수 없다. 그래서 두 가지를 알려 준다.

- `~/.local/bin`이 PATH에 없으면: 셸의 시작 파일(`~/.zshrc`, `~/.bashrc`, fish는 `config.fish`, 그 밖은 `~/.profile`)과 넣을 한 줄(`export PATH="$HOME/.local/bin:$PATH"`).
- PATH에서 앱의 것보다 먼저 오는 다른 `stickypane`이 있으면(예: Homebrew의 0.1.1): 그 경로와 `stickypane version`이 말한 버전, 그리고 고치는 법. Homebrew cask면 `brew uninstall --cask stickypane`, formula면 `brew uninstall stickypane`, 그 밖은 `rm <경로>`. 또는 PATH에서 링크 폴더를 그 폴더보다 앞에 둔다.

**에이전트 연결.** 앱 안의 `stickypane setup`을 사용자 범위(`--scope user`)로 돌린다. 먼저 `setup --dry-run`의 계획을 보이고, **실행**을 누를 때만 `setup --yes`를 돌려 그 출력을 보인다. setup은 앱이 만든 빈 임시 폴더에서 돌고 끝나면 지운다. 루트나 홈 폴더(심볼릭 링크를 따라간 경로까지)에서는 돌지 않는다. PATH는 로그인 셸의 것에 `~/.local/bin`, `/opt/homebrew/bin`, `/usr/local/bin`을 더한다. 그래야 setup이 `claude`·`codex`를 찾는다. **MCP 서버로도 등록**(기본 켬)은 `--mcp --command <전체 경로>`로 등록한다. 앱의 터미널 명령 링크가 있으면 그 링크를(앱을 업데이트해도 맞다), 없으면 앱 안의 `stickypane`을 쓴다. 그러면 에이전트가 PATH의 옛 사본을 띄우지 않는다. 이미 `stickypane`이라는 MCP 서버가 다른 명령(PATH의 맨 `stickypane` 등)으로 등록돼 있으면(`claude mcp get stickypane`, Codex의 `config.toml`) setup의 계획에 그 등록을 지우고(`mcp remove`, Claude Code는 원래 범위로) 전체 경로로 다시 더하는 두 줄이 들어가, 실행 전에 보인다. 연결한 뒤 프로젝트 창에서는 **이 프로젝트에도 stickypane init**을 권한다. 다시 돌려도 안전하다.

**보드 만들기.** 홈에서 고른 폴더에 `.sticky/`가 없으면 묻고(**보드 만들기**) 앱 안의 `stickypane init`을 그 폴더에서 돌린다. 앱이 띄운 프로젝트 창의 폴더에 보드가 없으면(`no_board`) 창 안에 같은 버튼을 두고, 만든 뒤 bridge를 다시 띄운다. 홈 폴더와 루트는 셸(`refuseInit`)과 본체(`stickypane init`) 둘 다 거절한다.

## 9.5 트레이 아이콘

[사용자 결정] 트레이 아이콘도 앱 아이콘의 세 장 메모지 모양이다(R-43).

| 플랫폼 | 파일 | 규칙 |
|---|---|---|
| macOS | `overlay/build/darwin/tray-template.svg` → `tray-template.png`(18×18)·`tray-template@2x.png`(36×36) | 템플릿 이미지: 검정 + 알파만. 앱 아이콘처럼 크레용으로 그린다: 살짝 휜 굵은 외곽선, 옅은 바탕 위 −41° 빗금. 위 메모지가 아래 메모지에 틈을 낸다. Wails `SystemTray.SetTemplateIcon`으로 넣어 메뉴 막대 색을 따른다. Wails가 막대 두께에 맞춰 크기를 정하므로 앱은 @2x 한 장을 넣는다 |
| Linux | `overlay/build/tray.png`(44×44, 컬러) | `SetIcon` |

`overlay/build/darwin/tray.sh`가 두 가지를 다시 만든다. 어느 쪽을 쓸지는 `shell.trayIcon`이 정하고 테스트가 지킨다(darwin이면 템플릿 + 템플릿 플래그).

## 9.6 검증

| 항목 | 방법 |
|---|---|
| DMG 레이아웃 | 만든 DMG를 `open`으로 마운트해(사용자와 같이) Finder 창만 `screencapture -l <창 id>`로 찍는다. 이 문서의 그림이 그것이다 |
| DMG 스크립트 | `overlay/scripts/dmg_test.sh`: 없는 `.app`, 빈 `.app`, `.dmg`가 아닌 출력, 버전이 `config.yml`과 다른 `.app`에 0이 아닌 코드로 끝나고, 이미지를 쓰지 않고, 아무것도 마운트하지 않는다 |
| 터미널 명령·에이전트 연결 | `install/service`의 테스트가 가짜 파일·실행기·관리자로 링크, 옛 사본 경고, 바꾸기 전 확인, 제거, setup의 폴더와 인자를 확인한다. 실제 `$HOME`·PATH 폴더는 건드리지 않는다. 실제 앱에서 한 번은 사람이 해 본다 |
| Linux 압축 | `overlay/scripts/linux_test.sh`: 가짜 앱으로 `overlay-linux.sh`를 돌려 압축 안이 폴더 하나에 `stickypane-app`·`stickypane`(Linux ELF)·`stickypane.desktop`·`stickypane.png`·`install.sh`·`LICENSE`·`THIRD_PARTY_NOTICES.md`·`licenses/`뿐인지 보고, 임시 `HOME`에서 `install.sh`로 깔고 `--uninstall`로 남김없이 지운다. 잘못된 인자에는 아무것도 쓰지 않고 멈춘다 |
| 번들 CLI | 마운트한 볼륨의 `Stickypane.app/Contents/Helpers/stickypane version`이 앱 버전을 말한다. `bridge` 테스트가 가짜 번들 배치에서 번들 사본을 먼저 고르는지 본다 |
| 앱 실행 | Applications에 복사 후 실행, 세션 목록이 뜨는지 |
| 격리된 첫 실행(수동) | 내려받은 것과 같은 상태를 만들어 본다: 마운트한 볼륨에서 `ditto /Volumes/stickypane/Stickypane.app /tmp/qa/Stickypane.app`, `xattr -w -r com.apple.quarantine "0081;$(printf %x $(date +%s));Safari;" /tmp/qa/Stickypane.app`, `open /tmp/qa/Stickypane.app`. Gatekeeper 경고 → 시스템 설정의 **그래도 열기**로 열리는지, 보드를 열 때 bridge가 번들 CLI로 뜨는지(`ps -o command= -p $(pgrep -f 'Helpers/stickypane bridge')`가 `/tmp/qa/Stickypane.app/Contents/Helpers/stickypane`을 가리키는지) 본다. 사람이 한다 |
| Linux | CI(ubuntu)에서 빌드와 Go 테스트, PR이면 tar.gz를 만들어 산출물로 올린다. 실기 실행은 [미검증] |
