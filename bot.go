package tlbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type ParseMode string

// Parse modes
const (
	ModeNone     ParseMode = ""
	ModeMarkdown ParseMode = "markdown"
)

// Bot represent a Telegram bot.
type Bot struct {
	token   string
	baseURL string
	client  *http.Client
}

// New creates a new Telegram bot with the given token, which is given by
// Botfather. See https://core.telegram.org/bots#botfather
func New(token string) Bot {
	return Bot{
		token:   token,
		baseURL: fmt.Sprintf("https://api.telegram.org/bot%v/", token),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Listen listens on the given address addr and returns a read-only Message
// channel.
func (b Bot) Listen(addr string) <-chan Message {
	messageCh := make(chan Message)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		defer w.WriteHeader(http.StatusOK)

		var u Update
		err := json.NewDecoder(req.Body).Decode(&u)
		if err != nil {
			log.Printf("error decoding request body: %v\n", err)
			return

		}
		messageCh <- u.Payload
	})

	go func() {
		// ListenAndServe always returns non-nil error
		err := http.ListenAndServe(addr, mux)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}()

	return messageCh
}

// SetWebhook assigns bot's webhook url with the given url.
func (b Bot) SetWebhook(webhook string) error {
	params := url.Values{}
	params.Set("url", webhook)

	var r struct {
		OK      bool   `json:"ok"`
		Desc    string `json:"description"`
		ErrCode int    `json:"errorcode"`
	}
	err := b.sendCommand("setWebhook", params, &r)
	if err != nil {
		return err
	}

	if !r.OK {
		return fmt.Errorf("%v (%v)", r.Desc, r.ErrCode)
	}

	return nil
}

// SendMessage sends text message to the recipient. Callers can send plain
// text or markdown messages by setting mode parameter.
func (b Bot) SendMessage(recipient int, message string, opts *SendOptions) (Message, error) {
	params := url.Values{
		"chat_id": {strconv.Itoa(recipient)},
		"text":    {message},
	}

	mapSendOptions(&params, opts)

	var r struct {
		OK      bool   `json:"ok"`
		Desc    string `json:"description"`
		ErrCode int    `json:"errorcode"`
		Message Message
	}
	b.sendCommand("sendMessage", params, &r)

	if !r.OK {
		return Message{}, fmt.Errorf("%v (%v)", r.Desc, r.ErrCode)
	}
	return r.Message, nil
}

func (b Bot) forwardMessage(recipient User, message Message) (Message, error) {
	panic("not implemented yet")
}

// SendPhoto sends given photo to recipient. Only remote URLs are supported for now.
// A trivial example is:
//
//  b := bot.New("your-token-here")
//  photo := bot.Photo{URL: "http://i.imgur.com/6S9naG6.png"}
//  err := b.SendPhoto(recipient, photo, "sample image", nil)
func (b Bot) SendPhoto(recipient int, photo Photo, opts *SendOptions) (Message, error) {
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(recipient))
	params.Set("caption", photo.Caption)

	mapSendOptions(&params, opts)
	var r struct {
		OK      bool    `json:"ok"`
		Desc    string  `json:"description"`
		ErrCode int     `json:"error_code"`
		Message Message `json:"message"`
	}

	var err error
	if photo.Exists() {
		params.Set("photo", photo.FileID)
		err = b.sendCommand("sendPhoto", params, &r)
	} else if photo.URL != "" {
		params.Set("photo", photo.URL)
		err = b.sendCommand("sendPhoto", params, &r)
	} else {
		err = b.sendFile("sendPhoto", photo.File, "photo", params, &r)
	}

	if err != nil {
		return Message{}, err
	}

	if !r.OK {
		return Message{}, fmt.Errorf("%v (%v)", r.Desc, r.ErrCode)
	}

	return r.Message, nil
}

func (b Bot) sendFile(method string, f File, form string, params url.Values, v interface{}) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(form, f.Name)
	if err != nil {
		return err
	}

	_, err = io.Copy(part, f.Body)
	if err != nil {
		return err
	}

	for k, v := range params {
		w.WriteField(k, v[0])
	}

	err = w.Close()
	if err != nil {
		return err
	}

	resp, err := b.client.Post(b.baseURL+method, w.FormDataContentType(), &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(&v)
}

// SendAudio sends audio files, if you want Telegram clients to display
// them in the music player. audio must be in the .mp3 format and must not
// exceed 50 MB in size.
func (b Bot) sendAudio(recipient User, audio Audio, opts *SendOptions) (Message, error) {
	panic("not implemented yet")
}

// SendDocument sends general files. Documents must not exceed 50 MB in size.
func (b Bot) sendDocument(recipient User, document Document, opts *SendOptions) (Message, error) {
	panic("not implemented yet")
}

//SendSticker sends stickers with .webp extensions.
func (b Bot) sendSticker(recipient User, sticker Sticker, opts *SendOptions) (Message, error) {
	panic("not implemented yet")
}

// SendVideo sends video files. Telegram clients support mp4 videos (other
// formats may be sent as Document). Video files must not exceed 50 MB in size.
func (b Bot) sendVideo(recipient User, video Video, opts *SendOptions) (Message, error) {
	panic("not implemented yet")
}

// SendVoice sends audio files, if you want Telegram clients to display
// the file as a playable voice message. For this to work, your audio must be
// in an .ogg file encoded with OPUS (other formats may be sent as Audio or
// Document). audio must not exceed 50 MB in size.
func (b Bot) sendVoice(recipient User, audio Audio, opts *SendOptions) (Message, error) {
	panic("not implemented yet")
}

// SendLocation sends location point on the map.
func (b Bot) SendLocation(recipient int, location Location, opts *SendOptions) (Message, error) {
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(recipient))
	params.Set("latitude", strconv.FormatFloat(location.Lat, 'f', -1, 64))
	params.Set("longitude", strconv.FormatFloat(location.Long, 'f', -1, 64))

	mapSendOptions(&params, opts)

	var r struct {
		OK      bool    `json:"ok"`
		Desc    string  `json:"description"`
		ErrCode int     `json:"errorcode"`
		Message Message `json:"message"`
	}
	err := b.sendCommand("sendLocation", params, &r)
	if err != nil {
		return Message{}, err
	}

	if !r.OK {
		return Message{}, fmt.Errorf("%v (%v)", r.Desc, r.ErrCode)
	}

	return r.Message, nil
}

// SendVenue sends information about a venue.
func (b Bot) SendVenue(recipient int, venue Venue, opts *SendOptions) (Message, error) {
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(recipient))
	params.Set("latitude", strconv.FormatFloat(venue.Location.Lat, 'f', -1, 64))
	params.Set("longitude", strconv.FormatFloat(venue.Location.Long, 'f', -1, 64))
	params.Set("title", venue.Title)
	params.Set("address", venue.Address)

	mapSendOptions(&params, opts)

	var r struct {
		OK      bool    `json:"ok"`
		Desc    string  `json:"description"`
		ErrCode int     `json:"errorcode"`
		Message Message `json:"message"`
	}
	err := b.sendCommand("sendVenue", params, &r)
	if err != nil {
		return Message{}, err
	}

	if !r.OK {
		return Message{}, fmt.Errorf("%v (%v)", r.Desc, r.ErrCode)
	}
	return r.Message, nil
}

// SendChatAction broadcasts type of action to recipient, such as `typing`,
// `uploading a photo` etc.
func (b Bot) SendChatAction(recipient int, action Action) error {
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(recipient))
	params.Set("action", string(action))

	var r struct {
		OK      bool   `json:"ok"`
		Desc    string `json:"description"`
		ErrCode int    `json:"error_code"`
	}

	err := b.sendCommand("sendChatAction", params, &r)
	if err != nil {
		return err

	}
	if !r.OK {
		return fmt.Errorf("%v (%v)", r.Desc, r.ErrCode)
	}

	return nil
}

type SendOptions struct {
	ReplyTo int

	ParseMode ParseMode

	DisableWebPagePreview bool

	DisableNotification bool

	ReplyMarkup ReplyMarkup
}

func (b Bot) GetFile(fileID string) (File, error) {
	params := url.Values{}
	params.Set("file_id", fileID)

	var r struct {
		OK      bool   `json:"ok"`
		Desc    string `json:"description"`
		ErrCode int    `json:"errorcode"`
		File    File   `json:"result"`
	}
	err := b.sendCommand("getFile", params, &r)
	if err != nil {
		return File{}, err
	}

	if !r.OK {
		return File{}, fmt.Errorf("%v (%v)", r.Desc, r.ErrCode)
	}

	return r.File, nil
}

func (b Bot) GetFileDownloadURL(fileID string) (string, error) {
	f, err := b.GetFile(fileID)
	if err != nil {
		return "", err
	}

	u := "https://api.telegram.org/file/bot" + b.token + "/" + f.FilePath
	return u, nil
}

func (b Bot) sendCommand(method string, params url.Values, v interface{}) error {
	resp, err := b.client.PostForm(b.baseURL+method, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(&v)
}

// Me return bot info
func Me(b Bot) (User, error) {
	return b.getMe()
}
func (b Bot) getMe() (User, error) {
	var r struct {
		OK      bool   `json:"ok"`
		Desc    string `json:"description"`
		ErrCode int    `json:"error_code"`

		User User `json:"result"`
	}
	err := b.sendCommand("getMe", url.Values{}, &r)
	if err != nil {
		return User{}, err
	}

	if !r.OK {
		return User{}, fmt.Errorf("%v (%v)", r.Desc, r.ErrCode)
	}

	return r.User, nil
}

func mapSendOptions(m *url.Values, opts *SendOptions) {
	if opts == nil {
		return
	}

	if opts.ReplyTo != 0 {
		m.Set("reply_to_message_id", strconv.Itoa(opts.ReplyTo))
	}

	if opts.DisableWebPagePreview {
		m.Set("disable_web_page_preview", "true")
	}

	if opts.DisableNotification {
		m.Set("disable_notification", "true")
	}

	if opts.ParseMode != ModeNone {
		m.Set("parse_mode", string(opts.ParseMode))
	}

	// TODO: map ReplyMarkup options as well
}
