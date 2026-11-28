package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Jisin0/autofilterbot/internal/captiontags"
	"github.com/Jisin0/autofilterbot/internal/model"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

type captionAudit struct {
	videos, saved, labeled, needsReview, unknown int
	samples                                      []string
}

func (a *captionAudit) add(file model.File) {
	name := sourceFileName(file)
	if !canHaveMovieAudio(file) {
		return
	}
	if !captiontags.IsVideoUpload(name, file.FileType == model.FileTypeVideo) {
		a.unknown++
		if file.Caption == nil {
			a.addSample(file)
		}
		return
	}
	a.videos++
	if file.Caption != nil {
		a.saved++
	}
	labeled := len(detectedAudio(file, false)) > 0
	if labeled {
		a.labeled++
	}
	if file.Caption == nil && !labeled {
		a.needsReview++
		a.addSample(file)
	}
}

func (a *captionAudit) addSample(file model.File) {
	if len(a.samples) < 8 {
		name := sourceFileName(file)
		runes := []rune(name)
		if len(runes) > 110 {
			name = string(runes[:110]) + "…"
		}
		if file.MessageLink != "" {
			name += "\n" + file.MessageLink
		}
		a.samples = append(a.samples, name)
	}
}

func CaptionAudit(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !privateAdminOnly(bot, ctx) || ctx.Message == nil {
		return nil
	}
	query := parseMovieSearch(commandArgument(ctx.Message.Text))
	scanContext, cancel := context.WithTimeout(_app.Ctx, 30*time.Second)
	defer cancel()
	var audit captionAudit
	err := _app.DB.VisitCaptionAuditFiles(scanContext, query.title, query.year, audit.add)
	summary := fmt.Sprintf("Caption audit\nRecognizable indexed videos: %d\nSaved caption metadata: %d\nAudio labels in filenames or captions: %d\nMissing caption and audio label: %d\nOther candidates with unconfirmed media type: %d",
		audit.videos, audit.saved, audit.labeled, audit.needsReview, audit.unknown)
	if err != nil {
		_app.Log.Warn("caption audit scan incomplete", zap.Error(err))
		summary += "\nPartial scan. Use /captionaudit <title> to narrow the search."
	}
	if len(audit.samples) > 0 {
		summary += "\n\nExamples to review with /refreshmalayalamsub\n" + strings.Join(audit.samples, "\n\n")
	}
	_, err = ctx.Message.Reply(bot, summary, &gotgbot.SendMessageOpts{LinkPreviewOptions: &gotgbot.LinkPreviewOptions{IsDisabled: true}})
	return err
}

func commandArgument(text string) string {
	command := strings.Fields(text)
	if len(command) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(text, command[0]))
}
