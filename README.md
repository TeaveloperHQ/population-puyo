# population-puyo (인구 뿌요)

사회 **'인구 문제'** 단원의 선진국·개발도상국 인구 특징 카드를 뿌요뿌요식 1:1 낙하 퍼즐 대전으로 옮긴 교실용 서버.
같은 교내망에 붙은 학생들이 **QR로 접속**해 방을 만들거나 친구를 초대해 겨룬다.
교사 PC에서 **exe 한 개를 더블클릭**하면 끝. 학생은 **앱 설치 없이 브라우저**로 들어온다.

📖 **선생님용 사용법은 [사용 설명서(MANUAL.md)](MANUAL.md)** 를 보세요. 이 README 는 만드는 사람을 위한 설명입니다.

구조는 형제 프로젝트 [vector-soccer](https://github.com/TeaveloperHQ/vector-soccer)·[classroom-quiz](https://github.com/TeaveloperHQ/classroom-quiz)와 같다
(LAN 바인딩 · QR · 액터 모델 허브 · 재접속 · 대회 · 결과 파일 저장).

## 게임 방식

- 판은 가로 6 × 세로 10. 블록 2개짜리 조각이 떨어지고, 블록마다 인구 특징 글자가 있다(카드 10종 × 선진국/개발도상국 면).
  색: 선진국 파랑, 개발도상국 주황.
- 같은 나라 블록 4개 이상이 상하좌우로 이어지면 터진다. 색이 2개뿐이라 그냥 터지기는 쉬우므로
  **공격은 연쇄(2연쇄부터)로만** 한다 — 연쇄 단계마다 `chainBonus[연쇄] + (묶음 크기 - 4)` 장의 방해 블록을 보낸다.
  보낼 공격은 먼저 내가 받을 방해와 상쇄된다.
- 방해 블록(덮인 카드)은 바로 옆에서 무언가 터지면 **무작위 면으로 뒤집혀** 일반 블록이 된다(연쇄가 이어질 수 있다).
- 생성 칸(가운데 맨 위)이 막히면 진다. 시간(2·3분)이 끝나면 터뜨린 블록이 많은 쪽 승리, 같으면 무승부
  (토너먼트는 연장 — 먼저 터뜨리는 쪽 승리).
- 두 학생은 **같은 조각 순서**를 받는다. 30초마다 낙하가 빨라진다(0.8초 → 최소 0.25초).
- 끝나면 정답 카드 표로 복습. 한 판 더는 판 위치(왼쪽/오른쪽)를 바꿔 새 경기.
- 조작: ◀ ▶(양 끝) · ⟳ 돌리기 · ⬇ 내리기(◀▶ 아래), 판 위 스와이프·탭, 키보드. 돌리기·내리기 좌우는 학생이 바꿀 수 있다(왼손잡이).

밸런스 메모(봇 시뮬레이션, 사람 속도 1.2초/조각): 색을 보고 놓는 봇이 무작위 봇을
"아무 터짐이나 공격"일 때 72%, "연쇄만 공격"일 때 91% 이긴다 — 연쇄만 공격하는 쪽이 실력이 더 드러난다.

## 대회 · 교사 화면

vector-soccer 와 같다. 교사 화면(`/host`, 교사 PC에서만 열림):

- **경기장** 탭: 접속 QR(크게 띄우기) · 접속/경기 중/대기 방 수 · 진행 중인 경기의 두 판(실시간) · 학생 상태.
- **대회** 탭: 리그전(조 편성)·토너먼트 만들기 · 라운드 진행 · 순위표/대진표 · ⋯ 메뉴(이 경기만 시작·승자 지정·취소·되돌리기) · 크게 보기.
  진행 상황은 `competitions/<id>.json` 에 저장 — 서버를 다시 켜도 이어서 한다.
- **결과** 탭: 날짜별 필터 · 학생별 요약 · 경기 기록(삭제 가능) · **엑셀(CSV) 내려받기**.

결과는 exe 옆 `population-puyo-results/<날짜>/<시각>-<경기id>.json` 에 한 판씩 저장된다.
폴더 이름에 게임 이름을 붙인 건 포털에서 받은 여러 게임 exe 가 같은 폴더(다운로드)에 있어도 섞이지 않게.

## 빌드 / 실행

```bash
./build.sh                                  # dist/population-puyo.exe (콘솔 창 = 실행 중 표시, 닫으면 종료)
./build.sh dist/population-puyo.exe gui     # 콘솔 없이(-H windowsgui)
go test ./...                               # 낙하·연쇄·공격·대회·허브 테스트
CF_NO_BROWSER=1 go run .                    # 개발 실행(브라우저 자동 열기 끔)
```

리눅스에서 윈도우 exe로 그대로 크로스컴파일된다(`CGO_ENABLED=0`). 기본 포트는 **8100**
(classroom-quiz 8080 · vector-soccer 8090 과 동시에 켜도 겹치지 않게), 사용 중이면 다음 포트로 넘어간다.

## 설계 메모

- **서버가 판정한다(단일 진실원천).** 60Hz로 낙하·연쇄를 계산하고 20Hz로 상태를 보낸다. 조작(이동·돌리기·내리기)은
  처리 즉시 결과를 보낸다. 학생 화면은 받은 판을 그리고, 카드 id 로 내려앉기·터짐을 연출한다.
- 모든 상태 변경은 `Hub.run()` 단일 고루틴 안에서만 일어난다(락 없음).
- 학생 식별: `sessionStorage` 토큰 — 새로고침해도 같은 학생으로 복귀(경기 중이면 경기로).
  서버를 다시 켜서 경기가 사라졌으면 `hello.inMatch=false` 로 로비로 돌려보낸다.
- `/host`, `/info`, 결과 API, 교사용 WebSocket 은 `localOnly`(loopback)로 막는다.
- 글꼴: 주아체(Jua, SIL OFL 1.1)를 한글·영문만 남겨 woff2(364KB)로 줄여 내장했다 — 인터넷 없는 교실에서도 같은 모습.
  라이선스는 [assets/fonts/OFL-Jua.txt](assets/fonts/OFL-Jua.txt).
- 카드 내용(10쌍)은 [assets/cards.js](assets/cards.js). 원본 디자인 에셋은 [population-cards/](population-cards/).

### WebSocket 프로토콜 (`/ws?name=..&sid=..&token=..`)

| 방향 | 메시지 |
|---|---|
| 학생 → 서버 | `create{rules}` `cancel` `join{code}` `invite{to,rules}` `inviteReply{from,accept}` `mv{d}` `rot{d}` `drop` `rematch` `leave` |
| 서버 → 학생 | `hello{inMatch}` `lobby{players,rooms,invites,sent,myRoom,comp}` `match{side,names,rules,board,comp}` `s{ph,pl,tl,b[2],sc,on,ev,gg,...}` `notice` `error` `cancelled` |
| 서버 → 교사 | `host{players,matches,rooms}` (10Hz) · `comp{comp,standings,current,champion,online,busy,roundNames,groupNames}` (바뀔 때) |
| 교사 → 서버 | `compCreate{name,type,rules,keys,groups}` `compStartRound` `compStartFixture{id}` `compDecide{id,winner}` `compReset{id}` `compClose` |

`s.b[i]` = 판 하나: `c`(놓인 블록 `[칸, id, 면, 종류]`) · `p`(떨어지는 조각 `[x, y, 회전, idA, 종류A, 면A, idB, 종류B, 면B]`) ·
`st`(fall/settle/pop/garbage/dead) · `pd`(받을 방해) · `ch`(연쇄) · `pp`(터지는 중 id) · `sc`(터뜨린 수), 내 판만 `n`(다음 조각)·`fi`(낙하 간격).
면: 1 선진국 · 2 개발도상국 · 3 방해(덮임).

## 파일

| 파일 | 역할 |
|---|---|
| main.go | LAN IP 탐지 · `0.0.0.0` 바인딩 · 라우팅 · QR · 브라우저 자동 열기 · 로그 |
| hub.go | 연결 · 로비(방/초대) · 경기 진행 틱 · 재접속 · 송신 메시지 |
| game.go | 판 · 조각 이동/회전(벽 차기) · 중력 · 연쇄 · 공격/상쇄 · 방해 블록 · 연장 · 일시정지/기권 |
| comp.go | 대회: 리그 라운드 · 토너먼트 대진/부전승/진출 · 순위 · 저장 (vector-soccer 와 같음) |
| hub_comp.go | 교사 대회 명령 · 대진을 경기로 열기 · 결과 반영 · 학생 안내 |
| results.go | 결과 저장 · 목록 · CSV |
| api.go | 교사 전용 API · `localOnly` |
| netmedia*.go | 교사 PC 접속 주소가 무선/유선인지 판별 |
| assets/student.html | 입장 · 로비 · 경기(캔버스, 젤리 블록) |
| assets/host.html | 교사 화면(경기장/대회/결과) |
| assets/cards.js · fonts/ | 카드 데이터(c05 는 이모지 카드) · 주아체 |
| branding/ | 아이콘 원본(SVG)·생성기·PNG·ico |
| rsrc_windows_amd64.syso | exe 아이콘 리소스(윈도우 빌드 시 자동 링크) |

## 아이콘

`branding/` — teaveloper 공용 규격(1024 캔버스 · 800 라운드 사각형 rx 184 · 공식 그라데이션 · 흰 죽방).
죽방 옆에 눈 달린 젤리 블록(선진국 파랑·개발도상국 주황)이 쌓이고 하나가 떨어진다.
작은 크기(16·32·48px)는 젤리 블록 3개만 남긴 `icon-small.svg`.

```bash
cd branding && python3 gen.py        # icon.svg / icon-small.svg 다시 만들기
R="-density 384 -background none"
magick $R icon.svg -resize 1024x1024 app-icon-1024.png   # 512·256 도 같은 방식
magick \( $R icon.svg -resize 256x256 \) \( $R icon.svg -resize 128x128 \) \( $R icon.svg -resize 64x64 \) \
       \( $R icon-small.svg -resize 48x48 \) \( $R icon-small.svg -resize 32x32 \) \( $R icon-small.svg -resize 16x16 \) app.ico
cd .. && rsrc -ico branding/app.ico -arch amd64 -o rsrc_windows_amd64.syso   # exe 아이콘(윈도우 빌드 시 자동 링크)
cp branding/icon-small.svg assets/favicon.svg && cp branding/icon.svg assets/icon.svg   # 파비콘 · 입장 화면 로고
```

## 라이선스

MIT — [LICENSE](LICENSE) 참고. 주아체는 SIL Open Font License 1.1.
