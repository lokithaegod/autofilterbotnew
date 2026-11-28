package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Jisin0/autofilterbot/internal/captiontags"
	"github.com/Jisin0/autofilterbot/internal/database/mongo"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

func availabilityQuery(user int64, query string) mongo.AvailabilityAlert {
	search := parseMovieSearch(strings.ToLower(strings.Join(strings.Fields(query), " ")))
	return mongo.AvailabilityAlert{UserID: user, Title: search.title, Year: search.year, Audio: search.audio}
}

func availabilityLabel(alert mongo.AvailabilityAlert) string {
	label := alert.Title
	if alert.Year != "" {
		label += " " + alert.Year
	}
	if alert.Audio != "" {
		label += " " + captiontags.LanguageLabel(alert.Audio) + " audio"
	}
	return label
}

func ManageAvailability(bot *gotgbot.Bot, ctx *ext.Context) error {
	m := ctx.Message
	if m == nil || m.From == nil || m.Chat.Type != "private" || !CheckSearchAccess(bot, ctx) {
		return nil
	}
	command := strings.SplitN(strings.Fields(m.Text)[0], "@", 2)[0]
	arg := commandArgument(m.Text)
	dbContext, cancel := context.WithTimeout(_app.Ctx, 8*time.Second)
	defer cancel()
	if command == "/alerts" {
		alerts, err := _app.DB.AvailabilityAlerts(dbContext, m.From.Id, 26)
		if err != nil {
			return err
		}
		response := "No pending availability alerts. Add one with /notify Hunger Games 2023 Tamil."
		if len(alerts) > 0 {
			response = "Availability alerts"
			for _, a := range alerts[:min(25, len(alerts))] {
				response += "\n• " + availabilityLabel(a)
				if a.DeliveryStarted {
					response += " (delivery unconfirmed)"
				}
			}
			if len(alerts) > 25 {
				response += "\nMore alerts are pending."
			}
			response += "\n\nCancel with /unnotify followed by the same search."
		}
		_, err = m.Reply(bot, response, nil)
		return err
	}
	if len(arg) < 2 || len(arg) > 120 {
		_, err := m.Reply(bot, "Usage: "+command+" Hunger Games 2023 Tamil\nYear and audio language are optional.", nil)
		return err
	}
	alert := availabilityQuery(m.From.Id, arg)
	if alert.Title == "" {
		return nil
	}
	if command == "/unnotify" {
		removed, err := _app.DB.RemoveAvailabilityAlert(dbContext, alert)
		if err != nil {
			return err
		}
		response := "No matching pending alert."
		if removed {
			response = "Alert cancelled for " + availabilityLabel(alert) + "."
		}
		_, err = m.Reply(bot, response, nil)
		return err
	}
	saved, err := saveAvailability(dbContext, alert)
	if err != nil {
		return err
	}
	_, err = m.Reply(bot, saved, nil)
	return err
}

func saveAvailability(ctx context.Context, alert mongo.AvailabilityAlert) (string, error) {
	alerts, err := _app.DB.AvailabilityAlerts(ctx, alert.UserID, 25)
	if err != nil {
		return "", err
	}
	for _, current := range alerts {
		if current.Title == alert.Title && current.Year == alert.Year && current.Audio == alert.Audio {
			if current.DeliveryStarted {
				return "Delivery was attempted for " + availabilityLabel(alert) + ". Use /unnotify and then /notify to try again.", nil
			}
			return "You already have an alert for " + availabilityLabel(alert) + ".", nil
		}
	}
	if len(alerts) >= 25 {
		return "You have 25 pending alerts. Cancel one with /unnotify before adding another.", nil
	}
	if err := _app.DB.SaveAvailabilityAlert(ctx, alert); err != nil {
		return "", err
	}
	return "Alert saved for " + availabilityLabel(alert) + ". If it’s available, I’ll send one private message between 7 AM and 1 AM IST. Use /alerts to view pending alerts.", nil
}

func AvailabilityCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	q := ctx.CallbackQuery
	if q == nil || q.Message == nil || q.Message.GetChat().Type != "private" {
		return nil
	}
	query := strings.TrimPrefix(q.Data, "notify|")
	if len(query) < 2 || len(query) > 120 {
		return nil
	}
	blocked, err := _app.DB.IsUserBlocked(q.From.Id)
	if err != nil || blocked || !SearchAllowed(q.From.Id) {
		_, err = q.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Alerts are temporarily unavailable.", ShowAlert: true})
		return err
	}
	dbContext, cancel := context.WithTimeout(_app.Ctx, 8*time.Second)
	defer cancel()
	reply, err := saveAvailability(dbContext, availabilityQuery(q.From.Id, query))
	if err != nil {
		_, _ = q.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Could not save the alert. Please retry.", ShowAlert: true})
		return err
	}
	_, _ = q.Answer(bot, nil)
	_, err = bot.SendMessage(q.From.Id, reply, nil)
	return err
}

var availabilityIST = time.FixedZone("IST", 5*60*60+30*60)

func availabilityHours(now time.Time) bool {
	hour := now.In(availabilityIST).Hour()
	return hour >= 7 || hour < 1
}

func runAvailabilityAlerts(ctx context.Context, bot *gotgbot.Bot) {
	setup, cancel := context.WithTimeout(ctx, 10*time.Second)
	if err := _app.DB.EnsureAvailabilityIndexes(setup); err != nil {
		_app.Log.Warn("availability alert indexes unavailable", zap.Error(err))
	}
	cancel()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if availabilityHours(time.Now()) {
				if err := checkAvailabilityAlerts(ctx, bot); err != nil {
					_app.Log.Warn("availability alert check failed", zap.Error(err))
				}
			}
		}
	}
}

func checkAvailabilityAlerts(ctx context.Context, bot *gotgbot.Bot) error {
	cycle, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()
	alerts, err := _app.DB.AvailabilityAlerts(cycle, 0, 10)
	if err != nil {
		return err
	}
	for _, a := range alerts {
		if cycle.Err() != nil || !availabilityHours(time.Now()) {
			break
		}
		if err := _app.DB.DeferAvailabilityAlert(cycle, a, time.Now().Add(5*time.Minute)); err != nil {
			return err
		}
		if err := deliverAvailabilityAlert(cycle, bot, a); err != nil {
			_app.Log.Warn("availability alert delivery failed", zap.Error(err))
		}
	}
	return nil
}

func deliverAvailabilityAlert(ctx context.Context, bot *gotgbot.Bot, a mongo.AvailabilityAlert) error {
	blocked, err := _app.DB.IsUserBlocked(a.UserID)
	if err != nil || blocked {
		return err
	}
	files, err := searchCatalogContext(ctx, mongo.CatalogFilter{Query: a.Title, Year: a.Year, Language: a.Audio, MediaOnly: true}, 1)
	if err != nil {
		return err
	}
	if len(files) == 0 || !availabilityHours(time.Now()) {
		return nil
	}

	claimed, err := _app.DB.ClaimAvailabilityAlert(ctx, a)
	if err != nil || !claimed {
		return err
	}
	buttons := files.Process(a.UserID, bot.Username, _app.Config)
	_, err = bot.SendMessage(a.UserID, fmt.Sprintf("Available now: %s", availabilityLabel(a)), &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
	var telegramErr *gotgbot.TelegramError
	var retry time.Duration
	if errors.As(err, &telegramErr) && telegramErr.Code == 429 {
		retry = time.Minute
		if telegramErr.ResponseParams != nil {
			retry = max(retry, time.Duration(telegramErr.ResponseParams.RetryAfter)*time.Second)
		}
	}
	finish, cancel := context.WithTimeout(_app.Ctx, 5*time.Second)
	defer cancel()
	return errors.Join(err, _app.DB.CompleteAvailabilityAlert(finish, a, retry, err == nil))
}
