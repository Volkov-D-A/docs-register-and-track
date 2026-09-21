package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestAttachmentDownloadIgnoresReducedUploadLimit(t *testing.T) {
	svc, _, settings, storage, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
	// An existing 2 MiB file remains downloadable after reducing uploads to 1 MiB.
	settings.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Value: "1"}, nil).Maybe()
	content := bytes.Repeat([]byte("x"), 2*1024*1024)
	attachment := &models.Attachment{StoragePath: "existing.pdf", FileSize: int64(len(content))}
	storage.On("DownloadFileToWriter", mock.Anything, attachment.StoragePath, mock.Anything, attachment.FileSize).
		Run(func(args mock.Arguments) {
			_, err := args.Get(2).(io.Writer).Write(content)
			require.NoError(t, err)
		}).Return(nil).Once()
	var output bytes.Buffer

	require.NoError(t, svc.StreamAttachment(context.Background(), attachment, &output))
	require.Equal(t, content, output.Bytes())
	settings.AssertNotCalled(t, "Get", "max_file_size_mb")
}

func TestAttachmentDownloadRejectsInvalidStoredSizeBeforeStorageAccess(t *testing.T) {
	for _, size := range []int64{-1, int64(MaximumAttachmentSizeMB)*1024*1024 + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			svc, repo, _, storage, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
			attachment := &models.Attachment{ID: uuid.New(), DocumentID: uuid.New(), FileSize: size, StoragePath: "invalid.pdf"}
			repo.On("GetByID", attachment.ID).Return(attachment, nil).Once()

			item, err := svc.AuthorizeDownload(attachment.ID.String())
			require.Error(t, err)
			require.Nil(t, item)
			require.Error(t, svc.StreamAttachment(context.Background(), attachment, io.Discard))
			storage.AssertNotCalled(t, "DownloadFileToWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestAttachmentDownloadChecksStoredLength(t *testing.T) {
	for _, tc := range []struct {
		name  string
		size  int64
		body  string
		valid bool
	}{
		{"empty", 0, "", true},
		{"exact", 3, "abc", true},
		{"shorter than metadata", 4, "abc", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _, storage, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
			attachment := &models.Attachment{StoragePath: "file.pdf", FileSize: tc.size}
			storage.On("DownloadFileToWriter", mock.Anything, attachment.StoragePath, mock.Anything, tc.size).
				Run(func(args mock.Arguments) {
					_, err := io.WriteString(args.Get(2).(io.Writer), tc.body)
					require.NoError(t, err)
				}).Return(nil).Once()
			err := svc.StreamAttachment(context.Background(), attachment, io.Discard)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "does not match stored metadata")
			}
		})
	}
}
