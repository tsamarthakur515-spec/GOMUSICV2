package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func entityDump(ents []telegram.MessageEntity) string {
	if len(ents) == 0 {
		return "(no entities)"
	}
	var b strings.Builder
	for i, e := range ents {
		switch t := e.(type) {
		case *telegram.MessageEntityBlockquote:
			fmt.Fprintf(&b, "%d: blockquote collapsed=%v offset=%d length=%d\n", i, t.Collapsed, t.Offset, t.Length)
		default:
			fmt.Fprintf(&b, "%d: %T %+v\n", i, e, e)
		}
	}
	return strings.TrimSpace(b.String())
}

func msgEntities(msg *telegram.NewMessage) []telegram.MessageEntity {
	if msg == nil || msg.Message == nil {
		return nil
	}
	return msg.Message.Entities
}

func handleDebugBQ(m *telegram.NewMessage) error {
	if m == nil || !isOwner(userIDOf(m)) {
		return nil
	}
	chatID := m.ChatID()
	htmlText := wrapBQ(smallcaps("debug blockquote") + "\n\n" +
		smallcaps("line two") + "\n" +
		smallcaps("line three") + "\n" +
		smallcaps("line four"))
	parsed, plain := captionEntities(htmlText)

	var report strings.Builder
	report.WriteString("<b>blockquote debug</b>\n\n")
	fmt.Fprintf(&report, "html has tag: <code>%v</code>\n", strings.Contains(htmlText, "blockquote"))
	fmt.Fprintf(&report, "parsed has bq: <code>%v</code>\n", hasBlockquote(parsed))
	fmt.Fprintf(&report, "plain utf16: <code>%d</code>\n", utf16Count(plain))
	report.WriteString("<pre>" + richEsc(entityDump(parsed)) + "</pre>\n\n")

	textMsg, err := Bot.SendMessage(chatID, htmlText, htmlSendOpts(nil))
	fmt.Fprintf(&report, "1 send text err=<code>%v</code> sent_bq=<code>%v</code>\n", err, hasBlockquote(msgEntities(textMsg)))
	report.WriteString("<pre>" + richEsc(entityDump(msgEntities(textMsg))) + "</pre>\n\n")

	if textMsg != nil {
		editHTML2 := wrapBQ(smallcaps("edited text quote") + "\n\n" + smallcaps("still four lines here ok"))
		e2, _ := captionEntities(editHTML2)
		_, eerr := Bot.EditMessage(chatID, textMsg.ID, editHTML2, &telegram.SendOptions{ParseMode: "HTML", Entities: e2})
		got, _ := textMsg.Client.GetMessageByID(chatID, textMsg.ID)
		fmt.Fprintf(&report, "2 edit text err=<code>%v</code> sent_bq=<code>%v</code>\n", eerr, hasBlockquote(msgEntities(got)))
		report.WriteString("<pre>" + richEsc(entityDump(msgEntities(got))) + "</pre>\n\n")
	}

	photo := ""
	if len(StartPhotos) > 0 {
		photo = StartPhotos[0]
	}
	var photoMsg *telegram.NewMessage
	var perr error
	if photo != "" {
		photoMsg, perr = Bot.SendMedia(chatID, photo, htmlMediaOpts(htmlText, nil))
	}
	fmt.Fprintf(&report, "3 send photo err=<code>%v</code> sent_bq=<code>%v</code>\n", perr, hasBlockquote(msgEntities(photoMsg)))
	report.WriteString("<pre>" + richEsc(entityDump(msgEntities(photoMsg))) + "</pre>\n\n")

	if photoMsg != nil {
		editCap := wrapBQ(smallcaps("edited photo caption") + "\n\n" + smallcaps("help menu style text"))
		e3, _ := captionEntities(editCap)
		_, eerr := Bot.EditMessage(chatID, photoMsg.ID, editCap, &telegram.SendOptions{ParseMode: "HTML", Entities: e3})
		got, _ := photoMsg.Client.GetMessageByID(chatID, photoMsg.ID)
		fmt.Fprintf(&report, "4 edit photo caption err=<code>%v</code> sent_bq=<code>%v</code>\n", eerr, hasBlockquote(msgEntities(got)))
		report.WriteString("<pre>" + richEsc(entityDump(msgEntities(got))) + "</pre>\n")
	}

	out := wrapBQ(strings.TrimSpace(report.String()))
	if _, err := sendHTML(Bot, chatID, out, nil); err != nil {
		log.Println("debugbq report:", err)
		_, _ = Bot.SendMessage(chatID, htmlToPlain(out), nil)
	}
	return nil
}
