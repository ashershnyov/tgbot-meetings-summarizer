package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/client/giga"
	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/client/salute"
	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/model"
	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/storage"
	tele "gopkg.in/telebot.v3"
)

const (
	timeFormat  = "02.01 15:04"
	gigaTimeout = 20 * time.Second
)

// Handler defines bot's handler.
type Handler struct {
	storage      storage.DB
	GigaChat     *giga.Client
	SaluteSpeech *salute.Client
}

// New returns a new handler.
func New(storage storage.DB, giga *giga.Client, salute *salute.Client) Handler {
	return Handler{
		storage:      storage,
		GigaChat:     giga,
		SaluteSpeech: salute,
	}
}

// Register registers handlers for the bot.
func (h *Handler) Register(bot *tele.Bot) {
	bot.Handle("/start", h.handleStart)
	bot.Handle("/list", h.handleList)
	bot.Handle("/get", h.handleGet)
	bot.Handle("/find", h.handleFind)
	bot.Handle("/chat", h.handleChat)
	bot.Handle(tele.OnVoice, h.handleVoice)
	bot.Handle(tele.OnAudio, h.handleAudio)
}

func (h *Handler) handleStart(c tele.Context) error {
	if err := h.storage.AddUser(context.Background(), c.Sender().ID); err != nil {
		return c.Send("Ошибка регистрации пользователя. Попробуйте снова.")
	}
	return c.Send("Пользователь зарегистрирован. Отправьте голосовое сообщение или аудиофайл для обработки.")
}

func (h *Handler) handleList(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	meetings, err := h.storage.ListMeetings(ctx, c.Sender().ID)
	if err != nil {
		return c.Send("Ошибка получения списка встреч.")
	}
	if len(meetings) == 0 {
		return c.Send("У вас пока нет сохранённых встреч.")
	}

	msg := "Ваши последние встречи:\n"
	for i, m := range meetings {
		sum := ""
		if m.Summary == nil {
			sum = "В обработке..."
		} else {
			sum = *m.Summary
		}
		msg += fmt.Sprintf("%d. [%s] ID: `%s`\n   %s\n\n", i+1, m.CreatedAt.Format(timeFormat), m.ID, sum)
	}
	return c.Send(msg)
}

func (h *Handler) handleGet(c tele.Context) error {
	args := strings.SplitN(c.Text(), " ", 2)
	if len(args) < 2 {
		return c.Send("Используйте: `/get <ID встречи>`")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m, err := h.storage.GetMeeting(ctx, c.Sender().ID, args[1])
	if err != nil {
		return c.Send("Встреча не найдена или у вас нет доступа ко встрече с таким ID.")
	}
	summary := "В обработке..."
	if m.Summary != nil {
		summary = *m.Summary
	}
	msg := fmt.Sprintf("Транскрипция встречи `%s`:\n%s\n\nКратко: %s", m.ID, m.Transcript, summary)
	return c.Send(msg)
}

func (h *Handler) handleFind(c tele.Context) error {
	args := strings.SplitN(c.Text(), " ", 2)
	if len(args) < 2 {
		return c.Send("Используйте: `/find <ключевые слова>`")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	meetings, err := h.storage.FindMeetings(ctx, c.Sender().ID, args[1])
	if err != nil {
		return c.Send("Ошибка поиска.")
	}
	if len(meetings) == 0 {
		return c.Send("Ничего не найдено по вашему запросу.")
	}

	msg := fmt.Sprintf("Найдено %d встреч:\n", len(meetings))
	for i, m := range meetings {
		sum := ""
		if m.Summary == nil {
			sum = "В обработке..."
		} else {
			sum = *m.Summary
		}
		msg += fmt.Sprintf("%d. [%s] ID: `%s`\n   %s\n", i+1, m.CreatedAt.Format(timeFormat), m.ID, sum)
	}
	return c.Send(msg + "\nДля полного текста: `/get <ID>`")
}

func (h *Handler) handleChat(c tele.Context) error {
	args := strings.SplitN(c.Text(), " ", 2)
	if len(args) < 2 {
		return c.Send("Используйте: `/chat <ваш вопрос>`")
	}

	respChan, err := h.GigaChat.SubmitChatJob(args[1])
	if err != nil && err == giga.ErrQueueOverflow {
		return c.Send("ИИ-ассистент не может принять ваш запрос. Повторите запрос позднее.")
	}
	if err != nil {
		return c.Send("Произошла ошибка при отправке запроса ИИ-ассистенту.")
	}

	c.Send("Запрос отправлен ИИ-ассистенту...")

	select {
	case reply := <-respChan:
		return c.Send(reply.Message)
	case <-time.After(gigaTimeout):
		return c.Send("Превышено время ожидания ответа ИИ-ассистента.")
	}
}

func (h *Handler) handleAudio(c tele.Context) error {
	return h.processAudio(c, "audio", c.Message().Audio.FileID, c.Message().Audio.MIME)
}
func (h *Handler) handleVoice(c tele.Context) error {
	return h.processAudio(c, "voice", c.Message().Voice.FileID, c.Message().Voice.MIME)
}

func (h *Handler) processAudio(c tele.Context, fileType, fileID, mime string) error {
	c.Send("Файл принят. Начинаю обработку...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		file, err := c.Bot().FileByID(fileID)
		if err != nil {
			c.Send("Ошибка получения файла из Telegram.")
			return
		}
		fReader, err := c.Bot().File(&file)

		transcript, err := h.SaluteSpeech.Transcribe(fReader, "test_transcribe", mime)
		if err != nil {
			c.Send(fmt.Sprintf("Ошибка транскрибации: %v", err))
			return
		}

		m := &model.Meeting{
			UserID:     c.Sender().ID,
			FileType:   fileType,
			Transcript: transcript,
		}
		meetingID, err := h.storage.AddMeeting(ctx, m)
		if err != nil {
			c.Send("Ошибка сохранения в базу данных.")
			return
		}

		respChan, err := h.GigaChat.SubmitSummaryJob(transcript)
		if err != nil && err == giga.ErrQueueOverflow {
			c.Send("ИИ-ассистент не может принять ваш запрос. Повторите запрос позднее.")
			return
		}
		if err != nil {
			c.Send("Произошла ошибка при отправке запроса ИИ-ассистенту.")
			return
		}

		select {
		case res := <-respChan:
			if err := h.storage.UpdateSummary(ctx, meetingID, res.Message); err != nil {
				c.Send("Ошибка сохранения выжимки.")
				return
			}
			c.Send(fmt.Sprintf("Встреча обработана!\nID: `%s`\nКратко: %s\nПоиск: `/find <ключевое слово>`", meetingID, res.Message))
		case <-time.After(gigaTimeout):
			c.Send("Транскрипция сохранена, но ИИ-выжимка задерживается.")
		}
	}()

	return nil
}
