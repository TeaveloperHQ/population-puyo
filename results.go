package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 경기 결과는 exe 옆 population-puyo-results/<날짜>/<시각>-<경기id>.json 에 한 판씩 저장한다.
// 폴더 이름에 게임 이름을 붙인 건 포털에서 받은 여러 게임 exe 가 같은 폴더(다운로드)에 있어도 섞이지 않게.
// 날짜 폴더라 "언제 어느 반이 했나"가 폴더 구조로 바로 보이고, 파일이라 백업·이관이 쉽다.

type ResultPlayer struct {
	SID      string `json:"sid"`
	Name     string `json:"name"`
	Popped   int    `json:"popped"`   // 터뜨린 카드
	MaxChain int    `json:"maxChain"` // 최대 연쇄
	Sent     int    `json:"sent"`     // 보낸 방해 카드
	Placed   int    `json:"placed"`   // 놓은 카드
	Result   string `json:"result"`   // "승" | "무" | "패"
}

type MatchResult struct {
	ID         string          `json:"id"`
	Date       string          `json:"date"` // 폴더명(YYYY-MM-DD)
	File       string          `json:"file"`
	StartedAt  string          `json:"startedAt"`  // RFC3339
	FinishedAt string          `json:"finishedAt"` // RFC3339
	PlayedSec  int             `json:"playedSec"`
	Rules      Rules           `json:"rules"`
	Reason     string          `json:"reason"`         // "topout" | "time" | "golden" | "forfeit"
	Comp       string          `json:"comp,omitempty"` // 대회 경기면 "대회 이름 · 라운드"
	Players    [2]ResultPlayer `json:"players"`
}

func resultsDir() string { return filepath.Join(exeDir(), "population-puyo-results") }

func buildResult(m *Match, finished time.Time) *MatchResult {
	r := &MatchResult{
		ID: m.id, StartedAt: m.startedAt.Format(time.RFC3339), FinishedAt: finished.Format(time.RFC3339),
		PlayedSec: int(m.played.Seconds()), Rules: m.rules, Reason: m.reason,
	}
	for i, p := range m.players {
		res := "무"
		if m.winner == i {
			res = "승"
		} else if m.winner == 1-i {
			res = "패"
		}
		st := m.boards[i].stats
		r.Players[i] = ResultPlayer{SID: p.sid, Name: p.name, Popped: st.Popped, MaxChain: st.MaxChain,
			Sent: st.Sent, Placed: st.Placed, Result: res}
	}
	return r
}

func (h *Hub) saveMatchResult(m *Match) {
	now := time.Now()
	r := buildResult(m, now)
	r.Comp = h.fixtureTitle(m)
	r.Date = now.Format("2006-01-02")
	r.File = now.Format("150405") + "-" + m.id + ".json"
	dir := filepath.Join(resultsDir(), r.Date)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("결과 폴더 만들기 실패: %v", err)
		return
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	p := filepath.Join(dir, r.File)
	if err := os.WriteFile(p+".tmp", b, 0644); err != nil {
		log.Printf("결과 저장 실패: %v", err)
		return
	}
	if err := os.Rename(p+".tmp", p); err != nil {
		log.Printf("결과 저장 실패: %v", err)
	}
}

// validName 은 경로 조작을 막는다(영숫자·-·_·. 만, .. 금지).
func validName(name string) bool {
	if name == "" || len(name) > 64 || strings.Contains(name, "..") {
		return false
	}
	for _, r := range name {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.'
		if !ok {
			return false
		}
	}
	return true
}

// listResults 는 저장된 경기 결과를 최근 순으로 돌려준다. date 가 비어 있지 않으면 그 날짜만.
func listResults(date string) ([]MatchResult, error) {
	root := resultsDir()
	dirs, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []MatchResult{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []MatchResult{}
	for _, d := range dirs {
		if !d.IsDir() || (date != "" && d.Name() != date) {
			continue
		}
		files, err := os.ReadDir(filepath.Join(root, d.Name()))
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			b, err := os.ReadFile(filepath.Join(root, d.Name(), f.Name()))
			if err != nil {
				continue
			}
			var r MatchResult
			if json.Unmarshal(b, &r) != nil {
				continue
			}
			r.Date, r.File = d.Name(), f.Name() // 폴더를 옮겨도 실제 위치 기준
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FinishedAt > out[j].FinishedAt })
	return out, nil
}

func listDates() ([]string, error) {
	dirs, err := os.ReadDir(resultsDir())
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, d := range dirs {
		if d.IsDir() && validName(d.Name()) {
			out = append(out, d.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out, nil
}

func deleteResult(date, file string) error {
	if !validName(date) || !validName(file) || !strings.HasSuffix(file, ".json") {
		return errors.New("잘못된 경로")
	}
	return os.Remove(filepath.Join(resultsDir(), date, file))
}

var reasonKo = map[string]string{"topout": "판이 꽉 참", "time": "시간 종료", "golden": "연장(먼저 터뜨림)", "forfeit": "기권"}

// resultsCSV 는 한 경기 = 학생 한 줄씩(경기당 2줄) 엑셀에서 바로 열리는 CSV 를 만든다.
func resultsCSV(list []MatchResult) []byte {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // 엑셀이 UTF-8 로 인식하도록 BOM
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"날짜", "종료 시각", "학번", "이름", "결과", "터뜨린 카드", "상대가 터뜨린 카드",
		"최대 연쇄", "보낸 방해", "놓은 카드", "상대 학번", "상대 이름", "경기 시간(초)", "종료 사유", "대회"})
	for _, r := range list {
		t, _ := time.Parse(time.RFC3339, r.FinishedAt)
		for i, p := range r.Players {
			o := r.Players[1-i]
			_ = w.Write([]string{r.Date, t.Format("15:04:05"), p.SID, p.Name, p.Result,
				strconv.Itoa(p.Popped), strconv.Itoa(o.Popped), strconv.Itoa(p.MaxChain), strconv.Itoa(p.Sent), strconv.Itoa(p.Placed),
				o.SID, o.Name, strconv.Itoa(r.PlayedSec), reasonKo[r.Reason], r.Comp})
		}
	}
	w.Flush()
	return buf.Bytes()
}
