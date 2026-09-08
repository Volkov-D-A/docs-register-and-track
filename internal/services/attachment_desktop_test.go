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

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

func TestSafeDownloadFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "keeps simple filename", filename: "report.pdf", want: "report.pdf"},
		{name: "trims spaces", filename: "  report.pdf  ", want: "report.pdf"},
		{name: "drops parent directories", filename: "../secret/report.pdf", want: "report.pdf"},
		{name: "normalizes windows path", filename: `..\\secret\\report.pdf`, want: "report.pdf"},
		{name: "drops control characters", filename: "report\n.pdf", want: "report.pdf"},
		{name: "empty fallback", filename: "   ", want: "attachment"},
		{name: "dot fallback", filename: ".", want: "attachment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, safeDownloadFilename(tt.filename))
		})
	}
}

func TestAttachmentService_ValidatePathInDownloads(t *testing.T) {
	// Проверка пути доступа к файлу для защиты от уязвимости Path Traversal
	t.Run("valid path", func(t *testing.T) {
		dir := t.TempDir()
		svc := &AttachmentService{downloadDir: func() (string, error) { return dir, nil }}
		downloadDir, _ := svc.getDownloadDir()
		err := svc.validatePathInDownloads(downloadDir + "/test.pdf")
		require.NoError(t, err)
	})

	t.Run("path traversal attack", func(t *testing.T) {
		dir := t.TempDir()
		svc := &AttachmentService{downloadDir: func() (string, error) { return dir, nil }}
		err := svc.validatePathInDownloads("C:\\Windows\\System32\\..\\..\\test.pdf")
		require.Error(t, err)
		requireAppError(t, err, "FORBIDDEN", 403, "папке загрузок")
	})

	t.Run("outside downloads", func(t *testing.T) {
		dir := t.TempDir()
		svc := &AttachmentService{downloadDir: func() (string, error) { return dir, nil }}
		err := svc.validatePathInDownloads("C:\\Windows\\System32\\cmd.exe")
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
		data, err := os.ReadFile("../../frontend/wailsjs/go/services/AttachmentService." + ext)
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
			var items []dto.Attachment
			if assignment {
				items, err = svc.UploadForAssignment(id)
			} else {
				items, err = svc.Upload(id)
			}
			require.NoError(t, err)
			require.Len(t, items, 1)
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
			for _, upload := range []func(string) ([]dto.Attachment, error){svc.Upload, svc.UploadForAssignment} {
				items, err := upload(uuid.NewString())
				if tc.want == "" {
					require.NoError(t, err)
					require.Empty(t, items)
				} else {
					require.ErrorContains(t, err, tc.want)
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
	_, err = svc.Upload("doc")
	require.ErrorIs(t, err, assert.AnError)
	_, err = file.Stat()
	require.ErrorIs(t, err, os.ErrClosed)
}

func TestDesktopAttachmentMetricsFromConstructor(t *testing.T) {
	metrics := observability.NewRegistry(10)
	client := &desktopAttachmentClient{call: func(context.Context, string) error { return assert.AnError }}
	svc, _, err := NewDesktopAttachmentService(client, DesktopAttachmentOptions{Metrics: metrics})
	require.NoError(t, err)
	_, err = svc.GetList("doc")
	require.ErrorIs(t, err, assert.AnError)
	snapshots := metrics.Snapshot()
	require.Len(t, snapshots, 1)
	require.Equal(t, "attachments.get_list", snapshots[0].Name)
	require.Equal(t, int64(1), snapshots[0].Errors)
}
