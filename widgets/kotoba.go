package widgets

import (
	"fmt"
	"html/template"
)

// AllWords returns the full JLPT word list used for server-side
// rendering and the /api/kotoba/next endpoint.
func AllWords() []Word {
	return words
}

// words is the full JLPT vocabulary pool (N5–N1).
var words = []Word{
	// N5
	{K: "水", R: "みず", M: "water", L: "n5"},
	{K: "火", R: "ひ", M: "fire", L: "n5"},
	{K: "山", R: "やま", M: "mountain", L: "n5"},
	{K: "川", R: "かわ", M: "river", L: "n5"},
	{K: "空", R: "そら", M: "sky", L: "n5"},
	{K: "花", R: "はな", M: "flower", L: "n5"},
	{K: "木", R: "き", M: "tree, wood", L: "n5"},
	{K: "石", R: "いし", M: "stone, rock", L: "n5"},
	{K: "道", R: "みち", M: "road, path", L: "n5"},
	{K: "手", R: "て", M: "hand", L: "n5"},
	{K: "目", R: "め", M: "eye", L: "n5"},
	{K: "口", R: "くち", M: "mouth", L: "n5"},
	{K: "耳", R: "みみ", M: "ear", L: "n5"},
	{K: "足", R: "あし", M: "foot, leg", L: "n5"},
	{K: "心", R: "こころ", M: "heart, mind", L: "n5"},
	{K: "雨", R: "あめ", M: "rain", L: "n5"},
	{K: "風", R: "かぜ", M: "wind", L: "n5"},
	{K: "月", R: "つき", M: "moon, month", L: "n5"},
	{K: "日", R: "ひ", M: "sun, day", L: "n5"},
	{K: "年", R: "とし", M: "year", L: "n5"},
	{K: "語", R: "ご", M: "word, language", L: "n5"},
	{K: "勉強", R: "べんきょう", M: "study", L: "n5"},
	{K: "漢字", R: "かんじ", M: "kanji", L: "n5"},
	{K: "時間", R: "じかん", M: "time", L: "n5"},
	{K: "友達", R: "ともだち", M: "friend", L: "n5"},
	{K: "言葉", R: "ことば", M: "word, language", L: "n5"},
	{K: "本", R: "ほん", M: "book", L: "n5"},
	// N4
	{K: "橋", R: "はし", M: "bridge", L: "n4"},
	{K: "庭", R: "にわ", M: "garden", L: "n4"},
	{K: "池", R: "いけ", M: "pond", L: "n4"},
	{K: "窓", R: "まど", M: "window", L: "n4"},
	{K: "声", R: "こえ", M: "voice", L: "n4"},
	{K: "夢", R: "ゆめ", M: "dream", L: "n4"},
	{K: "光", R: "ひかり", M: "light", L: "n4"},
	{K: "影", R: "かげ", M: "shadow", L: "n4"},
	{K: "森", R: "もり", M: "forest", L: "n4"},
	{K: "海", R: "うみ", M: "sea, ocean", L: "n4"},
	{K: "星", R: "ほし", M: "star", L: "n4"},
	{K: "雪", R: "ゆき", M: "snow", L: "n4"},
	{K: "波", R: "なみ", M: "wave", L: "n4"},
	{K: "鳥", R: "とり", M: "bird", L: "n4"},
	{K: "葉", R: "は", M: "leaf", L: "n4"},
	{K: "根", R: "ね", M: "root", L: "n4"},
	{K: "岩", R: "いわ", M: "rock, boulder", L: "n4"},
	{K: "砂", R: "すな", M: "sand", L: "n4"},
	// N3
	{K: "霧", R: "きり", M: "fog, mist", L: "n3"},
	{K: "崖", R: "がけ", M: "cliff", L: "n3"},
	{K: "滝", R: "たき", M: "waterfall", L: "n3"},
	{K: "峰", R: "みね", M: "peak, summit", L: "n3"},
	{K: "縁", R: "えん", M: "fate, connection", L: "n3"},
	{K: "鏡", R: "かがみ", M: "mirror", L: "n3"},
	{K: "扉", R: "とびら", M: "door, gate", L: "n3"},
	{K: "涙", R: "なみだ", M: "tear, teardrop", L: "n3"},
	{K: "傷", R: "きず", M: "wound, scar", L: "n3"},
	{K: "嘘", R: "うそ", M: "lie, falsehood", L: "n3"},
	{K: "誓", R: "ちかい", M: "oath, vow", L: "n3"},
	{K: "憎", R: "にくしみ", M: "hatred", L: "n3"},
	{K: "怒", R: "いかり", M: "anger, rage", L: "n3"},
	{K: "罪", R: "つみ", M: "crime, sin, guilt", L: "n3"},
	{K: "罰", R: "ばつ", M: "punishment", L: "n3"},
	{K: "闇", R: "やみ", M: "darkness", L: "n3"},
	// N2
	{K: "儚", R: "はかない", M: "fleeting, ephemeral", L: "n2"},
	{K: "憐", R: "あわれ", M: "pity, compassion", L: "n2"},
	{K: "憧", R: "あこがれ", M: "longing, yearning", L: "n2"},
	{K: "諦", R: "あきらめ", M: "resignation, giving up", L: "n2"},
	{K: "葛", R: "かつら", M: "arrowroot; conflict", L: "n2"},
	{K: "懐", R: "なつかしい", M: "nostalgic, dear", L: "n2"},
	{K: "煩", R: "わずらわしい", M: "troublesome, vexing", L: "n2"},
	{K: "濁", R: "にごり", M: "muddiness, impurity", L: "n2"},
	{K: "彷", R: "さまよい", M: "wandering", L: "n2"},
	// N1
	{K: "刹那", R: "せつな", M: "moment, instant", L: "n1"},
	{K: "無常", R: "むじょう", M: "impermanence", L: "n1"},
	{K: "幽玄", R: "ゆうげん", M: "subtle grace, mystery", L: "n1"},
	{K: "侘寂", R: "わびさび", M: "beauty in imperfection", L: "n1"},
	{K: "物哀", R: "もののあわれ", M: "pathos of things", L: "n1"},
	{K: "諦観", R: "ていかん", M: "resignation, detachment", L: "n1"},
	{K: "執着", R: "しゅうちゃく", M: "attachment, obsession", L: "n1"},
	{K: "業", R: "ごう", M: "karma, fate", L: "n1"},
	{K: "虚無", R: "きょむ", M: "nihilism, void", L: "n1"},
	{K: "断絶", R: "だんぜつ", M: "severance, rupture", L: "n1"},
}

type KotobaWidget struct{}

func (w *KotobaWidget) ID() string { return "kotoba" }

func (w *KotobaWidget) Render(ctx RenderContext) (template.HTML, error) {
	word := words[ctx.RNG.Intn(len(words))]
	inner := fmt.Sprintf(`
<div class="kanji-block">
  <div class="kanji-char" id="kanji-char">
    <a href="https://jisho.org/search/%s%%20%%23kanji"
       style="color:inherit;text-decoration:none;">%s</a>
  </div>
  <div class="kanji-reading" id="kanji-reading">%s</div>
  <div class="kanji-divider"></div>
  <div class="kanji-meta">
    <span class="kanji-level %s" id="kanji-level">%s</span>
  </div>
  <div class="kanji-meaning" id="kanji-meaning">%s</div>
</div>`, word.K, word.K, word.R, word.L, word.L, word.M)
	return wrap(ctx, "kotoba", "言葉",
		`<button class="wt-act" data-preload="/api/kotoba/next" data-preload-render="render_kotoba"><i class="ph-light ph-caret-right"></i></button>`,
		inner), nil
}


