package main

import (
	"encoding/json"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)

// 허브를 고루틴 없이 직접 구동한다(onRegister/onMessage/onTick 을 순서대로 호출).

type fakeHub struct {
	*Hub
	clock time.Time
	saved []*Match
}

func newFakeHub() *fakeHub {
	f := &fakeHub{Hub: newHub(), clock: t0}
	f.now = func() time.Time { return f.clock }
	f.save = func(m *Match) { f.saved = append(f.saved, m) }
	f.saveComp = func(*Competition) {}
	return f
}

func (f *fakeHub) join(token, name string) *client {
	c := &client{hub: f.Hub, send: make(chan []byte, 4096), token: token, name: name}
	f.onRegister(c)
	return c
}

func (f *fakeHub) say(c *client, v any) {
	b, _ := json.Marshal(v)
	f.onMessage(inMsg{c: c, data: b})
}

func (f *fakeHub) advance(d time.Duration) {
	for el := time.Duration(0); el < d; el += tickPeriod {
		f.clock = f.clock.Add(tickPeriod)
		f.onTick()
	}
}

// last 는 c 가 받은 메시지 중 타입 t 의 마지막 것을 돌려준다(버퍼는 비운다).
func last(c *client, t string) map[string]any {
	var got map[string]any
	for {
		select {
		case b, ok := <-c.send:
			if !ok {
				return got
			}
			var m map[string]any
			if json.Unmarshal(b, &m) == nil && m["t"] == t {
				got = m
			}
		default:
			return got
		}
	}
}

func TestRoomCodeMatch(t *testing.T) {
	f := newFakeHub()
	a, b := f.join("ta", "민수"), f.join("tb", "지우")
	f.say(a, map[string]any{"t": "create", "rules": Rules{TimeSec: 120}})
	f.advance(200 * time.Millisecond)
	rooms := last(b, "lobby")["rooms"].([]any)
	if len(rooms) != 1 {
		t.Fatalf("열린 방 1개가 보여야 함: %v", rooms)
	}
	code := rooms[0].(map[string]any)["code"].(string)
	f.say(b, map[string]any{"t": "join", "code": code})
	if ma, mb := last(a, "match"), last(b, "match"); ma == nil || mb == nil || ma["side"].(float64) != 0 || mb["side"].(float64) != 1 {
		t.Fatalf("두 학생 모두 경기 시작 메시지를 받아야 함: %v / %v", ma, mb)
	}
}

func TestHelloTellsWhetherInMatch(t *testing.T) {
	f := newFakeHub()
	a := f.join("ta", "민수")
	if h := last(a, "hello"); h == nil || h["inMatch"] != false {
		t.Fatalf("경기 없는 학생은 inMatch=false: %v", h)
	}
}
