"""인구 뿌요 아이콘 생성기 — python3 gen.py 로 icon.svg / icon-small.svg 를 다시 만든다.
래스터·ico 는 README(Vector_Soccer 와 같은) magick 명령으로."""

GRAD = '''  <defs>
    <linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%"   stop-color="#6366f1"/>
      <stop offset="50%"  stop-color="#8b5cf6"/>
      <stop offset="100%" stop-color="#06b6d4"/>
    </linearGradient>
  </defs>

  <rect x="112" y="112" width="800" height="800" rx="184" fill="url(#g)"/>
'''

BLUE = ("#8DBBF5", "#4f86d6")     # 선진국
ORANGE = ("#FFBB6B", "#e5862a")   # 개발도상국
EYE = "#3b3355"

def jelly(x, y, s, col, face=True, rot=0, eye=1.0):
    """게임 블록과 같은 젤리: 둥근 사각형(rx 30%) · 아래 진한 띠 · 왼쪽 위 흰 광택 · 오른쪽 위 눈 · 볼터치."""
    fill, edge = col
    r = s * 0.30
    cx, cy = x + s/2, y + s/2
    g = f'  <g transform="rotate({rot} {cx:.0f} {cy:.0f})">\n' if rot else '  <g>\n'
    g += f'    <rect x="{x:.0f}" y="{y:.0f}" width="{s:.0f}" height="{s:.0f}" rx="{r:.0f}" fill="{edge}"/>\n'
    g += f'    <rect x="{x:.0f}" y="{y:.0f}" width="{s:.0f}" height="{s*0.9:.0f}" rx="{r:.0f}" fill="{fill}"/>\n'
    g += (f'    <ellipse cx="{x+s*0.30:.0f}" cy="{y+s*0.24:.0f}" rx="{s*0.15:.0f}" ry="{s*0.085:.0f}" '
          f'fill="#ffffff" opacity="0.85" transform="rotate(-25 {x+s*0.30:.0f} {y+s*0.24:.0f})"/>\n')
    if face:
        for ex in ((0.56, 0.76) if eye == 1.0 else (0.52, 0.78)):
            g += f'    <ellipse cx="{x+s*ex:.0f}" cy="{y+s*0.42:.0f}" rx="{s*0.055*eye:.0f}" ry="{s*0.075*eye:.0f}" fill="{EYE}"/>\n'
        for bx in ((0.46, 0.86) if eye == 1.0 else ()):   # 작은 아이콘은 볼터치 생략(뭉개짐)
            g += f'    <ellipse cx="{x+s*bx:.0f}" cy="{y+s*0.58:.0f}" rx="{s*0.07:.0f}" ry="{s*0.04:.0f}" fill="#ff8fb8" opacity="0.6"/>\n'
    return g + '  </g>\n'

def sparkle(x, y, r):
    """네 갈래 반짝이."""
    k = r * 0.22
    return (f'  <path d="M{x} {y-r} Q{x+k} {y-k} {x+r} {y} Q{x+k} {y+k} {x} {y+r} '
            f'Q{x-k} {y+k} {x-r} {y} Q{x-k} {y-k} {x} {y-r} Z" fill="#ffffff"/>\n')

def big():
    ex, ey, er = 256, 600, 72           # 죽방 머리 (VS 와 같은 비율: 밑변 172 · 높이 156)
    s = 196                              # 블록 크기
    gx = 386; floor = 804                # 쌓인 블록 왼쪽 · 바닥선
    svg = f'''<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!--
    인구 뿌요(population puyo) 아이콘 — 큰 크기용(256px 이상). gen.py 로 생성.

    teaveloper 공용 규격: 800x800 라운드 사각형(rx 184) · 공식 그라데이션 · 흰색 죽방.

    이 앱만의 것: 죽방 옆에 젤리 블록(선진국 파랑 · 개발도상국 주황)이 쌓이고,
    위에서 하나가 떨어지는 중. 작은 크기(16·32·48px)는 블록 3개만 남긴 icon-small.svg.
  -->
{GRAD}
  <!-- 죽방 — 원 r{er} · 밑변 172 · 높이 156 (좁고 길면 열쇠구멍으로 보인다) -->
  <g fill="#ffffff">
    <circle cx="{ex}" cy="{ey}" r="{er}"/>
    <path d="M{ex} {ey+48} L{ex-86} {ey+204} L{ex+86} {ey+204} Z"/>
  </g>

  <!-- 쌓인 블록 -->
'''
    svg += jelly(gx, floor - s, s, BLUE)
    svg += jelly(gx + s + 14, floor - s, s, ORANGE)
    svg += jelly(gx + s + 14, floor - 2*s - 14, s, ORANGE)
    svg += '\n  <!-- 떨어지는 블록 + 낙하 줄 -->\n'
    fx, fy = gx + 4, 268
    svg += '  <g stroke="#ffffff" stroke-width="12" stroke-linecap="round" opacity="0.7">\n'
    for dx, l in ((0.3, 56), (0.7, 56)):
        svg += f'    <line x1="{fx+s*dx:.0f}" y1="{fy-34:.0f}" x2="{fx+s*dx:.0f}" y2="{fy-34-l:.0f}"/>\n'
    svg += '  </g>\n'
    svg += jelly(fx, fy, s, BLUE, rot=-8)
    svg += '\n  <!-- 반짝이 -->\n'
    svg += sparkle(250, 330, 50) + sparkle(326, 450, 24) + sparkle(800, 280, 38)
    return svg + '</svg>\n'

def small():
    s, gap = 330, 20
    x0 = 512 - s - gap/2; x1 = 512 + gap/2
    y0 = 512 - s - gap/2; y1 = 512 + gap/2
    svg = f'''<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!--
    인구 뿌요 아이콘 — 작은 크기용(16·32·48px). gen.py 로 생성.
    16px 에 죽방·반짝이까지 넣으면 뭉개진다. 젤리 블록 3개만 크게, 눈은 키우고 볼터치는 뺐다.
  -->
{GRAD}
'''
    svg += jelly(x1, y0, s, ORANGE, eye=1.7)
    svg += jelly(x0, y1, s, BLUE, eye=1.7)
    svg += jelly(x1, y1, s, ORANGE, eye=1.7)
    return svg + '</svg>\n'

open("icon.svg", "w").write(big())
open("icon-small.svg", "w").write(small())
