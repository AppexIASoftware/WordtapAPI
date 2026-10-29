package queries_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/gamecontent"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestGameContentRegression(t *testing.T) {
	custom := `[{"term":"  ","en":" Cat ","es":" Gato "},{"id":" chosen ","term":"Dog","translation":"Perro"},{"term":"invalid"}]`
	game := &entities.TeacherGame{CustomWordsJSON: &custom}
	invalid := `{"en":" ","term":"not used","es":"x"}`
	valid := `{"title":" Book ","translation":" Libro "}`
	lesson := &entities.Lesson{Items: []entities.LessonItem{{ID: "bad", ContentText: &invalid}, {ID: "good", SortOrder: 1, ContentText: &valid}}}
	words := gamecontent.Resolve(lesson, game)
	if len(words) != 1 || words[0].ID != "good" || words[0].Term != "Book" || words[0].Spanish != "Libro" {
		t.Fatalf("cards take precedence and filter invalid: %+v", words)
	}
	lesson.Items = lesson.Items[:1]
	words = gamecontent.Resolve(lesson, game)
	if len(words) != 2 || words[0].ID != "custom-1" || words[0].Term != "Cat" || words[1].ID != "chosen" {
		t.Fatalf("fallback aliases, filtering and IDs: %+v", words)
	}
	game.ItemCount = 1
	if words = gamecontent.Resolve(lesson, game); len(words) != 1 || words[0].Term != "Cat" {
		t.Fatalf("fallback item count: %+v", words)
	}
	malformed := `{`
	game.CustomWordsJSON = &malformed
	if words = gamecontent.Resolve(lesson, game); words == nil || len(words) != 0 {
		t.Fatalf("malformed input must yield empty slice: %+v", words)
	}
}

func TestGlobalGameUsesItsOwnContent(t *testing.T) {
	card := `{"en":"lesson","es":"lección"}`
	lesson := &entities.Lesson{Items: []entities.LessonItem{{ID: "card", ContentText: &card}}}
	custom := `[{"term":"global","translation":"global"}]`
	for _, tc := range []struct {
		name string
		game entities.TeacherGame
		want string
	}{
		{"global uses selected content", entities.TeacherGame{IsGlobal: true, CustomWordsJSON: &custom}, "global"},
		{"lesson uses lesson cards", entities.TeacherGame{CustomWordsJSON: &custom}, "lesson"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := gamecontent.Resolve(lesson, &tc.game)
			if len(got) != 1 || got[0].Term != tc.want {
				t.Fatalf("got %+v, want %s", got, tc.want)
			}
		})
	}
	if got := gamecontent.Resolve(lesson, &entities.TeacherGame{IsGlobal: true}); len(got) != 0 {
		t.Fatalf("global game with no content must not use lesson cards: %+v", got)
	}
}

func TestGameContentCardFormatsAndOrdering(t *testing.T) {
	cases := []struct{ name, payload, term, translation string }{
		{"language aliases", `{"en":" hello ","es":" hola ","term":"ignored","translation":"ignored"}`, "hello", "hola"},
		{"term", `{"term":" hello ","translation":" hola "}`, "hello", "hola"},
		{"title", `{"title":" hello ","translation":" hola "}`, "hello", "hola"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lesson := &entities.Lesson{Items: []entities.LessonItem{{ID: "card", ContentText: &tc.payload}}}
			words := gamecontent.Resolve(lesson, nil)
			if len(words) != 1 || words[0].ID != "card" || words[0].Term != tc.term || words[0].Translation != tc.translation || words[0].English != tc.term || words[0].Spanish != tc.translation {
				t.Fatalf("unexpected words: %+v", words)
			}
		})
	}
	lesson := &entities.Lesson{Items: []entities.LessonItem{
		{ID: "later", SortOrder: 2, VocabularyItem: &entities.VocabularyItem{English: "later", Spanish: "después"}},
		{ID: "b", SortOrder: 1, VocabularyItem: &entities.VocabularyItem{English: "b", Spanish: "b"}},
		{ID: "a", SortOrder: 1, VocabularyItem: &entities.VocabularyItem{English: "a", Spanish: "a"}},
	}}
	words := gamecontent.Resolve(lesson, nil)
	if len(words) != 3 || words[0].Term != "a" || words[1].Term != "b" || words[2].Term != "later" || lesson.Items[0].ID != "later" {
		t.Fatalf("ordering or mutation: %+v", words)
	}
	lesson.Items[1], lesson.Items[2] = lesson.Items[2], lesson.Items[1]
	if again := gamecontent.Resolve(lesson, nil); len(again) != 3 || again[0].Term != "a" || again[1].Term != "b" {
		t.Fatalf("ordering depends on input order: %+v", again)
	}
	if empty := gamecontent.Resolve(nil, nil); empty == nil || len(empty) != 0 {
		t.Fatalf("nil inputs: %+v", empty)
	}
}

func TestGameContentInlineAndFallbackCases(t *testing.T) {
	custom := `[{"id":"  ","term":" preferred ","translation":" primero ","en":"ignored","es":"ignored"},{"term":" ","en":" alternate ","translation":" ","es":" alterno "},{"id":"valid","term":" ","en":" ","es":"invalid"},{"term":"invalid","translation":" "}]`
	fallback := []gamecontent.Word{{ID: "custom-1", Term: "preferred", Translation: "primero", English: "preferred", Spanish: "primero"}, {ID: "custom-2", Term: "alternate", Translation: "alterno", English: "alternate", Spanish: "alterno"}}
	game := &entities.TeacherGame{CustomWordsJSON: &custom}
	for _, payload := range []string{"", "  ", `{`, `{"en":"hello"}`, `{"es":"hola"}`, `{"en":42,"es":"hola"}`, `{"en":" ","term":"ignored","es":"hola"}`, `{"en":"hello","es":" ","translation":"ignored"}`} {
		t.Run("fallback/"+payload, func(t *testing.T) {
			got := gamecontent.Resolve(&entities.Lesson{Items: []entities.LessonItem{{ID: "inline", ContentText: &payload}}}, game)
			if !reflect.DeepEqual(got, fallback) {
				t.Fatalf("got %+v, want %+v", got, fallback)
			}
		})
	}
	valid := `{"term":"card","translation":"tarjeta"}`
	invalid := `{"en":"missing translation"}`
	lesson := &entities.Lesson{Items: []entities.LessonItem{{ID: "bad", ContentText: &invalid}, {ID: "good", ContentText: &valid}}}
	want := []gamecontent.Word{{ID: "good", Term: "card", Translation: "tarjeta", English: "card", Spanish: "tarjeta"}}
	if got := gamecontent.Resolve(lesson, game); !reflect.DeepEqual(got, want) {
		t.Fatalf("one valid card suppresses all custom words: %+v", got)
	}
	if got := gamecontent.Resolve(nil, game); !reflect.DeepEqual(got, fallback) {
		t.Fatalf("nil lesson fallback: %+v", got)
	}
	if got := gamecontent.Resolve(lesson, nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("nil game: %+v", got)
	}
	for _, raw := range []string{"", "  ", `{`, `[{"term":"ok","translation":3}]`, `{"term":"x"}`} {
		t.Run("invalid custom/"+raw, func(t *testing.T) {
			got := gamecontent.Resolve(nil, &entities.TeacherGame{CustomWordsJSON: &raw})
			if got == nil || len(got) != 0 {
				t.Fatalf("expected empty non-nil words: %+v", got)
			}
		})
	}
	if got := gamecontent.Resolve(nil, &entities.TeacherGame{}); got == nil || len(got) != 0 {
		t.Fatalf("nil custom JSON: %+v", got)
	}
}

func TestGameContentItemCountBoundaries(t *testing.T) {
	custom := `[{"term":"one","translation":"uno"},{"term":"two","translation":"dos"}]`
	lesson := &entities.Lesson{Items: []entities.LessonItem{{ID: "a", VocabularyItem: &entities.VocabularyItem{English: "one", Spanish: "uno"}}, {ID: "b", VocabularyItem: &entities.VocabularyItem{English: "two", Spanish: "dos"}}}}
	for _, tc := range []struct{ count, want int }{{-1, 2}, {0, 2}, {1, 1}, {2, 2}, {3, 2}} {
		t.Run(fmt.Sprintf("count=%d", tc.count), func(t *testing.T) {
			game := &entities.TeacherGame{ItemCount: tc.count, CustomWordsJSON: &custom}
			for _, source := range []*entities.Lesson{lesson, nil} {
				got := gamecontent.Resolve(source, game)
				if len(got) != tc.want || got[0].Term != "one" {
					t.Fatalf("count %d: %+v", tc.count, got)
				}
			}
		})
	}
}

func TestGameContentReferencePrecedence(t *testing.T) {
	inline := `{"en":"inline","es":"linea"}`
	vocab := &entities.VocabularyItem{ID: "v", English: " vocab ", Spanish: " vocab-es "}
	phrase := &entities.Phrase{ID: "p", English: " phrase ", Spanish: " phrase-es "}
	vault := &entities.ContentBankItem{ID: "b", TargetText: " vault ", PromptText: " vault-es "}
	custom := `[{"term":"fallback","translation":"respaldo"}]`
	game := &entities.TeacherGame{CustomWordsJSON: &custom}
	cases := []struct {
		name                  string
		item                  entities.LessonItem
		id, term, translation string
	}{
		{"vocabulary before all", entities.LessonItem{VocabularyItem: vocab, Phrase: phrase, ContentBankItem: vault, ContentText: &inline}, "v", "vocab", "vocab-es"},
		{"phrase before vault and inline", entities.LessonItem{Phrase: phrase, ContentBankItem: vault, ContentText: &inline}, "p", "phrase", "phrase-es"},
		{"vault before inline", entities.LessonItem{ContentBankItem: vault, ContentText: &inline}, "b", "vault", "vault-es"},
		{"inline", entities.LessonItem{ContentText: &inline}, "item", "inline", "linea"},
		{"empty reference ID uses item ID", entities.LessonItem{VocabularyItem: &entities.VocabularyItem{English: "word", Spanish: "palabra"}}, "item", "word", "palabra"},
		{"invalid vocabulary blocks other references", entities.LessonItem{VocabularyItem: &entities.VocabularyItem{English: " ", Spanish: "x"}, Phrase: phrase, ContentBankItem: vault, ContentText: &inline}, "custom-1", "fallback", "respaldo"},
		{"invalid phrase blocks vault and inline", entities.LessonItem{Phrase: &entities.Phrase{English: "x", Spanish: " "}, ContentBankItem: vault, ContentText: &inline}, "custom-1", "fallback", "respaldo"},
		{"invalid vault blocks inline", entities.LessonItem{ContentBankItem: &entities.ContentBankItem{TargetText: " ", PromptText: "x"}, ContentText: &inline}, "custom-1", "fallback", "respaldo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.item.ID = "item"
			got := gamecontent.Resolve(&entities.Lesson{Items: []entities.LessonItem{tc.item}}, game)
			want := []gamecontent.Word{{ID: tc.id, Term: tc.term, Translation: tc.translation, English: tc.term, Spanish: tc.translation}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %+v, want %+v", got, want)
			}
		})
	}
}
