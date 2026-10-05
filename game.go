package main

import (
	"math/rand/v2"
	"time"
)

// 경기 한 판의 판정. 서버가 단일 진실원천이다 — 학생 화면은 받은 상태를 그리기만 한다.
// Match 의 메서드는 Hub.run() 고루틴 안에서만 호출된다(락 없음).
//
// "인구 뿌요": 카드 2장짜리 조각이 떨어진다. 떨어지는 동안은 글자만 보이고, 착지하면 진영 색이 드러난다.
// 같은 나라 카드가 PopN 장 이상 상하좌우로 이어지면 터진다. 색이 2개뿐이라 그냥 터지기는 쉬우므로
// 공격은 연쇄(터진 뒤 내려앉은 카드가 또 터짐)로만 한다 — 2연쇄부터 상대 판에 덮인 카드(방해)를 보낸다.
// 덮인 카드는 바로 옆에서 무언가 터지면 뒤집혀 일반 카드가 된다(그래서 연쇄가 이어질 수도 있다).
// 판 맨 위 가운데(생성 칸)가 막히면 진다. 시간이 끝나면 터뜨린 카드가 많은 쪽이 이긴다.
//
// 판 좌표: 가로 BW × 세로 BH, row 0 이 맨 위. 조각은 row -1(보이지 않는 칸)까지 올라갈 수 있다.

const (
	BW       = 6
	BH       = 10
	PopN     = 4
	NumKinds = 10
	SpawnX   = 2

	faceDev   = 1
	faceIng   = 2
	faceCover = 3 // 방해 카드(덮임)

	SettleDur  = 220 * time.Millisecond // 카드가 내려앉는 연출 시간
	PopDur     = 480 * time.Millisecond // 터지기 직전 깜빡임
	GarbageDur = 380 * time.Millisecond

	MaxGarbageDrop = 18 // 한 번에 떨어지는 방해 카드 최대(3줄)

	FallStart     = 800 * time.Millisecond // 한 칸 떨어지는 간격(시간이 갈수록 빨라짐)
	FallMin       = 250 * time.Millisecond
	FallStepEvery = 30 * time.Second
	FallStep      = 110 * time.Millisecond

	ReadyDur     = 3 * time.Second
	BothGoneWait = 10 * time.Minute
)

// 공격은 2연쇄부터. 연쇄 단계마다 공격 = chainBonus[연쇄] + 묶음마다 (터진 수 - PopN)
var chainBonus = []int{0, 0, 4, 8, 13, 19, 26, 34, 43, 53, 64}

type phase string

const (
	phReady phase = "ready"
	phPlay  phase = "play"
	phPause phase = "pause" // 한쪽 연결 끊김 — 재접속 대기
	phEnd   phase = "end"
)

type Rules struct {
	TimeSec int `json:"timeSec"`
	// GoldenGoal: 시간이 끝났는데 동점이면 먼저 터뜨리는 쪽이 이기는 연장(토너먼트 — 무승부 없음).
	GoldenGoal bool `json:"goldenGoal,omitempty"`
}

func (r Rules) valid() bool { return r.TimeSec >= 60 && r.TimeSec <= 300 }

type Cell struct {
	ID   int
	Face int // 0 = 빈칸
	Kind int
}

// 조각: A 가 축, B 는 A 의 Rot 방향(0 위, 1 오른쪽, 2 아래, 3 왼쪽)에 붙는다.
type Piece struct {
	X, Y, Rot int
	A, B      Cell
}

var rotDX = [4]int{0, 1, 0, -1}
var rotDY = [4]int{-1, 0, 1, 0}

func (p *Piece) bPos() (int, int) { return p.X + rotDX[p.Rot], p.Y + rotDY[p.Rot] }

type Stats struct {
	Popped   int `json:"popped"`   // 터뜨린 카드
	MaxChain int `json:"maxChain"` // 최대 연쇄
	Sent     int `json:"sent"`     // 보낸 방해 카드
	Placed   int `json:"placed"`   // 놓은 카드
}

// Event 는 화면 연출용 순간 사건(그 판 주인에게 간다).
type Event struct {
	K string `json:"k"`           // land | pop | atk(공격 보냄) | hit(방해 받음) | rv(덮인 카드 열림)
	N int    `json:"n,omitempty"` // 수
	C int    `json:"c,omitempty"` // 연쇄
}

type Board struct {
	grid     [BH][BW]Cell
	piece    *Piece
	pieceIdx int
	state    string // fall | settle | pop | garbage | dead
	timer    time.Duration
	fallT    time.Duration
	grounded bool
	chain    int
	attack   int   // 이번 연쇄에서 쌓인 공격
	popping  []int // 터지는 중인 카드 id
	pending  int   // 받을 방해 카드
	stats    Stats
	events   []Event
}

type Match struct {
	id      string
	rules   Rules
	players [2]*Player

	phase     phase
	phaseLeft time.Duration // ready: 남은 시간, pause: 기다린 시간

	boards [2]*Board
	seq    [][2]Cell // 두 사람이 같은 순서로 받는 조각(종류·면)
	nextID int

	playLeft  time.Duration
	played    time.Duration
	startedAt time.Time

	winner  int    // -1 무승부
	reason  string // "topout" | "time" | "golden" | "forfeit"
	rematch [2]bool
	saved   bool
	golden  bool     // 연장(먼저 터뜨리면 승리) 중
	fixture *Fixture // 대회 경기면 그 대진

	rng *rand.Rand
}

func newMatch(id string, a, b *Player, rules Rules, now time.Time, seed uint64) *Match {
	m := &Match{
		id: id, rules: rules, players: [2]*Player{a, b},
		startedAt: now, winner: -1,
		playLeft: time.Duration(rules.TimeSec) * time.Second,
		rng:      rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
	}
	for s := range m.boards {
		m.boards[s] = &Board{state: "fall"}
	}
	m.phase = phReady
	m.phaseLeft = ReadyDur
	for s := range m.boards {
		m.spawn(m.boards[s])
	}
	return m
}

func (m *Match) sideOf(p *Player) int {
	if m.players[0] == p {
		return 0
	}
	if m.players[1] == p {
		return 1
	}
	return -1
}

func (m *Match) newID() int { m.nextID++; return m.nextID }

// pieceAt 은 idx 번째 조각의 종류·면(두 사람 공통). 필요할 때 만들어 둔다.
func (m *Match) pieceAt(idx int) [2]Cell {
	for len(m.seq) <= idx {
		var p [2]Cell
		for i := range p {
			p[i] = Cell{Kind: m.rng.IntN(NumKinds), Face: 1 + m.rng.IntN(2)}
		}
		m.seq = append(m.seq, p)
	}
	return m.seq[idx]
}

// ── 조각 ────────────────────────────────────────────────

func (b *Board) free(x, y int) bool {
	if x < 0 || x >= BW || y >= BH {
		return false
	}
	return y < 0 || b.grid[y][x].Face == 0
}

func (b *Board) fits(x, y, rot int) bool {
	return b.free(x, y) && b.free(x+rotDX[rot], y+rotDY[rot])
}

func (m *Match) spawn(b *Board) {
	if b.grid[0][SpawnX].Face != 0 {
		b.state = "dead"
		b.piece = nil
		return
	}
	pc := m.pieceAt(b.pieceIdx)
	b.pieceIdx++
	pa, pb := pc[0], pc[1]
	pa.ID, pb.ID = m.newID(), m.newID()
	b.piece = &Piece{X: SpawnX, Y: 0, Rot: 0, A: pa, B: pb}
	b.state = "fall"
	b.fallT = 0
	b.grounded = false
}

func (m *Match) canControl(side int) *Board {
	if m.phase != phPlay {
		return nil
	}
	b := m.boards[side]
	if b.state != "fall" || b.piece == nil {
		return nil
	}
	return b
}

func (m *Match) move(side, d int) bool {
	b := m.canControl(side)
	if b == nil || (d != 1 && d != -1) {
		return false
	}
	p := b.piece
	if !b.fits(p.X+d, p.Y, p.Rot) {
		return false
	}
	p.X += d
	return true
}

func (m *Match) rotate(side, d int) bool {
	b := m.canControl(side)
	if b == nil || (d != 1 && d != -1) {
		return false
	}
	p := b.piece
	nr := (p.Rot + d + 4) % 4
	switch {
	case b.fits(p.X, p.Y, nr):
	case b.fits(p.X-rotDX[nr], p.Y-rotDY[nr], nr): // 벽·카드에 막히면 반대쪽으로 밀어 준다
		p.X -= rotDX[nr]
		p.Y -= rotDY[nr]
	default:
		nr2 := (p.Rot + 2) % 4 // 좁은 틈: 위아래 뒤집기
		if (nr == 1 || nr == 3) && b.fits(p.X, p.Y, nr2) {
			nr = nr2
		} else if (nr == 1 || nr == 3) && b.fits(p.X-rotDX[nr2], p.Y-rotDY[nr2], nr2) {
			p.X -= rotDX[nr2]
			p.Y -= rotDY[nr2]
			nr = nr2
		} else {
			return false
		}
	}
	p.Rot = nr
	return true
}

func (m *Match) drop(side int) bool {
	b := m.canControl(side)
	if b == nil {
		return false
	}
	p := b.piece
	for b.fits(p.X, p.Y+1, p.Rot) {
		p.Y++
	}
	m.lock(b)
	return true
}

func (m *Match) lock(b *Board) {
	p := b.piece
	b.piece = nil
	bx, by := p.bPos()
	for _, c := range [2]struct {
		x, y int
		cell Cell
	}{{p.X, p.Y, p.A}, {bx, by, p.B}} {
		if c.y >= 0 {
			b.grid[c.y][c.x] = c.cell
		}
	}
	b.stats.Placed += 2
	b.gravity()
	b.chain, b.attack = 0, 0
	b.state, b.timer = "settle", SettleDur
	b.events = append(b.events, Event{K: "land"})
}

// gravity 는 빈칸 위 카드를 바닥으로 내린다.
func (b *Board) gravity() {
	for x := 0; x < BW; x++ {
		w := BH - 1
		for y := BH - 1; y >= 0; y-- {
			if b.grid[y][x].Face != 0 {
				if y != w {
					b.grid[w][x] = b.grid[y][x]
					b.grid[y][x] = Cell{}
				}
				w--
			}
		}
	}
}

// groups 는 PopN 장 이상 이어진 같은 나라 카드 묶음.
func (b *Board) groups() [][][2]int {
	var seen [BH][BW]bool
	var out [][][2]int
	for y := 0; y < BH; y++ {
		for x := 0; x < BW; x++ {
			f := b.grid[y][x].Face
			if seen[y][x] || (f != faceDev && f != faceIng) {
				continue
			}
			stack := [][2]int{{x, y}}
			seen[y][x] = true
			var g [][2]int
			for len(stack) > 0 {
				c := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				g = append(g, c)
				for d := 0; d < 4; d++ {
					nx, ny := c[0]+rotDX[d], c[1]+rotDY[d]
					if nx < 0 || nx >= BW || ny < 0 || ny >= BH || seen[ny][nx] || b.grid[ny][nx].Face != f {
						continue
					}
					seen[ny][nx] = true
					stack = append(stack, [2]int{nx, ny})
				}
			}
			if len(g) >= PopN {
				out = append(out, g)
			}
		}
	}
	return out
}

// ── 진행 ────────────────────────────────────────────────

func (m *Match) fallInterval() time.Duration {
	return max(FallMin, FallStart-FallStep*time.Duration(m.played/FallStepEvery))
}

func (m *Match) stepBoard(side int, dt time.Duration) {
	b := m.boards[side]
	switch b.state {
	case "fall":
		b.fallT += dt
		iv := m.fallInterval()
		for b.fallT >= iv && b.piece != nil {
			b.fallT -= iv
			p := b.piece
			if b.fits(p.X, p.Y+1, p.Rot) {
				p.Y++
				b.grounded = false
			} else if b.grounded {
				m.lock(b) // 바닥에 닿은 채 한 칸 시간이 지나면 고정(그 사이 옆으로 밀 수 있다)
			} else {
				b.grounded = true
			}
		}
	case "settle":
		if b.timer -= dt; b.timer > 0 {
			return
		}
		gs := b.groups()
		if len(gs) == 0 {
			m.endChain(side)
			return
		}
		b.chain++
		n := 0
		b.popping = b.popping[:0]
		extra := 0
		for _, g := range gs {
			n += len(g)
			extra += len(g) - PopN
			for _, c := range g {
				b.popping = append(b.popping, b.grid[c[1]][c[0]].ID)
			}
		}
		if b.chain >= 2 { // 그냥 한 번 터지는 건 공격이 아니다
			b.attack += chainBonus[min(b.chain, len(chainBonus)-1)] + extra
		}
		b.stats.Popped += n
		b.stats.MaxChain = max(b.stats.MaxChain, b.chain)
		b.events = append(b.events, Event{K: "pop", N: n, C: b.chain})
		b.state, b.timer = "pop", PopDur
	case "pop":
		if b.timer -= dt; b.timer > 0 {
			return
		}
		m.clearPopping(b)
		b.gravity()
		b.state, b.timer = "settle", SettleDur
	case "garbage":
		if b.timer -= dt; b.timer > 0 {
			return
		}
		m.spawn(b)
	}
}

// clearPopping 은 터진 카드를 지우고, 바로 옆 덮인 카드를 무작위 면으로 연다.
func (m *Match) clearPopping(b *Board) {
	pop := map[int]bool{}
	for _, id := range b.popping {
		pop[id] = true
	}
	var popped [][2]int
	for y := 0; y < BH; y++ {
		for x := 0; x < BW; x++ {
			if pop[b.grid[y][x].ID] && b.grid[y][x].Face != 0 {
				popped = append(popped, [2]int{x, y})
			}
		}
	}
	opened := 0
	for _, c := range popped {
		for d := 0; d < 4; d++ {
			nx, ny := c[0]+rotDX[d], c[1]+rotDY[d]
			if nx >= 0 && nx < BW && ny >= 0 && ny < BH && b.grid[ny][nx].Face == faceCover {
				b.grid[ny][nx].Face = 1 + m.rng.IntN(2)
				opened++
			}
		}
	}
	for _, c := range popped {
		b.grid[c[1]][c[0]] = Cell{}
	}
	b.popping = b.popping[:0]
	if opened > 0 {
		b.events = append(b.events, Event{K: "rv", N: opened})
	}
}

// endChain: 연쇄가 끝나면 공격을 보낸다(내가 받을 방해부터 상쇄). 그다음 받을 방해가 있으면 떨어뜨린다.
func (m *Match) endChain(side int) {
	b, o := m.boards[side], m.boards[1-side]
	if a := b.attack; a > 0 {
		off := min(a, b.pending)
		b.pending -= off
		a -= off
		if a > 0 {
			o.pending += a
			b.stats.Sent += a
			b.events = append(b.events, Event{K: "atk", N: a})
			o.events = append(o.events, Event{K: "hit", N: a})
		}
	}
	b.attack, b.chain = 0, 0
	if b.pending > 0 {
		m.dropGarbage(b)
		b.state, b.timer = "garbage", GarbageDur
		return
	}
	m.spawn(b)
}

func (m *Match) dropGarbage(b *Board) {
	n := min(b.pending, MaxGarbageDrop)
	b.pending -= n
	var per [BW]int
	for x := range per {
		per[x] = n / BW
	}
	cols := m.rng.Perm(BW)
	for i := 0; i < n%BW; i++ {
		per[cols[i]]++
	}
	for x := 0; x < BW; x++ {
		top := BH - 1
		for top >= 0 && b.grid[top][x].Face != 0 {
			top--
		}
		for k := 0; k < per[x] && top >= 0; k++ {
			b.grid[top][x] = Cell{ID: m.newID(), Face: faceCover, Kind: m.rng.IntN(NumKinds)}
			top--
		}
	}
}

func (m *Match) step(dt time.Duration) {
	switch m.phase {
	case phEnd:
		return
	case phPause:
		m.phaseLeft += dt
		return
	case phReady:
		m.phaseLeft -= dt
		if m.phaseLeft <= 0 {
			m.phase = phPlay
		}
		return
	}
	m.played += dt
	m.playLeft = max(0, m.playLeft-dt)
	before := m.score()
	for s := range m.boards {
		m.stepBoard(s, dt)
	}
	d0, d1 := m.boards[0].state == "dead", m.boards[1].state == "dead"
	switch {
	case m.golden && !d0 && !d1:
		after := m.score()
		g0, g1 := after[0]-before[0], after[1]-before[1]
		if g0 > 0 || g1 > 0 { // 연장: 먼저 터뜨린 쪽 승리(같은 순간이면 더 많이 터뜨린 쪽)
			m.finish("golden")
			m.winner = 0
			if g1 > g0 {
				m.winner = 1
			}
		}
	case d0 || d1:
		m.finish("topout")
		switch {
		case d0 && d1:
			m.winner = -1
		case d0:
			m.winner = 1
		default:
			m.winner = 0
		}
	case m.playLeft <= 0:
		p0, p1 := m.boards[0].stats.Popped, m.boards[1].stats.Popped
		if p0 == p1 && m.rules.GoldenGoal {
			m.golden = true // 토너먼트 동점 → 연장
			return
		}
		m.finish("time")
		switch {
		case p0 > p1:
			m.winner = 0
		case p1 > p0:
			m.winner = 1
		}
	}
}

func (m *Match) finish(reason string) {
	m.phase = phEnd
	m.reason = reason
	m.winner = -1
	for _, b := range m.boards {
		b.piece = nil
	}
}

// score 는 터뜨린 카드 수(점수판 표시용).
func (m *Match) score() [2]int {
	return [2]int{m.boards[0].stats.Popped, m.boards[1].stats.Popped}
}

func (m *Match) forfeit(side int) {
	if m.phase == phEnd {
		return
	}
	m.finish("forfeit")
	m.winner = 1 - side
}

func (m *Match) pauseForReconnect() {
	if m.phase == phEnd || m.phase == phPause {
		return
	}
	m.phase = phPause
	m.phaseLeft = 0
}

func (m *Match) unpause() {
	if m.phase != phPause {
		return
	}
	m.phase = phReady
	m.phaseLeft = ReadyDur
}
