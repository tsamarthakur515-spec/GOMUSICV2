package main

import (
	"testing"

	"github.com/amarnathcjd/gogram/telegram"
)

func TestCaptionEntitiesPreserveBlockquote(t *testing.T) {
	captions := []string{
		startCaption(123, "Test User"),
		aboutCaption(),
		helpListCaption(123, "Test User"),
		wrapBQ(smallcaps("admin commands") + "\n\n" + smallcaps("description") + "\n\n" + richTable(
			[]string{"command", "description"},
			[][]string{{"/play", "play audio"}},
		)),
	}

	for i, caption := range captions {
		entities, _ := captionEntities(caption)
		foundBlockquote := false
		lastOffset := int32(-1)
		for _, entity := range entities {
			offset, _ := entityBounds(entity)
			if offset < lastOffset {
				t.Fatalf("caption %d entities are not sorted by offset", i)
			}
			lastOffset = offset
			if blockquote, ok := entity.(*telegram.MessageEntityBlockquote); ok {
				foundBlockquote = true
				if blockquote.Collapsed || blockquote.Offset != 0 || blockquote.Length <= 0 {
					t.Fatalf("caption %d has invalid blockquote bounds: %+v", i, blockquote)
				}
			}
		}
		if !foundBlockquote {
			t.Fatalf("caption %d lost its blockquote entity", i)
		}
	}
}

func TestMixedKeyboardAppliesButtonStyles(t *testing.T) {
	markup, ok := mixedKeyboard([][][2]string{
		{{"one", "one"}, {"two", "two"}},
		{{"link", "https://example.com"}},
	}).(*telegram.ReplyInlineMarkup)
	if !ok {
		t.Fatal("mixedKeyboard did not return inline markup")
	}

	for rowIndex, row := range markup.Rows {
		for buttonIndex, button := range row.Buttons {
			if button.Style == nil {
				t.Fatalf("button %d/%d has no color style", rowIndex, buttonIndex)
			}
		}
	}
}
