package main

import (
	"testing"
	"time"
)

const tick = time.Second / 60

func testMatch(t *testing.T) *Match {
	t.Helper()
	a := &Player{id: "a", name: "가"}
	b := &Player{id: "b", name: "나"}
	m := newMatch("m", a, b, Rules{TimeSec: 120}, time.Now(), 42)
	for m.phase == phReady {
		m.step(tick)
	}
	if m.phase != phPlay {
		t.Fatalf("phase = %s, want play", m.phase)
	}
	return m
}

// run 은 판 side 가 다음 조각을 받을 때까지(또는 d 만큼) 진행한다.
func runUntilFall(t *testing.T, m *Match, side int) {
	t.Helper()
	for k := 0; k < 60*10; k++ {
		if b := m.boards[side]; b.state == "fall" && b.piece != nil {
			return
		}
		m.step(tick)
	}
	t.Fatalf("board %d never returned to fall (state %s)", side, m.boards[side].state)
}

func put(b *Board, x, y, face int) { b.grid[y][x] = Cell{ID: 1000 + y*BW + x, Face: face} }

func TestSameSequenceForBoth(t *testing.T) {
	m := testMatch(t)
	a, b := m.boards[0].piece, m.boards[1].piece
	if a.A.Kind != b.A.Kind || a.A.Face != b.A.Face || a.B.Kind != b.B.Kind || a.B.Face != b.B.Face {
		t.Fatal("both players must get the same first piece")
	}
}

func TestMoveRotateDrop(t *testing.T) {
	m := testMatch(t)
	b := m.boards[0]
	for m.move(0, -1) {
	}
	if b.piece.X != 0 {
		t.Fatalf("x = %d, want 0", b.piece.X)
	}
	if m.rotate(0, -1) && b.piece.X < 0 {
		t.Fatal("rotate must not go through the wall")
	}
	m.rotate(0, 1) // 오른쪽으로 눕힘
	if bx, _ := b.piece.bPos(); bx < 0 || bx >= BW {
		t.Fatal("rotated cell out of board")
	}
	if !m.drop(0) || b.piece != nil || b.state != "settle" || b.stats.Placed != 2 {
		t.Fatalf("drop: state=%s", b.state)
	}
	n := 0
	for y := 0; y < BH; y++ {
		for x := 0; x < BW; x++ {
			if b.grid[y][x].Face != 0 {
				n++
				if y < BH-2 {
					t.Fatalf("card floating at row %d", y)
				}
			}
		}
	}
	if n != 2 {
		t.Fatalf("cards on board = %d", n)
	}
}

func TestGravityLocksAfterGrace(t *testing.T) {
	m := testMatch(t)
	b := m.boards[0]
	for k := 0; k < 60*20 && b.stats.Placed == 0; k++ {
		m.step(tick)
	}
	if b.stats.Placed != 2 {
		t.Fatal("piece should lock by gravity")
	}
}

func settleAll(t *testing.T, m *Match, side int) {
	t.Helper()
	b := m.boards[side]
	b.piece = nil
	b.state, b.timer = "settle", 0
	runUntilFall(t, m, side)
}

func TestSinglePopDoesNotAttack(t *testing.T) {
	m := testMatch(t)
	b, o := m.boards[0], m.boards[1]
	for x := 0; x < 5; x++ {
		put(b, x, BH-1, faceIng) // 5장 한 번에 터짐 = 1연쇄
	}
	settleAll(t, m, 0)
	if b.stats.Popped != 5 || b.stats.MaxChain != 1 {
		t.Fatalf("stats = %+v", b.stats)
	}
	if b.stats.Sent != 0 || o.pending != 0 {
		t.Fatalf("single pop must not attack: sent=%d pending=%d", b.stats.Sent, o.pending)
	}
}

func TestChainAttacks(t *testing.T) {
	m := testMatch(t)
	b, o := m.boards[0], m.boards[1]
	// 열3 의 개발도상국 4장이 터지면 그 위 선진국이 내려와 바닥의 선진국 3장과 이어져 2연쇄
	for x := 0; x < 3; x++ {
		put(b, x, BH-1, faceDev)
	}
	for k := 1; k <= 4; k++ {
		put(b, 3, BH-k, faceIng)
	}
	put(b, 3, BH-5, faceDev)
	settleAll(t, m, 0)
	if b.stats.MaxChain != 2 || b.stats.Popped != 8 {
		t.Fatalf("stats = %+v", b.stats)
	}
	if b.stats.Sent != chainBonus[2] || o.pending != chainBonus[2] {
		t.Fatalf("2-chain attack: sent=%d pending=%d", b.stats.Sent, o.pending)
	}
}

func TestCoverOpensNextToPop(t *testing.T) {
	m := testMatch(t)
	b := m.boards[0]
	b.piece = nil
	for x := 0; x < 4; x++ {
		put(b, x, BH-1, faceDev)
	}
	put(b, 4, BH-1, faceCover)
	b.state, b.timer = "settle", 0
	m.step(tick)
	for b.state == "pop" {
		m.step(tick)
	}
	if f := b.grid[BH-1][4].Face; f != faceDev && f != faceIng && b.grid[BH-1][0].Face == 0 {
		t.Fatalf("cover next to pop should open, face=%d", f)
	}
}

func TestGarbageDropsAndTopOutLoses(t *testing.T) {
	m := testMatch(t)
	b := m.boards[1]
	b.pending = 8
	m.drop(1)
	runUntilFall(t, m, 1)
	covers := 0
	for y := 0; y < BH; y++ {
		for x := 0; x < BW; x++ {
			if b.grid[y][x].Face == faceCover {
				covers++
			}
		}
	}
	if covers != 8 || b.pending != 0 {
		t.Fatalf("covers=%d pending=%d", covers, b.pending)
	}
	// 생성 칸을 막으면 다음 조각에서 진다
	b.piece = nil
	put(b, SpawnX, 0, faceDev)
	m.spawn(b)
	m.step(tick)
	if m.phase != phEnd || m.winner != 0 || m.reason != "topout" {
		t.Fatalf("topout: phase=%s winner=%d reason=%s", m.phase, m.winner, m.reason)
	}
}

func TestTimeUpMorePoppedWins(t *testing.T) {
	m := testMatch(t)
	m.boards[1].stats.Popped = 9
	m.playLeft = tick
	m.step(tick)
	if m.phase != phEnd || m.winner != 1 || m.reason != "time" {
		t.Fatalf("time: winner=%d reason=%s", m.winner, m.reason)
	}
}

func TestPauseAndForfeit(t *testing.T) {
	m := testMatch(t)
	m.pauseForReconnect()
	if m.move(0, 1) || m.drop(0) {
		t.Fatal("no control while paused")
	}
	m.unpause()
	if m.phase != phReady {
		t.Fatal("unpause → ready")
	}
	m.forfeit(1)
	if m.phase != phEnd || m.winner != 0 || m.reason != "forfeit" {
		t.Fatal("forfeit")
	}
}
