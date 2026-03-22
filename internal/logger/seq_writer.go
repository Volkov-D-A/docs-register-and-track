package logger

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SeqAsyncWriter реализует асинхронную отправку логов в Seq через io.Writer.
type SeqAsyncWriter struct {
	url    string
	client *http.Client
	ch     chan []byte
	done   chan struct{}
	once   sync.Once
}

// NewSeqAsyncWriter создает новый Writer для Seq.
func NewSeqAsyncWriter(url string) *SeqAsyncWriter {
	w := &SeqAsyncWriter{
		url:    fmt.Sprintf("%s/api/events/raw", url),
		client: &http.Client{Timeout: 5 * time.Second},
		ch:     make(chan []byte, 1000), // Буфер на 1000 сообщений, чтобы не блокировать потоки
		done:   make(chan struct{}),
	}
	go w.start()
	return w
}

// Write добавляет лог в буфер. Если буфер полон, сообщение отбрасывается
// (или можно добавить небольшую блокировку по желанию, но для UI приложения лучше отбрасывать).
func (w *SeqAsyncWriter) Write(p []byte) (n int, err error) {
	// Копируем срез, так как p может быть переиспользован slog-ом
	msg := make([]byte, len(p))
	copy(msg, p)

	select {
	case w.ch <- msg:
		return len(p), nil
	default:
		// Если канал переполнен - логи дропаются, чтобы не блокировать UI Wails.
		return len(p), nil
	}
}

// start обрабатывает фоновую отправку логов.
func (w *SeqAsyncWriter) start() {
	for {
		select {
		case msg := <-w.ch:
			w.send(msg)
		case <-w.done:
			// Отправляем оставшиеся в буфере логи при завершении работы.
			for len(w.ch) > 0 {
				w.send(<-w.ch)
			}
			return
		}
	}
}

// send выполняет HTTP POST запрос к Seq.
func (w *SeqAsyncWriter) send(msg []byte) {
	req, err := http.NewRequest(http.MethodPost, w.url, bytes.NewReader(msg))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/vnd.serilog.clef")

	resp, err := w.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// Close корректно завершает работу асинхронного Writer, ожидая отправки всех логов в буфере.
func (w *SeqAsyncWriter) Close() error {
	w.once.Do(func() {
		close(w.done)
		// Дожидаемся небольшого таймаута на всякий случай или просто возвращаемся.
		// Закрытие канала дает сигнал горутине завершиться. 
		time.Sleep(100 * time.Millisecond) // Краткая пауза, чтобы start() успел доотправить остатки.
	})
	return nil
}
