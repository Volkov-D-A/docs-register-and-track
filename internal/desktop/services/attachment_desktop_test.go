package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/operations"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestAttachmentService_ValidatePathInDownloads(t *testing.T) {
	// Проверка пути доступа к файлу для защиты от уязвимости Path Traversal
	t.Run("valid path", func(t *testing.T) {
		dir := t.TempDir()
		svc := &AttachmentService{downloadDir: func() (string, error) { return dir, nil }}
		downloadDir, _ := svc.getDownloadDir()
		_, err := svc.validatePathInDownloads(downloadDir + "/test.pdf")
		require.NoError(t, err)
	})

	t.Run("path traversal attack", func(t *testing.T) {
		dir := t.TempDir()
		svc := &AttachmentService{downloadDir: func() (string, error) { return dir, nil }}
		_, err := svc.validatePathInDownloads("C:\\Windows\\System32\\..\\..\\test.pdf")
		require.Error(t, err)
		requireAppError(t, err, "FORBIDDEN", 403, "папке загрузок")
	})

	t.Run("outside downloads", func(t *testing.T) {
		dir := t.TempDir()
		svc := &AttachmentService{downloadDir: func() (string, error) { return dir, nil }}
		_, err := svc.validatePathInDownloads("C:\\Windows\\System32\\cmd.exe")
		require.Error(t, err)
		requireAppError(t, err, "FORBIDDEN", 403, "папке загрузок")
	})
}

// Embedding the interface makes an unexpected HTTP operation fail the test.
type desktopAttachmentClient struct {
	serverclient.AttachmentClient
	upload   func(context.Context, string, string, string, int64, io.Reader) (*dto.Attachment, error)
	download func(context.Context, string) (*dto.Attachment, io.ReadCloser, error)
	call     func(context.Context, string) error
}

func (c *desktopAttachmentClient) UploadAttachment(ctx context.Context, doc, assignment, name string, size int64, body io.Reader) (*dto.Attachment, error) {
	return c.upload(ctx, doc, assignment, name, size, body)
}
func (c *desktopAttachmentClient) GetAttachmentContent(ctx context.Context, id string) (*dto.Attachment, io.ReadCloser, error) {
	return c.download(ctx, id)
}
func (c *desktopAttachmentClient) ListDocumentAttachments(ctx context.Context, id string) ([]dto.Attachment, error) {
	return []dto.Attachment{{ID: id}}, c.call(ctx, "list:"+id)
}
func (c *desktopAttachmentClient) ListAssignmentAttachments(ctx context.Context, id string) ([]dto.Attachment, error) {
	return []dto.Attachment{{AssignmentID: id}}, c.call(ctx, "assignment:"+id)
}
func (c *desktopAttachmentClient) DeleteAttachment(ctx context.Context, id string) error {
	return c.call(ctx, "delete:"+id)
}
func (c *desktopAttachmentClient) BulkDeleteAttachments(ctx context.Context, date string) (int, error) {
	return 3, c.call(ctx, "bulk:"+date)
}
func (c *desktopAttachmentClient) ReconcileAttachmentStorage(ctx context.Context) (*models.AttachmentStorageReconciliation, error) {
	return &models.AttachmentStorageReconciliation{MissingObjects: []string{"missing"}}, c.call(ctx, "reconcile")
}

func TestDesktopAttachmentContract(t *testing.T) {
	typ := reflect.TypeOf((*AttachmentService)(nil))
	var names []string
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		names = append(names, method.Name)
		for arg := 1; arg < method.Type.NumIn(); arg++ {
			require.Equal(t, reflect.TypeOf(""), method.Type.In(arg), "%s must only accept UI strings", method.Name)
		}
	}
	require.ElementsMatch(t, []string{"Upload", "UploadForAssignment", "GetList", "GetAssignmentFiles", "Delete", "BulkDeleteOlderThan", "ReconcileStorage", "DownloadToDisk", "OpenFile", "OpenFolder"}, names)
	for _, ext := range []string{"js", "d.ts"} {
		data, err := os.ReadFile("../../../frontend/wailsjs/go/services/AttachmentService." + ext)
		require.NoError(t, err)
		matches := regexp.MustCompile(`export function (\w+)\(`).FindAllStringSubmatch(string(data), -1)
		var generated []string
		for _, match := range matches {
			generated = append(generated, match[1])
		}
		require.ElementsMatch(t, names, generated)
	}
}

func TestDesktopAttachmentUpload(t *testing.T) {
	for _, assignment := range []bool{false, true} {
		t.Run(fmt.Sprint(assignment), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "report.txt")
			require.NoError(t, os.WriteFile(path, []byte("contents"), 0600))
			id := uuid.NewString()
			var uploaded *os.File
			client := &desktopAttachmentClient{upload: func(ctx context.Context, doc, assn, name string, size int64, body io.Reader) (*dto.Attachment, error) {
				if assignment {
					require.Equal(t, id, assn)
					require.Empty(t, doc)
				} else {
					require.Equal(t, id, doc)
					require.Empty(t, assn)
				}
				require.Equal(t, "report.txt", name)
				require.Equal(t, int64(8), size)
				uploaded = body.(*os.File)
				data, err := io.ReadAll(body)
				require.NoError(t, err)
				require.Equal(t, "contents", string(data))
				return &dto.Attachment{Filename: name}, nil
			}}
			svc, startup, err := NewDesktopAttachmentService(client, DesktopAttachmentOptions{OpenFilesDialog: func(context.Context, wailsruntime.OpenDialogOptions) ([]string, error) { return []string{path}, nil }})
			require.NoError(t, err)
			_, err = svc.Upload(id)
			require.ErrorContains(t, err, "not initialized")
			startup(context.Background())
			var items *dto.AttachmentUploadResult
			if assignment {
				items, err = svc.UploadForAssignment(id)
			} else {
				items, err = svc.Upload(id)
			}
			require.NoError(t, err)
			require.Len(t, items.Items, 1)
			require.Equal(t, "report.txt", items.Items[0].Attachment.Filename)
			_, err = uploaded.Read(make([]byte, 1))
			require.ErrorIs(t, err, os.ErrClosed)
		})
	}
}

func TestDesktopAttachmentPickerFailures(t *testing.T) {
	for _, tc := range []struct {
		name  string
		paths []string
		err   error
		want  string
	}{
		{name: "cancel"}, {name: "dialog error", err: assert.AnError, want: "failed to choose"},
		{name: "missing file", paths: []string{filepath.Join(t.TempDir(), "missing")}, want: "открыть"},
		{name: "directory", paths: []string{t.TempDir()}, want: "обычным файлом"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, start, err := NewDesktopAttachmentService(&desktopAttachmentClient{}, DesktopAttachmentOptions{OpenFilesDialog: func(context.Context, wailsruntime.OpenDialogOptions) ([]string, error) { return tc.paths, tc.err }})
			require.NoError(t, err)
			start(context.Background())
			for _, upload := range []func(string) (*dto.AttachmentUploadResult, error){svc.Upload, svc.UploadForAssignment} {
				items, err := upload(uuid.NewString())
				if tc.want == "" {
					require.NoError(t, err)
					require.Empty(t, items.Items)
				} else if tc.err != nil {
					require.ErrorContains(t, err, tc.want)
				} else {
					require.NoError(t, err)
					require.Len(t, items.Items, 1)
					require.Nil(t, items.Items[0].Attachment)
					require.Contains(t, items.Items[0].Error.Message, tc.want)
				}
			}
		})
	}
}

type trackedAttachmentBody struct {
	io.Reader
	closed bool
}

func (b *trackedAttachmentBody) Close() error { b.closed = true; return nil }

type brokenAttachmentReader struct{}

func (brokenAttachmentReader) Read([]byte) (int, error) { return 0, assert.AnError }

func TestDesktopAttachmentDownload(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "Downloads")
			body := &trackedAttachmentBody{Reader: strings.NewReader("contents")}
			if fail {
				body.Reader = io.MultiReader(strings.NewReader("partial"), brokenAttachmentReader{})
			}
			client := &desktopAttachmentClient{download: func(ctx context.Context, id string) (*dto.Attachment, io.ReadCloser, error) {
				require.Equal(t, "id", id)
				return &dto.Attachment{Filename: `../folder\report.txt`}, body, nil
			}}
			svc, _, err := NewDesktopAttachmentService(client, DesktopAttachmentOptions{DownloadDir: func() (string, error) { return dir, nil }})
			require.NoError(t, err)
			path, err := svc.DownloadToDisk("id")
			require.True(t, body.closed)
			if fail {
				require.ErrorIs(t, err, assert.AnError)
				entries, err := os.ReadDir(dir)
				require.NoError(t, err)
				require.Empty(t, entries)
				return
			}
			require.NoError(t, err)
			require.Equal(t, filepath.Join(dir, "report.txt"), path)
			body.Reader = strings.NewReader("second")
			second, err := svc.DownloadToDisk("id")
			require.NoError(t, err)
			require.Equal(t, filepath.Join(dir, "report (1).txt"), second)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "contents", string(data))
		})
	}
}

func TestDesktopAttachmentHTTPForwardingAndLifecycle(t *testing.T) {
	for _, serverErr := range []error{nil, assert.AnError} {
		t.Run(fmt.Sprint(serverErr), func(t *testing.T) {
			lifecycle := operations.NewLifecycle(time.Second)
			defer lifecycle.Shutdown(context.Background())
			var calls []string
			var contexts []context.Context
			client := &desktopAttachmentClient{call: func(ctx context.Context, op string) error {
				calls = append(calls, op)
				contexts = append(contexts, ctx)
				_, ok := ctx.Deadline()
				require.True(t, ok)
				return serverErr
			}}
			svc, _, err := NewDesktopAttachmentService(client, DesktopAttachmentOptions{Lifecycle: lifecycle})
			require.NoError(t, err)
			items, err := svc.GetList("doc")
			require.ErrorIs(t, err, serverErr)
			require.Equal(t, "doc", items[0].ID)
			items, err = svc.GetAssignmentFiles("assn")
			require.ErrorIs(t, err, serverErr)
			require.Equal(t, "assn", items[0].AssignmentID)
			require.ErrorIs(t, svc.Delete("file"), serverErr)
			count, err := svc.BulkDeleteOlderThan("date")
			require.ErrorIs(t, err, serverErr)
			require.Equal(t, 3, count)
			result, err := svc.ReconcileStorage()
			require.ErrorIs(t, err, serverErr)
			require.Equal(t, []string{"missing"}, result.MissingObjects)
			require.Equal(t, []string{"list:doc", "assignment:assn", "delete:file", "bulk:date", "reconcile"}, calls)
			for _, ctx := range contexts {
				require.ErrorIs(t, ctx.Err(), context.Canceled)
			}
		})
	}
	for _, shutdown := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel-%v", shutdown), func(t *testing.T) {
			lifecycle := operations.NewLifecycle(10 * time.Millisecond)
			defer lifecycle.Shutdown(context.Background())
			entered := make(chan struct{})
			client := &desktopAttachmentClient{call: func(ctx context.Context, _ string) error { close(entered); <-ctx.Done(); return ctx.Err() }}
			svc, _, err := NewDesktopAttachmentService(client, DesktopAttachmentOptions{Lifecycle: lifecycle})
			require.NoError(t, err)
			done := make(chan error, 1)
			go func() { _, err := svc.GetList("doc"); done <- err }()
			<-entered
			want := context.DeadlineExceeded
			if shutdown {
				require.NoError(t, lifecycle.Shutdown(context.Background()))
				want = context.Canceled
			}
			require.ErrorIs(t, <-done, want)
		})
	}
}

func TestDesktopAttachmentConfigurationAndConcurrentStartup(t *testing.T) {
	_, _, err := NewDesktopAttachmentService(nil, DesktopAttachmentOptions{})
	require.Error(t, err)
	svc, start, err := NewDesktopAttachmentService(&desktopAttachmentClient{}, DesktopAttachmentOptions{OpenFilesDialog: func(context.Context, wailsruntime.OpenDialogOptions) ([]string, error) { return nil, nil }})
	require.NoError(t, err)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start(context.Background())
			_, err := svc.Upload("doc")
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestDesktopAttachmentOpenPaths(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	file := filepath.Join(dir, "report.txt")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0600))
	var calls int
	svc, _, err := NewDesktopAttachmentService(&desktopAttachmentClient{}, DesktopAttachmentOptions{DownloadDir: func() (string, error) { return dir, nil }, Launch: func(name string, args ...string) error {
		calls++
		require.NotEmpty(t, name)
		require.Contains(t, []string{file, dir}, args[len(args)-1])
		return nil
	}})
	require.NoError(t, err)
	require.NoError(t, svc.OpenFile(file))
	require.NoError(t, svc.OpenFolder(file))
	require.Equal(t, 2, calls)
	link := filepath.Join(dir, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	require.Error(t, svc.OpenFile(link))
	require.Error(t, svc.OpenFolder(link))
	require.Equal(t, 2, calls)
}

func TestDesktopAttachmentRejectsTypedNilClient(t *testing.T) {
	var client *desktopAttachmentClient
	_, _, err := NewDesktopAttachmentService(client, DesktopAttachmentOptions{})
	require.Error(t, err)
}

func TestDesktopAttachmentUploadClosesFileOnHTTPError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0600))
	var file *os.File
	client := &desktopAttachmentClient{upload: func(_ context.Context, _, _, _ string, _ int64, body io.Reader) (*dto.Attachment, error) {
		file = body.(*os.File)
		return nil, assert.AnError
	}}
	svc, start, err := NewDesktopAttachmentService(client, DesktopAttachmentOptions{OpenFilesDialog: func(context.Context, wailsruntime.OpenDialogOptions) ([]string, error) { return []string{path}, nil }})
	require.NoError(t, err)
	start(context.Background())
	result, err := svc.Upload("doc")
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "INTERNAL_ERROR", result.Items[0].Error.Code)
	_, err = file.Stat()
	require.ErrorIs(t, err, os.ErrClosed)
}

func requireAppError(t *testing.T, err error, kind string, code int, message string) *models.AppError {
	t.Helper()

	appErr, ok := models.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, kind, appErr.Kind)
	assert.Equal(t, code, appErr.Code)
	if message != "" {
		assert.Contains(t, appErr.Message, message)
	}
	return appErr
}

func TestDesktopAttachmentBatchPreservesOutcomes(t *testing.T) {
	for _, assignment := range []bool{false, true} {
		t.Run(fmt.Sprint(assignment), func(t *testing.T) {
			dir := t.TempDir()
			paths := []string{}
			for _, name := range []string{"first.txt", "second.txt", "third.txt"} {
				path := filepath.Join(dir, name)
				require.NoError(t, os.WriteFile(path, []byte(name), 0600))
				paths = append(paths, path)
			}
			requestID := uuid.NewString()
			var names []string
			client := &desktopAttachmentClient{upload: func(_ context.Context, _, _, name string, _ int64, body io.Reader) (*dto.Attachment, error) {
				names = append(names, name)
				if name == "second.txt" {
					return nil, models.WithRequestID(models.NewBadRequest("файл слишком большой"), requestID)
				}
				return &dto.Attachment{ID: uuid.NewString(), Filename: name}, nil
			}}
			svc, start, err := NewDesktopAttachmentService(client, DesktopAttachmentOptions{OpenFilesDialog: func(context.Context, wailsruntime.OpenDialogOptions) ([]string, error) { return paths, nil }})
			require.NoError(t, err)
			start(context.Background())
			upload := svc.Upload
			if assignment {
				upload = svc.UploadForAssignment
			}
			result, err := upload(uuid.NewString())
			require.NoError(t, err)
			require.Equal(t, []string{"first.txt", "second.txt", "third.txt"}, names)
			require.Len(t, result.Items, 3)
			require.Equal(t, "first.txt", result.Items[0].Attachment.Filename)
			require.Nil(t, result.Items[0].Error)
			require.Nil(t, result.Items[1].Attachment)
			require.Equal(t, "second.txt", result.Items[1].Filename)
			require.Equal(t, "VALIDATION_ERROR", result.Items[1].Error.Code)
			require.Equal(t, requestID, result.Items[1].Error.RequestID)
			require.Equal(t, "third.txt", result.Items[2].Attachment.Filename)
		})
	}
}

func TestAttachmentUploadErrorHidesPrivateDetails(t *testing.T) {
	for _, err := range []error{fmt.Errorf("private local path /home/user/file"), models.NewInternal("private SQL", assert.AnError)} {
		result := attachmentUploadError(err)
		require.Equal(t, "INTERNAL_ERROR", result.Code)
		require.NotContains(t, result.Message, "private")
	}
}

func TestDesktopAttachmentRelocatedDownloads(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "actual-downloads")
	require.NoError(t, os.Mkdir(realDir, 0700))
	root := filepath.Join(base, "Downloads")
	if err := os.Symlink(realDir, root); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	canonicalDir, err := filepath.EvalSymlinks(realDir)
	require.NoError(t, err)
	file := filepath.Join(root, "report.txt")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0600))
	var launched []string
	svc := &AttachmentService{
		downloadDir: func() (string, error) { return root, nil },
		launch:      func(_ string, args ...string) error { launched = append(launched, args[len(args)-1]); return nil },
	}
	require.NoError(t, svc.OpenFile(file))
	require.NoError(t, svc.OpenFolder(file))
	require.NoError(t, svc.OpenFolder(filepath.Join(root, "deleted.txt")))
	// A name starting with two dots is not a parent-directory traversal.
	dotFile := filepath.Join(root, "..notes.txt")
	require.NoError(t, os.WriteFile(dotFile, []byte("data"), 0600))
	require.NoError(t, svc.OpenFile(dotFile))
	require.Equal(t, []string{filepath.Join(canonicalDir, "report.txt"), canonicalDir, canonicalDir, filepath.Join(canonicalDir, "..notes.txt")}, launched)

	outside := filepath.Join(base, "outside")
	require.NoError(t, os.Mkdir(outside, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("data"), 0600))
	escape := filepath.Join(root, "escape")
	require.NoError(t, os.Symlink(outside, escape))
	broken := filepath.Join(root, "broken.txt")
	require.NoError(t, os.Symlink(filepath.Join(outside, "missing.txt"), broken))
	loop := filepath.Join(root, "loop")
	require.NoError(t, os.Symlink(loop, loop))
	for _, path := range []string{
		filepath.Join(escape, "secret.txt"), filepath.Join(escape, "missing.txt"),
		broken, loop, filepath.Join(root, "missing-parent", "file.txt"),
		filepath.Join(root, "..", "outside", "secret.txt"),
	} {
		require.Error(t, svc.OpenFile(path), path)
		require.Error(t, svc.OpenFolder(path), path)
	}
	require.Error(t, svc.OpenFolder(root))
	require.Len(t, launched, 4, "rejected paths must never reach the OS launcher")
}
