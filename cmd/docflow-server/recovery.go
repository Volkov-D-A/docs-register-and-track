package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup"
	"github.com/Volkov-D-A/docs-register-and-track/internal/backup/smb"
	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
)

type recoveryRequest struct {
	SMB          smb.Config `json:"smb"`
	Password     string     `json:"password"`
	ID           string     `json:"id"`
	Confirmation string     `json:"confirmation"`
}

func runRecovery(args []string, out io.Writer) error {
	cfg, err := config.LoadServer()
	if err != nil {
		return err
	}
	schema, err := database.LatestSchemaVersion()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(args) == 2 && args[0] == "restore" {
		if err = backup.Restore(ctx, backup.PostgreSQL{Config: cfg.Database}, cfg.S3, args[1], cfg.Backup.Directory, cfg.Backup.MaxBytes, int(schema)); err != nil {
			return err
		}
		fmt.Fprintln(out, "restore completed; sessions revoked, schedule disabled")
		return nil
	}
	if len(args) != 1 || args[0] != "recovery" {
		return fmt.Errorf("use recovery or restore ARCHIVE")
	}
	raw, err := os.ReadFile(os.Getenv("DOCFLOW_RECOVERY_SECRET_FILE"))
	if err != nil {
		return fmt.Errorf("read recovery secret: %w", err)
	}
	secret := strings.TrimSpace(string(raw))
	if len(secret) < 32 {
		return fmt.Errorf("recovery secret must contain at least 32 characters")
	}
	expected := sha256.Sum256([]byte(secret))
	if cfg.Backup.Directory == "" {
		return fmt.Errorf("DOCFLOW_BACKUP_DIRECTORY is required")
	}
	if err = os.MkdirAll(cfg.Backup.Directory, 0700); err != nil {
		return err
	}
	var mu sync.Mutex
	state := "Ожидание"
	running := false
	var workers sync.WaitGroup
	defer workers.Wait()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		io.WriteString(w, recoveryHTML)
	})
	authorized := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			supplied := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			actual := sha256.Sum256([]byte(supplied))
			if subtle.ConstantTimeCompare(actual[:], expected[:]) != 1 {
				http.Error(w, "Неверный аварийный ключ", 401)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
			next(w, r)
		}
	}
	mux.HandleFunc("GET /status", authorized(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"state": state, "running": running})
	}))
	mux.HandleFunc("POST /copies", authorized(func(w http.ResponseWriter, r *http.Request) {
		var req recoveryRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			http.Error(w, "Неверный запрос", 400)
			return
		}
		limited, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		client, err := smb.Open(limited, req.SMB, req.Password)
		if err != nil {
			http.Error(w, "Не удалось подключиться к SMB", 400)
			return
		}
		defer client.Close()
		copies, err := backup.RemoteCopies(limited, client)
		if err != nil {
			http.Error(w, "Не удалось прочитать каталог копий", 400)
			return
		}
		json.NewEncoder(w).Encode(copies)
	}))
	mux.HandleFunc("POST /restore", authorized(func(w http.ResponseWriter, r *http.Request) {
		var req recoveryRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.Confirmation != "ВОССТАНОВИТЬ" {
			http.Error(w, "Введите ВОССТАНОВИТЬ для подтверждения", 400)
			return
		}
		mu.Lock()
		if running {
			mu.Unlock()
			http.Error(w, "Восстановление уже выполняется", 409)
			return
		}
		running = true
		state = "Загрузка и проверка копии"
		mu.Unlock()
		workers.Add(1)
		go func() {
			defer workers.Done()
			limited, cancel := context.WithTimeout(ctx, 6*time.Hour)
			defer cancel()
			err := func() error {
				work, err := os.MkdirTemp(cfg.Backup.Directory, "download-")
				if err != nil {
					return err
				}
				defer os.RemoveAll(work)
				client, err := smb.Open(limited, req.SMB, req.Password)
				if err != nil {
					return err
				}
				defer client.Close()
				archive, err := backup.DownloadCopy(limited, client, req.ID, work, cfg.Backup.MaxBytes/2)
				if err != nil {
					return err
				}
				mu.Lock()
				state = "Восстановление базы и файлов"
				mu.Unlock()
				return backup.Restore(limited, backup.PostgreSQL{Config: cfg.Database}, cfg.S3, archive, cfg.Backup.Directory, cfg.Backup.MaxBytes/2, int(schema))
			}()
			mu.Lock()
			defer mu.Unlock()
			running = false
			if err != nil {
				state = "Ошибка восстановления: " + err.Error()
				os.WriteFile(filepath.Join(cfg.Backup.Directory, fmt.Sprintf("restore-error-%d.log", time.Now().UnixNano())), []byte(state), 0600)
			} else {
				state = "Восстановление завершено. Сессии отозваны, расписание отключено. Можно остановить recovery и запустить приложение."
			}
		}()
		w.WriteHeader(202)
	}))
	address := os.Getenv("DOCFLOW_RECOVERY_LISTEN_ADDRESS")
	if address == "" {
		address = "127.0.0.1:8081"
	}
	server := &http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 35 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: time.Minute}
	ended := make(chan error, 1)
	go func() { ended <- server.ListenAndServe() }()
	fmt.Fprintln(out, "recovery panel listening on", address)
	select {
	case <-ctx.Done():
		limited, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(limited)
	case err := <-ended:
		stop()
		return err
	}
}

const recoveryHTML = `<!doctype html><html lang="ru"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Восстановление Docflow</title>
<style>body{font:16px system-ui;max-width:760px;margin:40px auto;padding:16px}label{display:block;margin:12px 0}input,select,button{font:inherit;padding:8px;max-width:100%}input{display:block;width:95%}button{margin:12px 8px 12px 0}#status{white-space:pre-wrap}</style>
<h1>Восстановление Docflow</h1><p>Остановите обычный сервер. Целевая база и bucket должны быть пустыми. Восстановление заменяет состояние приложения состоянием из копии; после проверки все сессии будут отозваны, расписание отключено.</p>
<label>Аварийный ключ<input id="token" type="password" autocomplete="off"></label>
<label>Сервер SMB<input id="host"></label><label>Общая папка<input id="share"></label><label>Подкаталог<input id="directory"></label><label>Пользователь<input id="user"></label><label>Домен<input id="domain"></label><label>Пароль SMB<input id="password" type="password" autocomplete="off"></label>
<button id="list">Найти копии</button><label>Резервная копия<select id="copies"></select></label><label>Подтверждение: ВОССТАНОВИТЬ<input id="confirmation"></label><button id="restore">Восстановить</button><button id="refresh">Проверить состояние</button><p id="status" role="status"></p>
<script>
const el=id=>document.getElementById(id);const request=()=>({smb:Object.fromEntries(['host','share','directory','user','domain'].map(k=>[k,el(k).value])),password:el('password').value,id:el('copies').value,confirmation:el('confirmation').value});
async function api(path,body){const response=await fetch(path,{method:body?'POST':'GET',headers:{Authorization:'Bearer '+el('token').value,'Content-Type':'application/json'},body:body?JSON.stringify(body):undefined});if(!response.ok)throw new Error(await response.text());return response.status===202?null:response.json()}
async function action(fn){try{await fn()}catch(error){el('status').textContent=error.message}}
el('list').onclick=()=>action(async()=>{const copies=await api('/copies',request());el('copies').replaceChildren(...copies.map(c=>{const option=document.createElement('option');option.value=c.id;option.textContent=c.createdAt+' — '+(c.size/1048576).toFixed(1)+' МБ';return option}));el('status').textContent='Найдено копий: '+copies.length});
el('restore').onclick=()=>action(async()=>{await api('/restore',request());el('password').value='';el('status').textContent='Восстановление запущено'});
el('refresh').onclick=()=>action(async()=>{el('status').textContent=(await api('/status')).state});
</script></html>`
