/* 인구 문제 카드 — 데이터 + 마크업 생성 (의존성 없음, 브라우저 전역 PopCards)
   cards.json 과 같은 내용. file:// 로 열어도 동작하도록 JS 로도 둠. */
(function () {
  var ICON = {
    dev: '<svg viewBox="0 0 30 26" fill="currentColor" aria-hidden="true"><rect x="9" y="0" width="12" height="4" rx="1"/><rect x="5" y="5.5" width="20" height="4" rx="1"/><rect x="3" y="11" width="24" height="4" rx="1"/><rect x="5" y="16.5" width="20" height="4" rx="1"/><rect x="8" y="22" width="14" height="4" rx="1"/></svg>',
    ing: '<svg viewBox="0 0 30 26" fill="currentColor" aria-hidden="true"><rect x="12" y="0" width="6" height="4" rx="1"/><rect x="9" y="5.5" width="12" height="4" rx="1"/><rect x="6" y="11" width="18" height="4" rx="1"/><rect x="3" y="16.5" width="24" height="4" rx="1"/><rect x="0" y="22" width="30" height="4" rx="1"/></svg>'
  };
  var LABEL = { dev: '선진국', ing: '개발도상국' };
  var CARDS = [
    { id: 'c01', dev: { text: '저출산' }, ing: { text: '높은 출생률' } },
    { id: 'c02', dev: { text: '고령화' }, ing: { text: '낮은 인구부양력' } },
    { id: 'c03', dev: { text: '외국인 근로자 유입' }, ing: { text: '가족계획 실시' } },
    { id: 'c04', dev: { text: '인구 정체 및 감소' }, ing: { text: '인구 지속 증가' } },
    { id: 'c05', dev: { image: 'images/graph.png', alt: '합계출산율 감소 그래프' }, ing: { image: 'images/poster.png', alt: '출산 억제 포스터' } },
    { id: 'c06', dev: { text: '합계출산율 감소' }, ing: { text: '높은 합계출산율' } },
    { id: 'c07', dev: { text: '고령 인구 증가' }, ing: { text: '출생 성비 불균형' } },
    { id: 'c08', dev: { text: '노인 부양 비용 증가' }, ing: { text: '식량 확보 필요' } },
    { id: 'c09', dev: { text: '청장년층 인구 비중 감소' }, ing: { text: '인구 증가 억제 필요' } },
    { id: 'c10', dev: { text: '노동력 부족' }, ing: { text: '빈곤과 기아' } }
  ];

  function esc(s) { return String(s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }

  function face(side, content, imageBase) {
    var body = content.image
      ? '<div class="pc-body pc-body--img"><img src="' + esc((imageBase || '') + content.image) + '" alt="' + esc(content.alt || '') + '"></div>'
      : '<div class="pc-body">' + esc(content.text) + '</div>';
    return '<div class="pc-face pc-face--' + side + '"><div class="pc-band"><span>' + LABEL[side] + '</span>' + ICON[side] + '</div>' + body + '</div>';
  }

  /* 카드 한 장의 HTML. opts: { side:'dev'|'ing', hidden:bool, index:number, imageBase:'' } */
  function cardHTML(card, opts) {
    opts = opts || {};
    var side = opts.side === 'ing' ? 'ing' : 'dev';
    return '<button type="button" class="pc-card" data-card-id="' + card.id + '"' +
      (opts.index != null ? ' data-index="' + opts.index + '"' : '') +
      ' data-side="' + side + '" data-hidden="' + (opts.hidden ? 'true' : 'false') + '"' +
      ' aria-label="' + LABEL[side] + ' 면">' +
      '<div class="pc-inner">' + face('dev', card.dev, opts.imageBase) + face('ing', card.ing, opts.imageBase) + '</div>' +
      '<div class="pc-cover" aria-hidden="true"><div class="pc-cover-mark">?</div><div class="pc-cover-label">인구 문제</div></div>' +
      '</button>';
  }

  /* 이미 그려진 카드의 면 바꾸기 (네트워크 동기화 시 이것만 호출) */
  function setSide(el, side) { el.dataset.side = side; el.setAttribute('aria-label', LABEL[side] + ' 면'); }

  /* 20장 덱: 10종 × 2벌 */
  function deck() { var d = []; CARDS.forEach(function (c) { d.push(c, c); }); return d; }

  window.PopCards = { CARDS: CARDS, LABEL: LABEL, cardHTML: cardHTML, setSide: setSide, deck: deck };
})();
