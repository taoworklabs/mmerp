// Package attachment keeps the files attached to records of any type. Who may see and
// add them is asked of the record's type through record.Can, every time.
package attachment

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/attachment/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type Service struct{ d Deps }

func NewService(d Deps) *Service {
	s := &Service{d: d}
	d.Record.OnDeleted(s)
	// A file's name may tell what it holds; downloads stay in the audit log only.
	d.Record.RestrictHistory("attachment.added", record.ViewFiles)
	d.Record.RestrictHistory("attachment.removed", record.ViewFiles)
	d.Record.RestrictHistory("attachment.downloaded", "")
	return s
}

// files is the store of attachment, apart from the files of other modules.
func files(ctx context.Context) platform.Files { return platform.FilesFrom(ctx).Sub("attachment") }

// List returns the attachments of a record, oldest first. A record the actor sees
// without its files gives an empty, hidden list.
func (s *Service) List(ctx context.Context, ref record.Ref) (AttachmentList, error) {
	out := AttachmentList{AllowedActions: []string{}, Items: []Attachment{}, MaxMB: maxBytes >> 20, Accept: []string{}}
	for _, t := range slices.Sorted(maps.Keys(accepted)) {
		out.Accept = append(out.Accept, accepted[t]...)
	}
	if err := s.d.Record.Visible(ctx, ref, record.View); err != nil {
		return out, err
	}
	if ok, err := s.d.Record.Can(ctx, ref.Type, ref.ID, record.ViewFiles); err != nil || !ok {
		out.Hidden = true
		return out, err
	}
	open, err := s.d.Record.Open(ctx, ref)
	if err != nil {
		return out, err
	}
	attach := false
	if open {
		if attach, err = s.d.Record.Can(ctx, ref.Type, ref.ID, record.Attach); err != nil {
			return out, err
		}
	}
	if attach {
		out.AllowedActions = append(out.AllowedActions, string(record.Attach))
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListFiles(ctx, store.ListFilesParams{DocType: ref.Type, DocID: ref.ID})
	if err != nil {
		return out, err
	}
	actor, _ := platform.ActorFrom(ctx)
	for _, r := range rows {
		a := Attachment{
			ID: r.ID, Name: r.Name, Size: r.Size, ContentType: r.ContentType,
			UploadedByName: r.UploadedByName, UploadedAt: r.UploadedAt.Time.Format(time.RFC3339), AllowedActions: []string{},
		}
		if open && (attach || r.UploadedBy == actor) {
			a.AllowedActions = append(a.AllowedActions, actionDelete)
		}
		out.Items = append(out.Items, a)
	}
	return out, nil
}

// Add attaches a file to a record. The file is written before its row, so a rollback
// leaves at most a file no row names, which cleanup removes.
func (s *Service) Add(ctx context.Context, ref record.Ref, name string, r io.Reader) error {
	if err := s.canAdd(ctx, ref); err != nil {
		return err
	}
	if name == "" || utf8.RuneCountInString(name) > 255 || !utf8.ValidString(name) {
		return errFileName
	}
	b, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return err
	}
	if len(b) > maxBytes {
		return ErrFileTooLarge
	}
	kind := detect(b)
	if kind == "" {
		return ErrFileType
	}
	fileID, err := files(ctx).Save(bytes.NewReader(b))
	if err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		// Locked, so a draft being deleted or cancelled meanwhile gets no file.
		if err := s.d.Record.LockOpen(ctx, ref); err != nil {
			return err
		}
		actor, _ := platform.ActorFrom(ctx)
		id, err := store.New(platform.DBFrom(ctx)).CreateFile(ctx, store.CreateFileParams{
			DocType: ref.Type, DocID: ref.ID, FileID: fileID, Name: name, Size: int64(len(b)), ContentType: kind, UploadedBy: actor,
		})
		if err != nil {
			return err
		}
		return s.d.Audit.RecordFor(ctx, "attachment.added", audit.Ref(ref), map[string]any{"id": id, "name": name})
	})
}

// canAdd fails before the upload is kept: the product gate's own error first, then not
// found for a record whose files the actor may not see.
func (s *Service) canAdd(ctx context.Context, ref record.Ref) error {
	if err := s.d.Record.WriteGate(ctx, ref); err != nil {
		return err
	}
	if err := s.d.Record.Visible(ctx, ref, record.ViewFiles); err != nil {
		return err
	}
	if ok, err := s.d.Record.Can(ctx, ref.Type, ref.ID, record.Attach); err != nil || !ok {
		return platform.OrErr(err, platform.ErrForbidden)
	}
	if open, err := s.d.Record.Open(ctx, ref); err != nil || !open {
		return platform.OrErr(err, record.ErrNotEditable)
	}
	return nil
}

// Remove deletes an attachment: its uploader may, and so may whoever holds attach. Only
// the row goes; cleanup removes the file later, so a rollback never loses a file a row names.
func (s *Service) Remove(ctx context.Context, id int64) error {
	row, err := s.file(ctx, id)
	if err != nil {
		return err
	}
	ref := record.Ref{Type: row.DocType, ID: row.DocID}
	if err := s.d.Record.WriteGate(ctx, ref); err != nil {
		return err
	}
	if actor, _ := platform.ActorFrom(ctx); row.UploadedBy != actor {
		if ok, err := s.d.Record.Can(ctx, ref.Type, ref.ID, record.Attach); err != nil || !ok {
			return platform.OrErr(err, platform.ErrForbidden)
		}
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.d.Record.LockOpen(ctx, ref); err != nil {
			return err
		}
		// Gone already: removed meanwhile by someone else.
		if _, err := store.New(platform.DBFrom(ctx)).DeleteFile(ctx, id); errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		} else if err != nil {
			return err
		}
		return s.d.Audit.RecordFor(ctx, "attachment.removed", audit.Ref(ref), map[string]any{"id": id, "name": row.Name})
	})
}

// Open reads an attachment the actor may see now; any other does not exist. Every
// download is audited.
func (s *Service) Open(ctx context.Context, id int64) (Attachment, *os.File, error) {
	row, err := s.file(ctx, id)
	if err != nil {
		return Attachment{}, nil, err
	}
	f, err := files(ctx).Open(row.FileID)
	if os.IsNotExist(err) {
		return Attachment{}, nil, platform.ErrNotFound
	}
	if err != nil {
		return Attachment{}, nil, err
	}
	ref := record.Ref{Type: row.DocType, ID: row.DocID}
	if err := s.d.Audit.RecordFor(ctx, "attachment.downloaded", audit.Ref(ref), map[string]any{"id": id}); err != nil {
		_ = f.Close()
		return Attachment{}, nil, err
	}
	return Attachment{ID: row.ID, Name: row.Name, Size: row.Size, ContentType: row.ContentType}, f, nil
}

// file reads an attachment's row if the actor may see its record's files now; any other
// does not exist, even with a guessed id.
func (s *Service) file(ctx context.Context, id int64) (store.AttachmentFile, error) {
	row, err := store.New(platform.DBFrom(ctx)).GetFile(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return row, platform.ErrNotFound
	}
	if err != nil {
		return row, err
	}
	return row, s.d.Record.Visible(ctx, record.Ref{Type: row.DocType, ID: row.DocID}, record.ViewFiles)
}

// Deleted forgets the attachments of a deleted draft, in its transaction; cleanup
// removes the files later.
func (s *Service) Deleted(ctx context.Context, ref record.Ref) error {
	return store.New(platform.DBFrom(ctx)).DeleteRecordFiles(ctx, store.DeleteRecordFilesParams{DocType: ref.Type, DocID: ref.ID})
}

// detect names the type of a file from its content, or "" for a type not accepted.
// The name and the browser's header are never trusted.
func detect(b []byte) string {
	t := http.DetectContentType(b)
	if t == "application/zip" {
		z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			return ""
		}
		for _, f := range z.File {
			switch f.Name {
			case "word/document.xml":
				return docx
			case "xl/workbook.xml":
				return xlsx
			}
		}
	}
	if _, ok := accepted[t]; !ok {
		return ""
	}
	return t
}
